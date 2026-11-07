package ggsel

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/shopspring/decimal"
)

func TestNew(t *testing.T) {
	t.Parallel()
	when := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	c, err := New(Options{
		APIKey:   "  seller-key  ",
		SellerID: 0,
		Clock:    func() time.Time { return when },
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if c.apiKey != "seller-key" {
		t.Fatalf("api key = %q", c.apiKey)
	}
	if c.baseURL != DefaultBaseURL {
		t.Fatalf("base = %q", c.baseURL)
	}
	if c.locale != "ru" {
		t.Fatalf("locale = %q", c.locale)
	}
	if c.sellerID != 0 {
		t.Fatalf("seller id = %d", c.sellerID)
	}
	if !c.now().Equal(when) {
		t.Fatalf("now = %s", c.now())
	}
	if c.http == nil || c.http.Timeout != defaultTimeout {
		t.Fatalf("timeout = %s", c.http.Timeout)
	}
	if c.sleep == nil {
		t.Fatal("sleep is nil")
	}

	transport := http.DefaultTransport
	src := &http.Client{Timeout: 5 * time.Second, Transport: transport}
	copied, err := New(Options{
		APIKey:     "k",
		HTTPClient: src,
		BaseURL:    "https://seller.ggsel.com/",
		Locale:     " en ",
		SellerID:   42,
	})
	if err != nil {
		t.Fatalf("New with client: %v", err)
	}
	if src.Transport != transport || src.Timeout != 5*time.Second {
		t.Fatal("New mutated the caller's http.Client")
	}
	if copied.http == src {
		t.Fatal("New reused the caller's http.Client")
	}
	if copied.http.Timeout != 5*time.Second {
		t.Fatalf("copied timeout = %s", copied.http.Timeout)
	}
	if copied.baseURL != "https://seller.ggsel.com" {
		t.Fatalf("base = %q", copied.baseURL)
	}
	if copied.locale != "en" || copied.sellerID != 42 {
		t.Fatalf("locale=%q seller=%d", copied.locale, copied.sellerID)
	}
}

func TestNewRejectsInput(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		opts Options
	}{
		{name: "empty key", opts: Options{}},
		{name: "blank key", opts: Options{APIKey: " \t "}},
		{name: "no scheme", opts: Options{APIKey: "k", BaseURL: "seller.ggsel.com"}},
		{name: "relative", opts: Options{APIKey: "k", BaseURL: "/api"}},
		{name: "empty host", opts: Options{APIKey: "k", BaseURL: "http://"}},
		{name: "broken", opts: Options{APIKey: "k", BaseURL: "://bad"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if _, err := New(tt.opts); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestDoAuthAndBody(t *testing.T) {
	t.Parallel()
	const key = "super-secret-key"
	var gotAuth, gotAccept, gotType, gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotAccept = r.Header.Get("Accept")
		gotType = r.Header.Get("Content-Type")
		buf, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read body: %v", err)
		}
		gotBody = string(buf)
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"price":19.99,"extra":{"keep":true}}`)
	}))
	t.Cleanup(srv.Close)

	c := newTestClient(t, srv.URL, Options{APIKey: "  " + key + "  "})
	price := decimalOf(decimal.RequireFromString("19.99"))
	off := false
	body := struct {
		Title *string  `json:"title,omitempty"`
		Price *Decimal `json:"price,omitempty"`
		On    *bool    `json:"on,omitempty"`
	}{Price: &price, On: &off}
	var out struct {
		Price Decimal `json:"price"`
	}
	err := c.do(context.Background(), call{
		method: http.MethodPost,
		path:   "/api_sellers/v2/offers",
		header: http.Header{"Authorization": []string{"Bearer leaked"}},
		body:   body,
		dest:   &out,
		auth:   authV2,
	})
	if err != nil {
		t.Fatalf("do: %v", err)
	}
	if gotAuth != key {
		t.Fatalf("Authorization = %q", gotAuth)
	}
	if strings.Contains(gotAuth, "Bearer") {
		t.Fatalf("Authorization uses Bearer: %q", gotAuth)
	}
	if gotAccept != "application/json" || gotType != "application/json" {
		t.Fatalf("accept=%q content-type=%q", gotAccept, gotType)
	}
	if gotBody != `{"price":19.99,"on":false}` {
		t.Fatalf("body = %s", gotBody)
	}
	if !out.Price.Equal(price) {
		t.Fatalf("price = %s", out.Price)
	}
}

func TestDoAuthNoneStripsAuthorization(t *testing.T) {
	t.Parallel()
	var got string
	var seen bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = true
		got = r.Header.Get("Authorization")
		if len(r.Header.Values("Authorization")) != 0 {
			t.Errorf("Authorization values = %v", r.Header.Values("Authorization"))
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(srv.Close)

	c := newTestClient(t, srv.URL, Options{APIKey: "super-secret-key"})
	var typedNil *struct {
		A string `json:"a"`
	}
	err := c.do(context.Background(), call{
		method: http.MethodPost,
		path:   "/api_sellers/api/apilogin",
		header: http.Header{"Authorization": []string{"super-secret-key"}},
		body:   typedNil,
		auth:   authNone,
	})
	if err != nil {
		t.Fatalf("do: %v", err)
	}
	if !seen {
		t.Fatal("request was not sent")
	}
	if got != "" {
		t.Fatalf("Authorization = %q", got)
	}
}

func TestDoErrorHidesSecrets(t *testing.T) {
	t.Parallel()
	const key = "super-secret-key"
	const token = "query-secret-token"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("token") != token {
			t.Errorf("token query = %q", r.URL.Query().Get("token"))
		}
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = io.WriteString(w, `{"retval":1,"retdesc":"bad","leak":"`+key+`","token":"`+token+`"}`)
	}))
	t.Cleanup(srv.Close)

	c := newTestClient(t, srv.URL, Options{APIKey: key})
	err := c.do(context.Background(), call{
		method: http.MethodGet,
		path:   "/api_sellers/api/purchase/info/15",
		query:  url.Values{"token": {token}},
		auth:   authV2,
		retry:  true,
	})
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("error = %v", err)
	}
	if apiErr.Path != "/api_sellers/api/purchase/info/15" || apiErr.StatusCode != http.StatusUnauthorized {
		t.Fatalf("api error = %+v", apiErr)
	}
	if apiErr.RetVal == nil || *apiErr.RetVal != 1 || apiErr.RetDesc != "bad" {
		t.Fatalf("retval = %v retdesc = %q", apiErr.RetVal, apiErr.RetDesc)
	}
	if !IsUnauthorized(err) {
		t.Fatal("expected unauthorized")
	}
	text := err.Error()
	if strings.Contains(text, key) || strings.Contains(text, token) || strings.Contains(text, "leak") {
		t.Fatalf("error text leaks secrets: %s", text)
	}
	if strings.Contains(string(apiErr.Body), key) == false {
		t.Fatal("raw body should still keep the payload for the caller")
	}
}

func TestDoTransportErrorHidesURL(t *testing.T) {
	t.Parallel()
	const key = "super-secret-key"
	const token = "query-secret-token"
	c := newTestClient(t, "", Options{
		APIKey:  key,
		BaseURL: "https://seller.ggsel.com",
		HTTPClient: &http.Client{
			Timeout: 5 * time.Second,
			Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				return nil, errors.New("dial " + r.URL.String() + " key " + key)
			}),
		},
	})
	err := c.do(context.Background(), call{
		method: http.MethodGet,
		path:   "/private",
		query:  url.Values{"token": {token}},
		auth:   authV2,
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if IsUnauthorized(err) {
		t.Fatal("transport error is not unauthorized")
	}
	for current := err; current != nil; current = errors.Unwrap(current) {
		text := current.Error()
		if strings.Contains(text, key) || strings.Contains(text, token) || strings.Contains(text, "seller.ggsel.com") {
			t.Fatalf("error chain leaks request: %s", text)
		}
	}
	if !strings.Contains(err.Error(), "request failed") || !strings.Contains(err.Error(), "dial") {
		t.Fatalf("error = %s", err.Error())
	}
}

func TestDoContextCanceled(t *testing.T) {
	t.Parallel()
	const token = "query-secret-token"
	c := newTestClient(t, "", Options{
		APIKey:  "super-secret-key",
		BaseURL: "https://seller.ggsel.com",
		HTTPClient: &http.Client{
			Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				return nil, &url.Error{Op: "Get", URL: r.URL.String(), Err: context.Canceled}
			}),
		},
	})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := c.do(ctx, call{
		method: http.MethodGet,
		path:   "/private",
		query:  url.Values{"token": {token}},
		auth:   authV2,
		retry:  true,
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v", err)
	}
	if strings.Contains(err.Error(), token) || strings.Contains(err.Error(), "super-secret-key") {
		t.Fatalf("error text leaks request: %s", err.Error())
	}
}

func TestDoRejectsBadCall(t *testing.T) {
	t.Parallel()
	c := newTestClient(t, "", Options{
		APIKey:  "k",
		BaseURL: "https://seller.ggsel.com",
		HTTPClient: &http.Client{
			Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
				t.Fatal("request was sent")
				return nil, nil
			}),
		},
	})
	tests := []call{
		{method: http.MethodGet, path: "relative", auth: authV2},
		{method: http.MethodGet, path: "/ok?token=secret", auth: authV2},
		{method: "", path: "/ok", auth: authV2},
		{method: http.MethodGet, path: "/ok", auth: authMode(9)},
	}
	for _, call := range tests {
		if err := c.do(context.Background(), call); err == nil {
			t.Fatalf("expected error for %+v", call)
		}
	}
	if err := c.do(nil, call{method: http.MethodGet, path: "/ok", auth: authNone}); err == nil {
		t.Fatal("expected nil context error")
	}
}

func TestDoDecodeAndSize(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		status  int
		body    string
		dest    bool
		wantErr string
	}{
		{name: "empty success", status: http.StatusNoContent, dest: false},
		{name: "invalid json", status: http.StatusOK, body: `{`, dest: true, wantErr: "decode"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.status)
				_, _ = io.WriteString(w, tt.body)
			}))
			t.Cleanup(srv.Close)
			c := newTestClient(t, srv.URL, Options{APIKey: "k"})
			var dest any
			if tt.dest {
				dest = &struct {
					Price Decimal `json:"price"`
				}{}
			}
			err := c.do(context.Background(), call{
				method: http.MethodGet,
				path:   "/ok",
				dest:   dest,
				auth:   authV2,
			})
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("do: %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("error = %v", err)
			}
			var apiErr *APIError
			if errors.As(err, &apiErr) {
				t.Fatalf("decode error is APIError: %v", err)
			}
		})
	}
}

func TestDoResponseTooLarge(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		buf := bytes.Repeat([]byte("x"), maxResponseBody+1)
		_, _ = w.Write(buf)
	}))
	t.Cleanup(srv.Close)
	c := newTestClient(t, srv.URL, Options{APIKey: "k"})
	err := c.do(context.Background(), call{
		method: http.MethodGet,
		path:   "/huge",
		auth:   authV2,
	})
	if err == nil || !strings.Contains(err.Error(), "exceeds") {
		t.Fatalf("error = %v", err)
	}
}

func TestBuildURLKeepsPrefix(t *testing.T) {
	t.Parallel()
	c := &Client{baseURL: "https://seller.ggsel.com/extra"}
	got, err := c.buildURL("/v", url.Values{"a": {"b"}})
	if err != nil {
		t.Fatalf("buildURL: %v", err)
	}
	if got != "https://seller.ggsel.com/extra/v?a=b" {
		t.Fatalf("url = %s", got)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func newTestClient(t *testing.T, base string, opts Options) *Client {
	t.Helper()
	if opts.APIKey == "" {
		opts.APIKey = "test-key"
	}
	if opts.BaseURL == "" {
		opts.BaseURL = base
	}
	if opts.HTTPClient == nil {
		opts.HTTPClient = &http.Client{Timeout: 5 * time.Second}
	}
	c, err := New(opts)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	c.sleep = func(context.Context, time.Duration) error { return nil }
	return c
}
