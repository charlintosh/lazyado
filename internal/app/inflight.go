package ui

import (
	"context"
	"errors"

	"github.com/charlintosh/lazyado/internal/api"
)

type inflight struct {
	ctx    context.Context
	cancel context.CancelFunc
}

func (f *inflight) restart(c *api.Client) *api.Client {
	if f.cancel != nil {
		f.cancel()
	}
	f.ctx, f.cancel = context.WithCancel(context.Background())
	return c.WithContext(f.ctx)
}

func (f *inflight) stop() {
	if f.cancel != nil {
		f.cancel()
	}
	f.ctx, f.cancel = nil, nil
}

func (f *inflight) client(c *api.Client) *api.Client {
	if f.ctx == nil {
		return c
	}
	return c.WithContext(f.ctx)
}

func canceled(err error) bool {
	return errors.Is(err, context.Canceled)
}
