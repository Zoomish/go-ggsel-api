package ggsel

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type retryCtxKey struct{}

func withSafeRetry(ctx context.Context) context.Context {
	return context.WithValue(ctx, retryCtxKey{}, true)
}

func safeRetry(ctx context.Context) bool {
	v, _ := ctx.Value(retryCtxKey{}).(bool)
	return v
}

type retryTripper struct {
	base    http.RoundTripper
	max     int
	min     time.Duration
	maxWait time.Duration
	client  *Client
}

func (t *retryTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if req == nil {
		return nil, fmt.Errorf("ggsel: nil request")
	}
	next := t.base
	if next == nil {
		next = http.DefaultTransport
	}
	if !safeRetry(req.Context()) || t.max <= 0 {
		return next.RoundTrip(req)
	}
	var resp *http.Response
	var err error
	for attempt := 0; attempt <= t.max; attempt++ {
		req, err = replay(req)
		if err != nil {
			return nil, err
		}
		resp, err = next.RoundTrip(req)
		if err != nil {
			if attempt == t.max || !retryableNet(err) || req.Context().Err() != nil {
				return nil, err
			}
			if sleepErr := t.wait(req.Context(), t.backoff(attempt)); sleepErr != nil {
				return nil, sleepErr
			}
			continue
		}
		if attempt == t.max || !retryableStatus(resp.StatusCode) {
			return resp, nil
		}
		wait := retryDelay(resp, t.backoff(attempt), t.now())
		drain(resp)
		if sleepErr := t.wait(req.Context(), wait); sleepErr != nil {
			return nil, sleepErr
		}
	}
	return resp, err
}

func (t *retryTripper) now() time.Time {
	if t == nil || t.client == nil {
		return time.Now()
	}
	return t.client.now()
}

func (t *retryTripper) wait(ctx context.Context, d time.Duration) error {
	fn := sleepContext
	if t.client != nil && t.client.sleep != nil {
		fn = t.client.sleep
	}
	return fn(ctx, d)
}

func retryableStatus(code int) bool {
	switch code {
	case http.StatusTooManyRequests, http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		return true
	default:
		return false
	}
}

func retryableNet(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	var netErr net.Error
	return errors.As(err, &netErr)
}

func replay(req *http.Request) (*http.Request, error) {
	next := req.Clone(req.Context())
	if req.Body == nil || req.Body == http.NoBody {
		return next, nil
	}
	if req.GetBody == nil {
		return nil, fmt.Errorf("ggsel: request body is not replayable")
	}
	body, err := req.GetBody()
	if err != nil {
		return nil, fmt.Errorf("ggsel: rewind request body: %w", err)
	}
	next.Body = body
	return next, nil
}

func drain(resp *http.Response) {
	if resp == nil || resp.Body == nil {
		return
	}
	_, err := io.Copy(io.Discard, io.LimitReader(resp.Body, 1<<20))
	closeErr := resp.Body.Close()
	if err != nil || closeErr != nil {
		return
	}
}

func (t *retryTripper) backoff(attempt int) time.Duration {
	minWait := t.min
	maxWait := t.maxWait
	if minWait <= 0 {
		minWait = defaultBackoffMin
	}
	if maxWait <= 0 {
		maxWait = defaultBackoffMax
	}
	if maxWait < minWait {
		maxWait = minWait
	}
	d := minWait
	for i := 0; i < attempt; i++ {
		if d > maxWait/2 {
			d = maxWait
			break
		}
		d *= 2
	}
	if d < minWait {
		d = minWait
	}
	if d > maxWait {
		d = maxWait
	}
	low := d / 2
	if low < minWait {
		low = minWait
	}
	if low >= d {
		return d
	}
	span := int64(d - low)
	return low + time.Duration(jitter(span+1))
}

func jitter(n int64) int64 {
	if n <= 1 {
		return 0
	}
	var buf [8]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return 0
	}
	return int64(binary.LittleEndian.Uint64(buf[:]) % uint64(n))
}

func retryDelay(resp *http.Response, fallback time.Duration, now time.Time) time.Duration {
	if resp == nil {
		return fallback
	}
	wait, ok := parseRetryAfter(resp.Header.Get("Retry-After"), now)
	if !ok {
		return fallback
	}
	if wait > maxRetryAfter {
		return maxRetryAfter
	}
	return wait
}

func parseRetryAfter(v string, now time.Time) (time.Duration, bool) {
	v = strings.TrimSpace(v)
	if v == "" {
		return 0, false
	}
	if sec, err := strconv.Atoi(v); err == nil {
		if sec < 0 {
			return 0, false
		}
		return time.Duration(sec) * time.Second, true
	}
	when, err := http.ParseTime(v)
	if err != nil {
		return 0, false
	}
	d := when.Sub(now)
	if d < 0 {
		return 0, true
	}
	return d, true
}

func sleepContext(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return nil
	}
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
