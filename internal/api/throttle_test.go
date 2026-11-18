package api

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/charlintosh/lazyado/internal/config"
)

type fakeClock struct {
	mu    sync.Mutex
	now   time.Time
	slept []time.Duration
}

func (f *fakeClock) Now() time.Time {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.now
}

func (f *fakeClock) Sleep(ctx context.Context, d time.Duration) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.slept = append(f.slept, d)
	f.now = f.now.Add(d)
	return nil
}

func (f *fakeClock) Slept() []time.Duration {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]time.Duration(nil), f.slept...)
}

type reply struct {
	status int
	header http.Header
	body   string
}

func scriptedClient(t *testing.T, replies ...reply) (*Client, *fakeClock, *atomic.Int32) {
	t.Helper()
	c := NewClient(&config.Config{Organization: "contoso", Project: "web", Team: "web team", PAT: "test-pat-not-real"})
	clock := &fakeClock{now: time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)}
	c.gate.now, c.gate.sleep = clock.Now, clock.Sleep
	var calls atomic.Int32
	c.httpClient.Transport = roundTrip(func(r *http.Request) (*http.Response, error) {
		i := int(calls.Add(1)) - 1
		rep := replies[min(i, len(replies)-1)]
		h := rep.header
		if h == nil {
			h = http.Header{}
		}
		return &http.Response{StatusCode: rep.status, Header: h, Body: io.NopCloser(strings.NewReader(rep.body))}, nil
	})
	return c, clock, &calls
}

func headers(kv ...string) http.Header {
	h := http.Header{}
	for i := 0; i+1 < len(kv); i += 2 {
		h.Set(kv[i], kv[i+1])
	}
	return h
}

func TestBlockedGetIsRetriedAfterRetryAfter(t *testing.T) {
	c, clock, calls := scriptedClient(t,
		reply{status: http.StatusTooManyRequests, header: headers("Retry-After", "7"), body: `{"message":"TF400733"}`},
		reply{status: http.StatusOK, body: `{"value":[]}`},
	)
	resp, err := c.get("/git/repositories")
	if err != nil {
		t.Fatalf("a blocked GET must be retried, got %v", err)
	}
	resp.Body.Close()
	if calls.Load() != 2 {
		t.Fatalf("sent %d requests, want the original and one retry", calls.Load())
	}
	if got := clock.Slept(); len(got) != 1 || got[0] != 7*time.Second {
		t.Fatalf("waited %v, want exactly the 7s Retry-After", got)
	}
}

func TestBlockedGetGivesUpWithAClearMessage(t *testing.T) {
	c, _, calls := scriptedClient(t, reply{status: http.StatusTooManyRequests, body: `{"message":"TF400733: canceled"}`})
	_, err := c.get("/git/repositories")
	if err == nil || !strings.Contains(err.Error(), "rate limiting") {
		t.Fatalf("got %v, want a rate limit explanation", err)
	}
	if calls.Load() != maxRetries+1 {
		t.Fatalf("sent %d requests, want %d", calls.Load(), maxRetries+1)
	}
}

func TestWritesAreNeverRetried(t *testing.T) {
	c, _, calls := scriptedClient(t, reply{status: http.StatusTooManyRequests, body: `{}`})
	if _, err := c.post("/wit/wiql", strings.NewReader(`{}`)); err == nil {
		t.Fatal("a blocked POST must fail")
	}
	if calls.Load() != 1 {
		t.Fatalf("sent %d requests, a write must be sent once", calls.Load())
	}
}

func TestRetryAfterOnSuccessPausesTheNextRequest(t *testing.T) {
	c, clock, _ := scriptedClient(t,
		reply{status: http.StatusOK, header: headers("Retry-After", "3"), body: `{}`},
		reply{status: http.StatusOK, body: `{}`},
	)
	for range 2 {
		resp, err := c.get("/wit/workitemtypes")
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
	}
	if got := clock.Slept(); len(got) != 1 || got[0] != 3*time.Second {
		t.Fatalf("waited %v, want one 3s pause before the second request", got)
	}
}

func TestRetryAfterIsCapped(t *testing.T) {
	c, clock, _ := scriptedClient(t,
		reply{status: http.StatusOK, header: headers("Retry-After", "3600"), body: `{}`},
		reply{status: http.StatusOK, body: `{}`},
	)
	for range 2 {
		resp, err := c.get("/wit/workitemtypes")
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
	}
	if got := clock.Slept(); len(got) != 1 || got[0] != maxRetryWait {
		t.Fatalf("waited %v, want the %v cap", got, maxRetryWait)
	}
}

func TestThrottleReportsEpisodes(t *testing.T) {
	c, _, _ := scriptedClient(t,
		reply{status: http.StatusOK, header: headers("X-RateLimit-Remaining", "150", "X-RateLimit-Limit", "200"), body: `{}`},
		reply{status: http.StatusOK, header: headers("X-RateLimit-Remaining", "20", "X-RateLimit-Limit", "200", "X-RateLimit-Resource", "Core"), body: `{}`},
		reply{status: http.StatusOK, header: headers("X-RateLimit-Delay", "0.250"), body: `{}`},
		reply{status: http.StatusOK, body: `{}`},
		reply{status: http.StatusOK, header: headers("X-RateLimit-Delay", "1"), body: `{}`},
	)
	want := []Throttle{
		{Active: false, Remaining: 150, Limit: 200},
		{Active: true, Remaining: 20, Limit: 200, Resource: "Core", Episode: 1},
		{Active: true, Remaining: 20, Limit: 200, Resource: "Core", Episode: 1},
		{Active: false, Remaining: 20, Limit: 200, Resource: "Core", Episode: 1},
		{Active: true, Remaining: 20, Limit: 200, Resource: "Core", Episode: 2},
	}
	for i, w := range want {
		resp, err := c.get("/wit/workitemtypes")
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if got := c.Throttle(); got != w {
			t.Fatalf("after reply %d throttle is %+v, want %+v", i, got, w)
		}
	}
}

func TestUsageAddsUpRequestCost(t *testing.T) {
	c, _, _ := scriptedClient(t, reply{status: http.StatusOK, header: headers("X-RateLimit-Cost", "0.25"), body: `{}`})
	for range 4 {
		resp, err := c.get("/wit/workitemtypes")
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
	}
	if u := c.Usage(); u.Requests != 4 || u.Cost != 1 {
		t.Fatalf("usage %+v, want 4 requests costing 1 TSTU", u)
	}
}

func TestRequestsInFlightAreCapped(t *testing.T) {
	c := NewClient(&config.Config{Organization: "contoso", Project: "web", PAT: "test-pat-not-real"})
	var inFlight, peak atomic.Int32
	release := make(chan struct{})
	c.httpClient.Transport = roundTrip(func(*http.Request) (*http.Response, error) {
		n := inFlight.Add(1)
		for {
			p := peak.Load()
			if n <= p || peak.CompareAndSwap(p, n) {
				break
			}
		}
		<-release
		inFlight.Add(-1)
		return &http.Response{StatusCode: http.StatusOK, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(`{}`))}, nil
	})
	var wg sync.WaitGroup
	for range maxInFlight * 3 {
		wg.Go(func() {
			if resp, err := c.get("/build/builds/1/timeline"); err == nil {
				resp.Body.Close()
			}
		})
	}
	deadline := time.Now().Add(2 * time.Second)
	for inFlight.Load() < maxInFlight && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	close(release)
	wg.Wait()
	if p := peak.Load(); p != maxInFlight {
		t.Fatalf("peak of %d requests in flight, want %d", p, maxInFlight)
	}
}

func TestCanceledContextStopsAQueuedRequest(t *testing.T) {
	c, _, calls := scriptedClient(t, reply{status: http.StatusOK, body: `{}`})
	for range maxInFlight {
		c.gate.slots <- struct{}{}
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		_, err := c.WithContext(ctx).get("/git/repositories")
		done <- err
	}()
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("got %v, want context.Canceled", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("a canceled request kept waiting for a slot")
	}
	if calls.Load() != 0 {
		t.Fatal("a canceled request must never be sent")
	}
}

func TestRetryAfterParsesSecondsAndDates(t *testing.T) {
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	cases := map[string]time.Duration{
		"":                              0,
		"12":                            12 * time.Second,
		"0.5":                           500 * time.Millisecond,
		"Fri, 02 Jan 2026 03:04:15 GMT": 10 * time.Second,
		"Fri, 02 Jan 2026 03:00:00 GMT": 0,
		"soon":                          0,
	}
	for in, want := range cases {
		if got := retryAfter(in, now); got != want {
			t.Errorf("retryAfter(%q) = %v, want %v", in, got, want)
		}
	}
}

func TestVersionedAddsTheAPIVersionOnce(t *testing.T) {
	cases := map[string]string{
		"/wit/wiql":                       "https://x/_apis/wit/wiql?api-version=7.1",
		"wit/wiql":                        "https://x/_apis/wit/wiql?api-version=7.1",
		"/git/refs?filter=heads/&$top=10": "https://x/_apis/git/refs?filter=heads/&$top=10&api-version=7.1",
	}
	for endpoint, want := range cases {
		if got := versioned("https://x/_apis", endpoint, apiVersion); got != want {
			t.Errorf("versioned(%q) = %q, want %q", endpoint, got, want)
		}
	}
}
