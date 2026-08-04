package natsx

import (
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
)

// ErrNotReady is returned by JS() while the background connection is still
// being established.
var ErrNotReady = errors.New("nats jetstream not ready")

// JSProvider yields a JetStream context. Publishers depend on this instead of a
// concrete context so a service can start serving HTTP before NATS is up.
type JSProvider interface {
	JS() (nats.JetStreamContext, error)
}

type staticProvider struct{ js nats.JetStreamContext }

// Static wraps an already-connected JetStream context (tests, tooling).
func Static(js nats.JetStreamContext) JSProvider { return staticProvider{js: js} }

func (s staticProvider) JS() (nats.JetStreamContext, error) {
	if s.js == nil {
		return nil, ErrNotReady
	}
	return s.js, nil
}

// Background connects to NATS JetStream off the critical path.
//
// Services used to block on ConnectJS/EnsureStreams in main() before mounting
// the HTTP server, so a broker that was slow or briefly unreachable meant
// nothing ever answered /healthz and Docker marked the container unhealthy —
// which then failed the whole `docker compose up` through depends_on.
// Now the process serves immediately, retries the broker forever in the
// background, and reports broker state on /readyz.
type Background struct {
	url string

	mu      sync.RWMutex
	nc      *nats.Conn
	js      nats.JetStreamContext
	lastErr error

	startOnce sync.Once
	closeOnce sync.Once
	done      chan struct{}
}

// NewBackground returns a not-yet-connected provider for url.
func NewBackground(url string) *Background {
	if url == "" {
		url = nats.DefaultURL
	}
	return &Background{url: url, done: make(chan struct{})}
}

// Start kicks off the connect loop and returns immediately. onReady, when set,
// runs once with a live JetStream context (e.g. to subscribe consumers); an
// error from onReady makes the loop reconnect and retry.
func (b *Background) Start(onReady func(nats.JetStreamContext) error) {
	b.startOnce.Do(func() { go b.loop(onReady) })
}

func (b *Background) loop(onReady func(nats.JetStreamContext) error) {
	backoff := time.Second
	for attempt := 1; ; attempt++ {
		select {
		case <-b.done:
			return
		default:
		}

		nc, js, err := ConnectJS(b.url)
		if err == nil {
			if err = EnsureStreams(js); err != nil {
				nc.Close()
			}
		}
		if err == nil && onReady != nil {
			if err = onReady(js); err != nil {
				nc.Close()
			}
		}
		if err == nil {
			b.mu.Lock()
			b.nc, b.js, b.lastErr = nc, js, nil
			b.mu.Unlock()
			slog.Info("nats ready", "url", b.url, "attempt", attempt)
			return
		}

		b.mu.Lock()
		b.lastErr = err
		b.mu.Unlock()
		slog.Warn("nats not ready; retrying in background",
			"url", b.url, "attempt", attempt, "err", err, "retry_in", backoff)

		select {
		case <-b.done:
			return
		case <-time.After(backoff):
		}
		if backoff < 30*time.Second {
			backoff *= 2
		}
	}
}

// JS returns the JetStream context, or ErrNotReady while connecting.
func (b *Background) JS() (nats.JetStreamContext, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.js == nil {
		if b.lastErr != nil {
			return nil, b.lastErr
		}
		return nil, ErrNotReady
	}
	return b.js, nil
}

// Ready reports whether JetStream is usable right now.
func (b *Background) Ready() bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.js != nil
}

// Err returns the last connect error, or nil when connected.
func (b *Background) Err() error {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.js != nil {
		return nil
	}
	return b.lastErr
}

// Close stops the retry loop and drops the connection.
func (b *Background) Close() {
	b.closeOnce.Do(func() { close(b.done) })
	b.mu.Lock()
	nc := b.nc
	b.nc, b.js = nil, nil
	b.mu.Unlock()
	if nc != nil {
		nc.Close()
	}
}

// ReadyCheck adapts the provider to httpx readiness checks.
func (b *Background) ReadyCheck() error { return b.Err() }
