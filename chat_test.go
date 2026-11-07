package ggsel

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestListChats(t *testing.T) {
	t.Parallel()
	email := "buyer@example.com"
	var query string
	srv := v1OKServer(t, func(w http.ResponseWriter, r *http.Request) {
		query = r.URL.RawQuery
		if r.URL.Query().Get("filter_new") != "" || r.URL.Query().Get("page") != "" {
			t.Errorf("unexpected query %s", r.URL.RawQuery)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{
			"cnt_pages":2,
			"items":[{"id_i":15,"email":"buyer@example.com","product":3,"last_message":"2026-01-01T00:00:00Z","cnt_msg":4,"cnt_new":null}]
		}`)
	})
	c := newV1Client(t, srv)
	page, err := c.ListChats(context.Background(), ListChatsParams{Email: &email})
	if err != nil {
		t.Fatalf("ListChats: %v", err)
	}
	if page.PageCount != 2 || len(page.Items) != 1 || page.Items[0].InvoiceID != 15 || page.Items[0].LastMessageAt != "2026-01-01T00:00:00Z" || page.Items[0].NewCount != nil {
		t.Fatalf("page = %+v", page)
	}
	if page.Items[0].MessageCount == nil || *page.Items[0].MessageCount != 4 {
		t.Fatalf("count = %v", page.Items[0].MessageCount)
	}
	if !strings.Contains(query, "email=buyer%40example.com") || !strings.Contains(query, "token=tok") {
		t.Fatalf("query = %s", query)
	}
}

func TestListMessages(t *testing.T) {
	t.Parallel()
	count := int64(10)
	var query string
	srv := v1OKServer(t, func(w http.ResponseWriter, r *http.Request) {
		query = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `[{"id":1,"message":"hi","buyer":2,"seller":3,"date_written":"2026-01-01","extra":true}]`)
	})
	c := newV1Client(t, srv)
	items, err := c.ListMessages(context.Background(), 15, ListMessagesParams{Count: &count})
	if err != nil {
		t.Fatalf("ListMessages: %v", err)
	}
	if len(items) != 1 || items[0].Text != "hi" || items[0].ID != 1 || items[0].IsFromBuyer != 2 || items[0].IsFromSeller != 3 {
		t.Fatalf("items = %+v", items)
	}
	if !strings.Contains(query, "id_i=15") || !strings.Contains(query, "count=10") || !strings.Contains(query, "token=tok") || strings.Contains(query, "id_from=") || strings.Contains(query, "newer=") {
		t.Fatalf("query = %s", query)
	}
}

func TestSendMessage(t *testing.T) {
	t.Parallel()
	var method, raw, query string
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api_sellers/api/apilogin" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"retval":0,"token":"tok","valid_thru":"2026-01-02T05:04:05Z"}`)
			return
		}
		n := hits.Add(1)
		method = r.Method
		query = r.URL.RawQuery
		buf, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read: %v", err)
		}
		raw = string(buf)
		if n == 1 {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = io.WriteString(w, `{"retval":1,"retdesc":"later"}`)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)
	c := newV1Client(t, srv)
	err := c.SendMessage(context.Background(), 15, "привет")
	var apiErr *APIError
	if hits.Load() != 1 || !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("hits=%d err=%v", hits.Load(), err)
	}
	if method != http.MethodPost || raw != `{"message":"привет"}` {
		t.Fatalf("method=%s body=%s", method, raw)
	}
	if !strings.Contains(query, "id_i=15") || !strings.Contains(query, "token=tok") {
		t.Fatalf("query = %s", query)
	}
	hits.Store(0)
	srv2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api_sellers/api/apilogin" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"retval":0,"token":"tok","valid_thru":"2026-01-02T05:04:05Z"}`)
			return
		}
		hits.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv2.Close)
	c2 := newV1Client(t, srv2)
	if err := c2.SendMessage(context.Background(), 15, "ok"); err != nil {
		t.Fatalf("SendMessage: %v", err)
	}
	if hits.Load() != 1 {
		t.Fatalf("success hits = %d", hits.Load())
	}
}

func TestSendMessageRejectsEmpty(t *testing.T) {
	t.Parallel()
	c := newTestClient(t, "", Options{
		APIKey:   "k",
		SellerID: 1,
		BaseURL:  "https://seller.ggsel.com",
		HTTPClient: &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			t.Fatal("request was sent")
			return nil, nil
		})},
	})
	if err := c.SendMessage(context.Background(), 0, "hi"); err == nil {
		t.Fatal("id 0 was accepted")
	}
	if err := c.SendMessage(context.Background(), 4, "  "); err == nil {
		t.Fatal("blank text was accepted")
	}
}

func v1OKServer(t *testing.T, next http.HandlerFunc) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api_sellers/api/apilogin" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"retval":0,"token":"tok","valid_thru":"2026-01-02T05:04:05Z"}`)
			return
		}
		next(w, r)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func newV1Client(t *testing.T, srv *httptest.Server) *Client {
	t.Helper()
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	return newTestClient(t, srv.URL, Options{
		APIKey:   "secret",
		SellerID: 4,
		Clock:    func() time.Time { return now },
	})
}
