package ggsel

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestSignV1(t *testing.T) {
	t.Parallel()
	got := signV1("secret", "1700000000")
	const want = "a09abffb61e9ed1074de76ba1133a9900ffb7cccde778fb92b31afb76f5390ff"
	if got != want {
		t.Fatalf("sign = %s", got)
	}
	if strings.ToLower(got) != got {
		t.Fatal("sign is not lowercase hex")
	}
}

func TestLoginCachesToken(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	current := now
	var logins atomic.Int32
	var orders atomic.Int32
	var mu sync.Mutex
	var gotSign, gotTimestamp string
	var authHeader string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "" {
			t.Errorf("Authorization = %q", r.Header.Get("Authorization"))
		}
		switch r.URL.Path {
		case "/api_sellers/api/apilogin":
			logins.Add(1)
			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Errorf("read login: %v", err)
			}
			var fields map[string]any
			if err := json.Unmarshal(body, &fields); err != nil {
				t.Errorf("login json: %v", err)
			}
			if len(fields) != 3 {
				t.Errorf("login fields = %v", fields)
			}
			mu.Lock()
			gotSign, _ = fields["sign"].(string)
			gotTimestamp, _ = fields["timestamp"].(string)
			mu.Unlock()
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"retval":0,"desc":"ok","token":"cached-token","seller_id":42,"valid_thru":"2026-01-02T04:04:05Z"}`)
		default:
			orders.Add(1)
			authHeader = r.Header.Get("Authorization")
			if r.URL.Query().Get("token") != "cached-token" {
				t.Errorf("token = %q", r.URL.Query().Get("token"))
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"retval":0,"content":{"amount":1,"invoice_state":1}}`)
		}
	}))
	t.Cleanup(srv.Close)
	c := newTestClient(t, srv.URL, Options{
		APIKey:   "secret",
		SellerID: 42,
		Clock:    func() time.Time { return current },
	})
	if _, err := c.GetOrder(context.Background(), 7); err != nil {
		t.Fatalf("first order: %v", err)
	}
	if _, err := c.GetOrder(context.Background(), 7); err != nil {
		t.Fatalf("second order: %v", err)
	}
	if logins.Load() != 1 || orders.Load() != 2 {
		t.Fatalf("logins=%d orders=%d", logins.Load(), orders.Load())
	}
	mu.Lock()
	gotStamp := gotTimestamp
	gotSig := gotSign
	mu.Unlock()
	if gotStamp != strconv.FormatInt(now.Unix(), 10) {
		t.Fatalf("timestamp = %s", gotStamp)
	}
	if gotSig != signV1("secret", gotStamp) {
		t.Fatalf("sign = %s", gotSig)
	}
	if authHeader != "" {
		t.Fatalf("order Authorization = %q", authHeader)
	}
	current = now.Add(2 * time.Hour)
	if _, err := c.GetOrder(context.Background(), 7); err != nil {
		t.Fatalf("expired order: %v", err)
	}
	if logins.Load() != 2 {
		t.Fatalf("logins after expiry = %d", logins.Load())
	}
}

func TestLoginSingleflight(t *testing.T) {
	t.Parallel()
	var logins atomic.Int32
	started := make(chan struct{})
	release := make(chan struct{})
	var once sync.Once
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api_sellers/api/apilogin" {
			logins.Add(1)
			once.Do(func() { close(started) })
			<-release
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"retval":0,"token":"t","valid_thru":"2026-01-02T05:04:05Z"}`)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"retval":0,"content":{"amount":1,"invoice_state":1}}`)
	}))
	t.Cleanup(srv.Close)
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	c := newTestClient(t, srv.URL, Options{
		APIKey:   "secret",
		SellerID: 7,
		Clock:    func() time.Time { return now },
	})
	errc := make(chan error, 2)
	for range 2 {
		go func() {
			_, err := c.GetOrder(context.Background(), 1)
			errc <- err
		}()
	}
	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("login did not start")
	}
	close(release)
	for range 2 {
		if err := <-errc; err != nil {
			t.Fatalf("GetOrder: %v", err)
		}
	}
	if logins.Load() != 1 {
		t.Fatalf("logins = %d", logins.Load())
	}
}

func TestLoginRetvalDoesNotCache(t *testing.T) {
	t.Parallel()
	var logins atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logins.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"retval":3,"desc":"bad sign","token":"should-not-cache"}`)
	}))
	t.Cleanup(srv.Close)
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	c := newTestClient(t, srv.URL, Options{
		APIKey:   "secret",
		SellerID: 7,
		Clock:    func() time.Time { return now },
	})
	err := c.Login(context.Background())
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.RetVal == nil || *apiErr.RetVal != 3 || apiErr.Desc != "bad sign" {
		t.Fatalf("error = %v", err)
	}
	sign := signV1("secret", strconv.FormatInt(now.Unix(), 10))
	if strings.Contains(err.Error(), "secret") || strings.Contains(err.Error(), sign) || strings.Contains(err.Error(), "should-not-cache") {
		t.Fatalf("error text leaks secrets: %s", err.Error())
	}
	if err := c.Login(context.Background()); err == nil {
		t.Fatal("second login succeeded")
	}
	if logins.Load() != 2 {
		t.Fatalf("logins = %d", logins.Load())
	}
}

func TestOrderUnauthorizedDoesNotRelogin(t *testing.T) {
	t.Parallel()
	const key = "super-secret-key"
	const token = "query-secret-token"
	var logins atomic.Int32
	var orders atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "" {
			t.Errorf("Authorization = %q", r.Header.Get("Authorization"))
		}
		if r.URL.Path == "/api_sellers/api/apilogin" {
			logins.Add(1)
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"retval":0,"token":"`+token+`","valid_thru":"2026-01-02T05:04:05Z"}`)
			return
		}
		orders.Add(1)
		status := http.StatusUnauthorized
		if orders.Load() > 1 {
			status = http.StatusOK
		}
		w.WriteHeader(status)
		if status == http.StatusOK {
			_, _ = io.WriteString(w, `{"retval":0,"content":{"amount":1,"invoice_state":3}}`)
			return
		}
		_, _ = io.WriteString(w, `{"retval":-1,"retdesc":"bad","leak":"`+token+`","key":"`+key+`"}`)
	}))
	t.Cleanup(srv.Close)
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	c := newTestClient(t, srv.URL, Options{
		APIKey:   key,
		SellerID: 9,
		Clock:    func() time.Time { return now },
	})
	_, err := c.GetOrder(context.Background(), 4)
	if !IsUnauthorized(err) {
		t.Fatalf("error = %v", err)
	}
	if strings.Contains(err.Error(), key) || strings.Contains(err.Error(), token) || strings.Contains(err.Error(), signV1(key, strconv.FormatInt(now.Unix(), 10))) {
		t.Fatalf("error text leaks secrets: %s", err.Error())
	}
	if logins.Load() != 1 || orders.Load() != 1 {
		t.Fatalf("after 401 logins=%d orders=%d", logins.Load(), orders.Load())
	}
	order, err := c.GetOrder(context.Background(), 4)
	if err != nil {
		t.Fatalf("retry order: %v", err)
	}
	if order.InvoiceState != InvoiceStatePaid {
		t.Fatalf("state = %d", order.InvoiceState)
	}
	if logins.Load() != 2 || orders.Load() != 2 {
		t.Fatalf("after refresh logins=%d orders=%d", logins.Load(), orders.Load())
	}
}

func TestV2DoesNotSendToken(t *testing.T) {
	t.Parallel()
	var v2Query string
	var v2Auth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		v2Query = r.URL.RawQuery
		v2Auth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"data":[],"pagination":{}}`)
	}))
	t.Cleanup(srv.Close)
	c := newTestClient(t, srv.URL, Options{APIKey: "seller-key", SellerID: 1})
	if _, err := c.ListOffers(context.Background(), ListOffersParams{}); err != nil {
		t.Fatalf("ListOffers: %v", err)
	}
	if v2Auth != "seller-key" || strings.Contains(v2Query, "token") {
		t.Fatalf("auth=%q query=%s", v2Auth, v2Query)
	}
}

func TestSellerIDRequired(t *testing.T) {
	t.Parallel()
	c := newTestClient(t, "", Options{
		APIKey:  "k",
		BaseURL: "https://seller.ggsel.com",
		HTTPClient: &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			t.Fatal("request was sent")
			return nil, nil
		})},
	})
	_, err := c.GetOrder(context.Background(), 1)
	if !errors.Is(err, ErrSellerIDRequired) {
		t.Fatalf("error = %v", err)
	}
}

func TestInvalidateV1KeepsFreshToken(t *testing.T) {
	t.Parallel()
	c, err := New(Options{APIKey: "k"})
	if err != nil {
		t.Fatal(err)
	}
	until := time.Now().Add(time.Hour)
	c.storeV1("fresh", until)
	c.invalidateV1("stale")
	token, ok := c.cachedV1()
	if !ok || token != "fresh" {
		t.Fatalf("token = %q ok=%v", token, ok)
	}
}

func TestUnparsedValidThruIsNotCached(t *testing.T) {
	t.Parallel()
	var logins atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/api_sellers/api/apilogin" {
			logins.Add(1)
			_, _ = io.WriteString(w, `{"retval":0,"token":"once","valid_thru":"tomorrow"}`)
			return
		}
		_, _ = io.WriteString(w, `{"retval":0,"content":{"amount":1,"invoice_state":1}}`)
	}))
	t.Cleanup(srv.Close)
	c := newTestClient(t, srv.URL, Options{APIKey: "secret", SellerID: 4})
	if _, err := c.GetOrder(context.Background(), 1); err != nil {
		t.Fatalf("first: %v", err)
	}
	if _, err := c.GetOrder(context.Background(), 1); err != nil {
		t.Fatalf("second: %v", err)
	}
	if logins.Load() != 2 {
		t.Fatalf("logins = %d", logins.Load())
	}
}

func TestTokenSkewBoundary(t *testing.T) {
	t.Parallel()
	valid := time.Date(2026, 1, 2, 4, 4, 5, 0, time.UTC)
	now := valid.Add(-time.Hour)
	current := now
	var logins atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/api_sellers/api/apilogin" {
			logins.Add(1)
			_, _ = io.WriteString(w, `{"retval":0,"token":"tok","valid_thru":"2026-01-02T04:04:05Z"}`)
			return
		}
		_, _ = io.WriteString(w, `{"retval":0,"content":{"amount":1,"invoice_state":1}}`)
	}))
	t.Cleanup(srv.Close)
	c := newTestClient(t, srv.URL, Options{
		APIKey:   "secret",
		SellerID: 4,
		Clock:    func() time.Time { return current },
	})
	if _, err := c.GetOrder(context.Background(), 1); err != nil {
		t.Fatal(err)
	}
	current = valid.Add(-31 * time.Second)
	if _, err := c.GetOrder(context.Background(), 1); err != nil {
		t.Fatal(err)
	}
	if logins.Load() != 1 {
		t.Fatalf("logins inside skew window = %d", logins.Load())
	}
	current = valid.Add(-29 * time.Second)
	if _, err := c.GetOrder(context.Background(), 1); err != nil {
		t.Fatal(err)
	}
	if logins.Load() != 2 {
		t.Fatalf("logins past skew = %d", logins.Load())
	}
}

func TestCanceledLeaderDoesNotFailWaiter(t *testing.T) {
	t.Parallel()
	var logins atomic.Int32
	started := make(chan struct{})
	release := make(chan struct{})
	var once sync.Once
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/api_sellers/api/apilogin" {
			logins.Add(1)
			once.Do(func() { close(started) })
			<-release
			_, _ = io.WriteString(w, `{"retval":0,"token":"tok","valid_thru":"2026-01-02T05:04:05Z"}`)
			return
		}
		_, _ = io.WriteString(w, `{"retval":0,"content":{"amount":19.99,"invoice_state":3}}`)
	}))
	t.Cleanup(srv.Close)
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	c := newTestClient(t, srv.URL, Options{
		APIKey:   "secret",
		SellerID: 7,
		Clock:    func() time.Time { return now },
	})
	ctx, cancel := context.WithCancel(context.Background())
	leader := make(chan error, 1)
	go func() {
		_, err := c.GetOrder(ctx, 1)
		leader <- err
	}()
	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("login did not start")
	}
	cancel()
	select {
	case err := <-leader:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("leader = %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("leader did not return")
	}
	waiter := make(chan error, 1)
	go func() {
		_, err := c.GetOrder(context.Background(), 1)
		waiter <- err
	}()
	time.Sleep(50 * time.Millisecond)
	close(release)
	select {
	case err := <-waiter:
		if err != nil {
			t.Fatalf("waiter = %v", err)
		}
		if errors.Is(err, context.Canceled) {
			t.Fatal("waiter inherited the leader cancel")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("waiter did not return")
	}
	if logins.Load() != 1 {
		t.Fatalf("logins = %d", logins.Load())
	}
}

func TestDetachLoginDeadline(t *testing.T) {
	t.Parallel()
	c := &Client{http: &http.Client{}}
	ctx, cancel := c.detachLogin(context.Background())
	defer cancel()
	deadline, ok := ctx.Deadline()
	if !ok {
		t.Fatal("zero timeout has no deadline")
	}
	left := time.Until(deadline)
	if left < defaultTimeout-time.Second || left > defaultTimeout+time.Second {
		t.Fatalf("deadline in %s", left)
	}
	parent, stop := context.WithTimeout(context.Background(), 40*time.Millisecond)
	defer stop()
	c.http.Timeout = time.Second
	limited, stopLimited := c.detachLogin(parent)
	defer stopLimited()
	got, ok := limited.Deadline()
	if !ok || time.Until(got) > 200*time.Millisecond {
		t.Fatal("caller deadline was dropped")
	}
	parentCancel, cancelParent := context.WithCancel(context.Background())
	child, stopChild := c.detachLogin(parentCancel)
	cancelParent()
	defer stopChild()
	if child.Err() != nil {
		t.Fatal("parent cancel reached the login context")
	}
}
