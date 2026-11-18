package api

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/charlintosh/lazyado/internal/debug"
)

var throttleLogger = debug.Scope("throttle")

const (
	maxInFlight      = 6
	maxRetries       = 2
	defaultRetryWait = 5 * time.Second
	maxRetryWait     = 30 * time.Second
	lowBudgetRatio   = 0.2
)

type Throttle struct {
	Active    bool
	Remaining float64
	Limit     float64
	Resource  string
	Episode   int
}

type Usage struct {
	Requests int
	Cost     float64
}

type rateHeaders struct {
	remaining  float64
	limit      float64
	delay      float64
	cost       float64
	retryAfter time.Duration
	resource   string
	known      bool
}

type gate struct {
	slots chan struct{}
	now   func() time.Time
	sleep func(context.Context, time.Duration) error

	mu         sync.Mutex
	pauseUntil time.Time
	state      Throttle
	usage      Usage
}

func newGate() *gate {
	return &gate{slots: make(chan struct{}, maxInFlight), now: time.Now, sleep: sleepCtx}
}

func sleepCtx(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-t.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (g *gate) acquire(ctx context.Context) error {
	select {
	case g.slots <- struct{}{}:
	case <-ctx.Done():
		return ctx.Err()
	}
	if wait := g.pauseLeft(); wait > 0 {
		throttleLogger.Debugf("pausing %v before the next request", wait.Round(time.Millisecond))
		if err := g.sleep(ctx, wait); err != nil {
			g.release()
			return err
		}
	}
	return nil
}

func (g *gate) release() {
	<-g.slots
}

func (g *gate) pauseLeft() time.Duration {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.pauseUntil.Sub(g.now())
}

func (g *gate) observe(status int, h http.Header) rateHeaders {
	r := parseRateHeaders(h)
	g.mu.Lock()
	defer g.mu.Unlock()
	g.usage.Requests++
	g.usage.Cost += r.cost
	if r.retryAfter > 0 {
		if until := g.now().Add(min(r.retryAfter, maxRetryWait)); until.After(g.pauseUntil) {
			g.pauseUntil = until
		}
	}
	active := r.delay > 0 || r.retryAfter > 0 || status == http.StatusTooManyRequests ||
		(r.limit > 0 && r.remaining/r.limit < lowBudgetRatio)
	if active && !g.state.Active {
		g.state.Episode++
	}
	g.state.Active = active
	if r.known {
		g.state.Remaining, g.state.Limit, g.state.Resource = r.remaining, r.limit, r.resource
	}
	return r
}

func (g *gate) throttle() Throttle {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.state
}

func (g *gate) totals() Usage {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.usage
}

func parseRateHeaders(h http.Header) rateHeaders {
	r := rateHeaders{resource: h.Get("X-RateLimit-Resource")}
	r.remaining, r.known = headerFloat(h, "X-RateLimit-Remaining")
	r.limit, _ = headerFloat(h, "X-RateLimit-Limit")
	r.delay, _ = headerFloat(h, "X-RateLimit-Delay")
	r.cost, _ = headerFloat(h, "X-RateLimit-Cost")
	r.retryAfter = retryAfter(h.Get("Retry-After"), time.Now())
	return r
}

func headerFloat(h http.Header, name string) (float64, bool) {
	v := strings.TrimSpace(h.Get(name))
	if v == "" {
		return 0, false
	}
	f, err := strconv.ParseFloat(v, 64)
	return f, err == nil
}

func retryAfter(v string, now time.Time) time.Duration {
	v = strings.TrimSpace(v)
	if v == "" {
		return 0
	}
	if secs, err := strconv.ParseFloat(v, 64); err == nil {
		return max(time.Duration(secs*float64(time.Second)), 0)
	}
	if at, err := http.ParseTime(v); err == nil {
		return max(at.Sub(now), 0)
	}
	return 0
}

func retryable(method string, status int) bool {
	return method == http.MethodGet && (status == http.StatusTooManyRequests || status == http.StatusServiceUnavailable)
}

func retryWait(r rateHeaders, attempt int) time.Duration {
	if r.retryAfter > 0 {
		return min(r.retryAfter, maxRetryWait)
	}
	return min(defaultRetryWait*time.Duration(attempt+1), maxRetryWait)
}
