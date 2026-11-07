package ggsel

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/shopspring/decimal"
)

func TestRetryOnlySafeGET(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		method  string
		retry   bool
		status  int
		retries int
		hits    int
	}{
		{name: "get 503", method: http.MethodGet, retry: true, status: http.StatusServiceUnavailable, retries: 0, hits: 3},
		{name: "get 502", method: http.MethodGet, retry: true, status: http.StatusBadGateway, retries: 0, hits: 3},
		{name: "get 504", method: http.MethodGet, retry: true, status: http.StatusGatewayTimeout, retries: 0, hits: 3},
		{name: "get 429", method: http.MethodGet, retry: true, status: http.StatusTooManyRequests, retries: 0, hits: 3},
		{name: "get 500", method: http.MethodGet, retry: true, status: http.StatusInternalServerError, retries: 0, hits: 1},
		{name: "get 401", method: http.MethodGet, retry: true, status: http.StatusUnauthorized, retries: 0, hits: 1},
		{name: "get 400", method: http.MethodGet, retry: true, status: http.StatusBadRequest, retries: 0, hits: 1},
		{name: "get 404", method: http.MethodGet, retry: true, status: http.StatusNotFound, retries: 0, hits: 1},
		{name: "get 422", method: http.MethodGet, retry: true, status: http.StatusUnprocessableEntity, retries: 0, hits: 1},
		{name: "get without flag", method: http.MethodGet, retry: false, status: http.StatusServiceUnavailable, retries: 0, hits: 1},
		{name: "post 503", method: http.MethodPost, retry: true, status: http.StatusServiceUnavailable, retries: 0, hits: 1},
		{name: "retries disabled", method: http.MethodGet, retry: true, status: http.StatusServiceUnavailable, retries: -1, hits: 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var hits atomic.Int32
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				hits.Add(1)
				w.WriteHeader(tt.status)
				_, _ = io.WriteString(w, `{"errors":[{"code":"NOPE"}]}`)
			}))
			t.Cleanup(srv.Close)
			c := newTestClient(t, srv.URL, Options{APIKey: "k", MaxRetries: tt.retries})
			err := c.do(context.Background(), call{
				method: tt.method,
				path:   "/resource",
				auth:   authV2,
				retry:  tt.retry,
			})
			if err == nil {
				t.Fatal("expected error")
			}
			var apiErr *APIError
			if !errors.As(err, &apiErr) || apiErr.StatusCode != tt.status {
				t.Fatalf("error = %v", err)
			}
			if int(hits.Load()) != tt.hits {
				t.Fatalf("hits = %d, want %d", hits.Load(), tt.hits)
			}
		})
	}
}

func TestRetryRecovers(t *testing.T) {
	t.Parallel()
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := hits.Add(1)
		if n < 3 {
			w.Header().Set("Retry-After", "7")
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"price":19.99,"extra":1}`)
	}))
	t.Cleanup(srv.Close)

	var waits []time.Duration
	c := newTestClient(t, srv.URL, Options{
		APIKey:     "k",
		HTTPClient: &http.Client{Timeout: time.Minute},
	})
	c.sleep = func(_ context.Context, d time.Duration) error {
		waits = append(waits, d)
		return nil
	}
	var out struct {
		Price Decimal `json:"price"`
	}
	err := c.do(context.Background(), call{
		method: http.MethodGet,
		path:   "/offers",
		dest:   &out,
		auth:   authV2,
		retry:  true,
	})
	if err != nil {
		t.Fatalf("do: %v", err)
	}
	if hits.Load() != 3 {
		t.Fatalf("hits = %d", hits.Load())
	}
	if len(waits) != 2 || waits[0] != 7*time.Second || waits[1] != 7*time.Second {
		t.Fatalf("waits = %v", waits)
	}
	want, err := decimal.NewFromString("19.99")
	if err != nil {
		t.Fatal(err)
	}
	if !out.Price.Equal(decimalOf(want)) {
		t.Fatalf("price = %s", out.Price)
	}
}

func TestRetryAfterIsCapped(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	tests := []struct {
		name   string
		header string
		want   time.Duration
	}{
		{name: "seconds", header: "120", want: maxRetryAfter},
		{name: "http date", header: now.Add(10 * time.Second).UTC().Format(http.TimeFormat), want: 10 * time.Second},
		{name: "past date", header: now.Add(-5 * time.Second).UTC().Format(http.TimeFormat), want: 0},
		{name: "negative", header: "-1", want: 100 * time.Millisecond},
		{name: "absent", header: "", want: 100 * time.Millisecond},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var got time.Duration
			var hits atomic.Int32
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				n := hits.Add(1)
				if n == 1 {
					if tt.header != "" {
						w.Header().Set("Retry-After", tt.header)
					}
					w.WriteHeader(http.StatusBadGateway)
					return
				}
				w.WriteHeader(http.StatusOK)
			}))
			t.Cleanup(srv.Close)
			c := newTestClient(t, srv.URL, Options{
				APIKey:          "k",
				RetryBackoffMin: 100 * time.Millisecond,
				RetryBackoffMax: 2 * time.Second,
				Clock:           func() time.Time { return now },
				HTTPClient:      &http.Client{Timeout: time.Minute},
			})
			c.sleep = func(_ context.Context, d time.Duration) error {
				got = d
				return nil
			}
			if err := c.do(context.Background(), call{
				method: http.MethodGet,
				path:   "/offers",
				auth:   authV2,
				retry:  true,
			}); err != nil {
				t.Fatalf("do: %v", err)
			}
			if got != tt.want {
				t.Fatalf("wait = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestRetrySleepHonorsCancel(t *testing.T) {
	t.Parallel()
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	t.Cleanup(srv.Close)
	c := newTestClient(t, srv.URL, Options{APIKey: "k"})
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	c.sleep = func(ctx context.Context, _ time.Duration) error {
		cancel()
		return ctx.Err()
	}
	err := c.do(ctx, call{
		method: http.MethodGet,
		path:   "/offers",
		auth:   authV2,
		retry:  true,
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v", err)
	}
	if hits.Load() != 1 {
		t.Fatalf("hits = %d", hits.Load())
	}
}

func TestRetryNetError(t *testing.T) {
	t.Parallel()
	var hits atomic.Int32
	c := newTestClient(t, "", Options{
		APIKey:  "k",
		BaseURL: "https://seller.ggsel.com",
		HTTPClient: &http.Client{
			Timeout: 5 * time.Second,
			Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				n := hits.Add(1)
				if n < 3 {
					return nil, tempNetError("connection reset")
				}
				payload := `{"ok":true}`
				return &http.Response{
					StatusCode:    http.StatusOK,
					Status:        "200 OK",
					Proto:         "HTTP/1.1",
					ProtoMajor:    1,
					ProtoMinor:    1,
					Header:        make(http.Header),
					Body:          io.NopCloser(strings.NewReader(payload)),
					ContentLength: int64(len(payload)),
					Request:       r,
				}, nil
			}),
		},
	})
	err := c.do(context.Background(), call{
		method: http.MethodGet,
		path:   "/offers",
		dest:   &struct{}{},
		auth:   authV2,
		retry:  true,
	})
	if err != nil {
		t.Fatalf("do: %v", err)
	}
	if hits.Load() != 3 {
		t.Fatalf("hits = %d", hits.Load())
	}
}

func TestRetrySkipsCanceledNetError(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		err  error
	}{
		{name: "canceled", err: ctxNetError{context.Canceled}},
		{name: "deadline", err: ctxNetError{context.DeadlineExceeded}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var hits atomic.Int32
			c := newTestClient(t, "", Options{
				APIKey:  "k",
				BaseURL: "https://seller.ggsel.com",
				HTTPClient: &http.Client{
					Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
						hits.Add(1)
						return nil, tt.err
					}),
				},
			})
			err := c.do(context.Background(), call{
				method: http.MethodGet,
				path:   "/offers",
				auth:   authV2,
				retry:  true,
			})
			if !errors.Is(err, tt.err) && !errors.Is(err, errors.Unwrap(tt.err)) {
				t.Fatalf("error = %v", err)
			}
			if hits.Load() != 1 {
				t.Fatalf("hits = %d", hits.Load())
			}
		})
	}
}

func TestBackoffStaysInsideBounds(t *testing.T) {
	t.Parallel()
	rt := &retryTripper{min: 100 * time.Millisecond, maxWait: 2 * time.Second}
	if got := rt.backoff(0); got != 100*time.Millisecond {
		t.Fatalf("attempt 0 = %s", got)
	}
	got := rt.backoff(1)
	if got < 100*time.Millisecond || got > 200*time.Millisecond {
		t.Fatalf("attempt 1 = %s", got)
	}
	got = rt.backoff(10)
	if got < time.Second || got > 2*time.Second {
		t.Fatalf("attempt 10 = %s", got)
	}
}

func TestSleepContext(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := sleepContext(ctx, time.Hour); !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v", err)
	}
	if err := sleepContext(context.Background(), 0); err != nil {
		t.Fatalf("zero sleep: %v", err)
	}
}

func TestRetryAfterReturnsStatusWhenPauseExceedsDeadline(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "60")
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = io.WriteString(w, `{"errors":[{"code":"BUSY"}]}`)
	}))
	t.Cleanup(srv.Close)
	c := newTestClient(t, srv.URL, Options{
		APIKey: "k",
		HTTPClient: &http.Client{
			Timeout: 200 * time.Millisecond,
		},
	})
	c.sleep = sleepContext
	_, err := c.ListOffers(context.Background(), ListOffersParams{})
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("error = %v", err)
	}
	if errors.Is(err, context.DeadlineExceeded) {
		t.Fatal("deadline replaced the API status")
	}
}

type tempNetError string

func (e tempNetError) Error() string   { return string(e) }
func (e tempNetError) Timeout() bool   { return false }
func (e tempNetError) Temporary() bool { return true }

type ctxNetError struct{ error }

func (e ctxNetError) Unwrap() error { return e.error }
func (ctxNetError) Timeout() bool   { return true }
func (ctxNetError) Temporary() bool { return true }

var (
	_ net.Error = tempNetError("")
	_ net.Error = ctxNetError{}
)
