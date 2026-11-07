package ggsel

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"
)

// Client вызывает Seller API. Его можно использовать из нескольких горутин.
// New не ходит в сеть.
type Client struct {
	baseURL  string
	http     *http.Client
	apiKey   string
	sellerID int64
	locale   string
	clock    func() time.Time
	sleep    func(context.Context, time.Duration) error

	mu          sync.Mutex
	tokenV1     string
	v1ValidThru time.Time
	v1Group     singleflight.Group
}

// New собирает клиент. Пустой ключ — ошибка. SellerID может быть нулём:
// он нужен только методам контура v1.
func New(opts Options) (*Client, error) {
	key := strings.TrimSpace(opts.APIKey)
	if key == "" {
		return nil, fmt.Errorf("ggsel: API key is required")
	}
	base, err := normalizeBaseURL(opts.BaseURL)
	if err != nil {
		return nil, err
	}
	locale := strings.TrimSpace(opts.Locale)
	if locale == "" {
		locale = "ru"
	}
	clock := opts.Clock
	if clock == nil {
		clock = time.Now
	}
	retries := opts.MaxRetries
	if retries == 0 {
		retries = defaultMaxRetries
	}
	if retries < 0 {
		retries = 0
	}
	minWait := opts.RetryBackoffMin
	if minWait <= 0 {
		minWait = defaultBackoffMin
	}
	maxWait := opts.RetryBackoffMax
	if maxWait <= 0 {
		maxWait = defaultBackoffMax
	}
	if maxWait < minWait {
		maxWait = minWait
	}
	c := &Client{
		baseURL:  base,
		apiKey:   key,
		sellerID: opts.SellerID,
		locale:   locale,
		clock:    clock,
		sleep:    sleepContext,
	}
	c.http = httpClient(opts, c, retries, minWait, maxWait)
	return c, nil
}

func (c *Client) now() time.Time {
	if c == nil || c.clock == nil {
		return time.Now()
	}
	return c.clock()
}

func normalizeBaseURL(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		raw = DefaultBaseURL
	}
	raw = strings.TrimRight(raw, "/")
	u, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("ggsel: parse base url: %w", err)
	}
	if u.Scheme == "" || u.Host == "" {
		return "", fmt.Errorf("ggsel: base url must include scheme and host")
	}
	return raw, nil
}

func httpClient(opts Options, c *Client, retries int, minWait, maxWait time.Duration) *http.Client {
	src := opts.HTTPClient
	timeout := defaultTimeout
	var (
		transport     http.RoundTripper
		checkRedirect func(req *http.Request, via []*http.Request) error
		jar           http.CookieJar
	)
	if src != nil {
		timeout = src.Timeout
		transport = src.Transport
		checkRedirect = src.CheckRedirect
		jar = src.Jar
	}
	if transport == nil {
		transport = http.DefaultTransport
	}
	return &http.Client{
		Transport: &retryTripper{
			base:    transport,
			max:     retries,
			min:     minWait,
			maxWait: maxWait,
			client:  c,
		},
		Timeout:       timeout,
		CheckRedirect: checkRedirect,
		Jar:           jar,
	}
}
