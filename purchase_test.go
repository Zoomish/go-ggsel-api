package ggsel

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestGetOrder(t *testing.T) {
	t.Parallel()
	const body = `{
		"retval":0,
		"retdesc":"ok",
		"extra":true,
		"content":{
			"item_id":null,
			"content_id":9,
			"name":"Ключ",
			"amount":19.99,
			"currency_type":"RUB",
			"invoice_state":3,
			"purchase_date":"2026-01-01T00:00:00Z",
			"cnt_goods":"2",
			"agent_percent":0.1,
			"agent_fee":1.5,
			"profit":18.49,
			"unknown":{"keep":true}
		}
	}`
	var locale, auth, query string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api_sellers/api/apilogin" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"retval":0,"token":"tok","valid_thru":"2026-01-02T05:04:05Z"}`)
			return
		}
		locale = r.Header.Get("locale")
		auth = r.Header.Get("Authorization")
		query = r.URL.RawQuery
		if r.URL.Path != "/api_sellers/api/purchase/info/15" {
			t.Errorf("path = %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, body)
	}))
	t.Cleanup(srv.Close)
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	c := newTestClient(t, srv.URL, Options{
		APIKey:   "secret",
		SellerID: 4,
		Locale:   "en",
		Clock:    func() time.Time { return now },
	})
	order, err := c.GetOrder(context.Background(), 15)
	if err != nil {
		t.Fatalf("GetOrder: %v", err)
	}
	if locale != "en" || auth != "" || !strings.Contains(query, "token=tok") {
		t.Fatalf("locale=%q auth=%q query=%s", locale, auth, query)
	}
	if order.ContentID != 9 || order.InvoiceState != InvoiceStatePaid || order.CountGoods == nil || *order.CountGoods != "2" {
		t.Fatalf("order = %+v", order)
	}
	if !order.Amount.Equal(testDecimal(t, "19.99")) || order.AgentPercent == nil || !order.AgentPercent.Equal(testDecimal(t, "0.1")) {
		t.Fatalf("amount = %s percent = %v", order.Amount, order.AgentPercent)
	}
	if !order.Profit.Equal(testDecimal(t, "18.49")) {
		t.Fatalf("profit = %s", order.Profit)
	}
}

func TestGetOrderRetval(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/api_sellers/api/apilogin" {
			_, _ = io.WriteString(w, `{"retval":0,"token":"tok","valid_thru":"2026-01-02T05:04:05Z"}`)
			return
		}
		_, _ = io.WriteString(w, `{"retval":2,"retdesc":"denied"}`)
	}))
	t.Cleanup(srv.Close)
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	c := newTestClient(t, srv.URL, Options{APIKey: "secret", SellerID: 4, Clock: func() time.Time { return now }})
	_, err := c.GetOrder(context.Background(), 3)
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.RetVal == nil || *apiErr.RetVal != 2 || apiErr.RetDesc != "denied" {
		t.Fatalf("error = %v", err)
	}
}

func TestGetOrderRejectsID(t *testing.T) {
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
	if _, err := c.GetOrder(context.Background(), 0); err == nil {
		t.Fatal("id 0 was accepted")
	}
}

func TestCheckUniqueCode(t *testing.T) {
	t.Parallel()
	var path, query string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api_sellers/api/apilogin" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"retval":0,"token":"tok","valid_thru":"2026-01-02T05:04:05Z"}`)
			return
		}
		path = r.RequestURI
		query = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{
			"retval":0,
			"retdesc":"ok",
			"inv":8,
			"id_goods":3,
			"amount":19.99,
			"type_curr":"RUB",
			"amount_usd":0.2,
			"profit":18,
			"date_pay":"2026-01-01",
			"agent_percent":"5",
			"extra":1
		}`)
	}))
	t.Cleanup(srv.Close)
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	c := newTestClient(t, srv.URL, Options{APIKey: "secret", SellerID: 4, Clock: func() time.Time { return now }})
	code, err := c.CheckUniqueCode(context.Background(), "a/b")
	if err != nil {
		t.Fatalf("CheckUniqueCode: %v", err)
	}
	if !strings.Contains(path, "/api_sellers/api/purchases/unique-code/a%2Fb") {
		t.Fatalf("path = %s", path)
	}
	if !strings.Contains(query, "token=tok") {
		t.Fatalf("query = %s", query)
	}
	if code.InvoiceID != 8 || code.AgentPercent == nil || *code.AgentPercent != "5" {
		t.Fatalf("code = %+v", code)
	}
	if !code.Amount.Equal(testDecimal(t, "19.99")) || !code.AmountUSD.Equal(testDecimal(t, "0.2")) {
		t.Fatalf("amount = %s usd = %s", code.Amount, code.AmountUSD)
	}
}

func TestCheckUniqueCodeHidesSecret(t *testing.T) {
	t.Parallel()
	const secret = "order-secret-code"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/api_sellers/api/apilogin" {
			_, _ = io.WriteString(w, `{"retval":0,"token":"tok","valid_thru":"2026-01-02T05:04:05Z"}`)
			return
		}
		w.WriteHeader(http.StatusNotFound)
		_, _ = io.WriteString(w, `{"retval":-1,"retdesc":"missing"}`)
	}))
	t.Cleanup(srv.Close)
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	c := newTestClient(t, srv.URL, Options{APIKey: "secret", SellerID: 4, Clock: func() time.Time { return now }})
	_, err := c.CheckUniqueCode(context.Background(), secret)
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("error = %v", err)
	}
	if apiErr.Path != "/api_sellers/api/purchases/unique-code/{unique_code}" {
		t.Fatalf("path = %s", apiErr.Path)
	}
	if strings.Contains(err.Error(), secret) || strings.Contains(err.Error(), "tok") {
		t.Fatalf("error text leaks secrets: %s", err.Error())
	}
}

func TestCheckUniqueCodeRequiresCode(t *testing.T) {
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
	if _, err := c.CheckUniqueCode(context.Background(), "  "); err == nil {
		t.Fatal("blank code was accepted")
	}
}
