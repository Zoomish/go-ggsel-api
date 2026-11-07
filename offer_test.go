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

	"github.com/shopspring/decimal"
)

func TestListOffersQuery(t *testing.T) {
	t.Parallel()
	page := int64(2)
	search := "steam"
	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	status := OfferListStatusActive
	sort := OfferSortPriceDesc
	delivery := DeliveryManual
	categoryID := int64(9)
	limit := int64(20)
	tests := []struct {
		name   string
		params ListOffersParams
		want   map[string]string
	}{
		{
			name:   "partial",
			params: ListOffersParams{Page: &page, Search: &search},
			want:   map[string]string{"page": "2", "search": "steam"},
		},
		{
			name: "full",
			params: ListOffersParams{
				Page:          &page,
				Limit:         &limit,
				Search:        &search,
				CategoryID:    &categoryID,
				Status:        &status,
				Sort:          &sort,
				Delivery:      &delivery,
				UpdatedAtFrom: &from,
			},
			want: map[string]string{
				"page":            "2",
				"limit":           "20",
				"search":          "steam",
				"category_id":     "9",
				"status":          "active",
				"sort":            "price_desc",
				"delivery":        "manual",
				"updated_at_from": "2026-01-01T00:00:00Z",
			},
		},
		{
			name:   "empty",
			params: ListOffersParams{},
			want:   map[string]string{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var got http.Request
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				got = *r
				w.Header().Set("Content-Type", "application/json")
				_, _ = io.WriteString(w, `{"data":[],"pagination":{"page":1,"limit":100,"has_next_page":false,"has_previous_page":false,"total_pages":0,"total_count":null}}`)
			}))
			t.Cleanup(srv.Close)
			c := newTestClient(t, srv.URL, Options{APIKey: "seller-key", Locale: "en"})
			page, err := c.ListOffers(context.Background(), tt.params)
			if err != nil {
				t.Fatalf("ListOffers: %v", err)
			}
			if len(page.Offers) != 0 {
				t.Fatalf("offers = %d", len(page.Offers))
			}
			if got.Header.Get("Authorization") != "seller-key" || strings.Contains(got.Header.Get("Authorization"), "Bearer") {
				t.Fatalf("Authorization = %q", got.Header.Get("Authorization"))
			}
			if got.Header.Get("locale") != "en" {
				t.Fatalf("locale = %q", got.Header.Get("locale"))
			}
			if _, ok := got.URL.Query()["token"]; ok {
				t.Fatalf("token query = %s", got.URL.RawQuery)
			}
			if len(got.URL.Query()) != len(tt.want) {
				t.Fatalf("query = %s", got.URL.RawQuery)
			}
			for key, value := range tt.want {
				if got.URL.Query().Get(key) != value {
					t.Fatalf("%s = %q", key, got.URL.Query().Get(key))
				}
			}
		})
	}
}

func TestListOffersDecodes(t *testing.T) {
	t.Parallel()
	const body = `{
		"data":[{
			"id":7,
			"status":"active",
			"title_ru":"Ключ",
			"title_en":null,
			"price":19.99,
			"currency":"RUB",
			"category":{"id":3,"title":"Games","content_type":"key","fee":0.05,"payment_fee":0.02,"tree":"Games","has_children":false},
			"has_options":true,
			"has_products":false,
			"has_splitted_products":false,
			"cover_image_ru_url":"https://img/ru.png",
			"extra":{"ignore":true}
		}],
		"pagination":{"page":1,"limit":100,"has_next_page":true,"has_previous_page":false,"total_pages":4,"total_count":40},
		"unknown":1
	}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("locale") != "ru" {
			t.Errorf("locale = %q", r.Header.Get("locale"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, body)
	}))
	t.Cleanup(srv.Close)
	c := newTestClient(t, srv.URL, Options{APIKey: "seller-key"})
	page, err := c.ListOffers(context.Background(), ListOffersParams{})
	if err != nil {
		t.Fatalf("ListOffers: %v", err)
	}
	if len(page.Offers) != 1 || page.Offers[0].TitleEN != nil || !page.Offers[0].HasOptions || page.Offers[0].HasProducts {
		t.Fatalf("offer = %+v", page.Offers[0])
	}
	want := testDecimal(t, "19.99")
	if page.Offers[0].Price == nil || !page.Offers[0].Price.Equal(want) {
		t.Fatalf("price = %v", page.Offers[0].Price)
	}
	fee := testDecimal(t, "0.05")
	if !page.Offers[0].Category.Fee.Equal(fee) || page.Offers[0].Currency != CurrencyRUB {
		t.Fatalf("category = %+v", page.Offers[0].Category)
	}
	if page.Pagination.TotalPages != 4 || page.Pagination.TotalCount == nil || *page.Pagination.TotalCount != 40 || !page.Pagination.HasNextPage {
		t.Fatalf("pagination = %+v", page.Pagination)
	}
}

func TestListOffersNullData(t *testing.T) {
	t.Parallel()
	srv := jsonServer(t, `{"data":null,"pagination":{}}`)
	c := newTestClient(t, srv.URL, Options{APIKey: "k"})
	_, err := c.ListOffers(context.Background(), ListOffersParams{})
	if err == nil || !errors.Is(err, errEmptyData) {
		t.Fatalf("error = %v", err)
	}
}

func TestListOffersRejectsFilter(t *testing.T) {
	t.Parallel()
	archived := OfferListStatus(OfferStatusArchived)
	badSort := OfferSort("newest")
	badDelivery := Delivery("courier")
	tests := []ListOffersParams{
		{Status: &archived},
		{Sort: &badSort},
		{Delivery: &badDelivery},
	}
	for _, params := range tests {
		c := newTestClient(t, "", Options{
			APIKey:  "k",
			BaseURL: "https://seller.ggsel.com",
			HTTPClient: &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
				t.Fatal("request was sent")
				return nil, nil
			})},
		})
		if _, err := c.ListOffers(context.Background(), params); err == nil {
			t.Fatalf("expected error for %+v", params)
		}
	}
}

func TestListOffersUnauthorizedDoesNotRetry(t *testing.T) {
	t.Parallel()
	const key = "super-secret-key"
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		if r.Header.Get("Authorization") != key {
			t.Errorf("Authorization = %q", r.Header.Get("Authorization"))
		}
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = io.WriteString(w, `{"errors":[{"code":"UNAUTHORIZED"}]}`)
	}))
	t.Cleanup(srv.Close)
	c := newTestClient(t, srv.URL, Options{APIKey: key})
	_, err := c.ListOffers(context.Background(), ListOffersParams{})
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusUnauthorized || len(apiErr.Errors) != 1 || apiErr.Errors[0].Code != "UNAUTHORIZED" {
		t.Fatalf("error = %v", err)
	}
	if !IsUnauthorized(err) {
		t.Fatal("expected unauthorized")
	}
	if strings.Contains(err.Error(), key) {
		t.Fatalf("error text leaks key: %s", err.Error())
	}
	if hits.Load() != 1 {
		t.Fatalf("hits = %d", hits.Load())
	}
}

func TestListOffersRetries503(t *testing.T) {
	t.Parallel()
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := hits.Add(1)
		if n < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"data":[],"pagination":{"has_next_page":false,"has_previous_page":false}}`)
	}))
	t.Cleanup(srv.Close)
	c := newTestClient(t, srv.URL, Options{APIKey: "k"})
	if _, err := c.ListOffers(context.Background(), ListOffersParams{}); err != nil {
		t.Fatalf("ListOffers: %v", err)
	}
	if hits.Load() != 3 {
		t.Fatalf("hits = %d", hits.Load())
	}
}

func TestGetOffer(t *testing.T) {
	t.Parallel()
	const body = `{
		"data":{
			"id":15,
			"status":"paused",
			"is_autoselling":false,
			"title_ru":"Товар",
			"title_en":"Item",
			"description_ru":"О","description_en":"D",
			"instructions_ru":"И","instructions_en":"I",
			"cover_image_ru_url":"https://img/ru.png",
			"cover_image_en_url":null,
			"category":{"id":3,"title":"Games","content_type":"key","fee":1.5,"payment_fee":0.1,"tree":"Games","has_children":true},
			"price":19.99,
			"currency":"RUB",
			"quantity":4,
			"is_unlimited_quantity":false,
			"sold_splitted_products_count":1,
			"sold_products_count":2,
			"in_stock_products_count":8,
			"in_stock_splitted_products_count":0,
			"has_splitted_products":true,
			"updated_at":"2026-01-01T00:00:00Z",
			"created_at":"2026-01-01T00:00:00Z",
			"extra":true
		}
	}`
	var path string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		if r.URL.RawQuery != "" {
			t.Errorf("query = %s", r.URL.RawQuery)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, body)
	}))
	t.Cleanup(srv.Close)
	c := newTestClient(t, srv.URL, Options{APIKey: "seller-key"})
	offer, err := c.GetOffer(context.Background(), 15)
	if err != nil {
		t.Fatalf("GetOffer: %v", err)
	}
	if path != "/api_sellers/v2/offers/15" {
		t.Fatalf("path = %s", path)
	}
	if offer.InStockProductsCount != 8 || offer.IsAutoselling || offer.CoverImageENURL != nil {
		t.Fatalf("offer = %+v", offer)
	}
	if !offer.Price.Equal(testDecimal(t, "19.99")) || !offer.Category.PaymentFee.Equal(testDecimal(t, "0.1")) {
		t.Fatalf("price = %s fee = %s", offer.Price, offer.Category.PaymentFee)
	}
}

func TestGetOfferRejectsID(t *testing.T) {
	t.Parallel()
	c := newTestClient(t, "", Options{
		APIKey:  "k",
		BaseURL: "https://seller.ggsel.com",
		HTTPClient: &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			t.Fatal("request was sent")
			return nil, nil
		})},
	})
	for _, id := range []int64{0, -1} {
		if _, err := c.GetOffer(context.Background(), id); err == nil {
			t.Fatalf("id %d was accepted", id)
		}
	}
}

func TestGetOfferNotFound(t *testing.T) {
	t.Parallel()
	const key = "super-secret-key"
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.WriteHeader(http.StatusNotFound)
		_, _ = io.WriteString(w, `{"errors":[{"code":"NOT_FOUND","entity":{"id":15,"resource":"offer","description":"missing"}}]}`)
	}))
	t.Cleanup(srv.Close)
	c := newTestClient(t, srv.URL, Options{APIKey: key})
	_, err := c.GetOffer(context.Background(), 15)
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusNotFound || apiErr.Path != "/api_sellers/v2/offers/15" {
		t.Fatalf("error = %v", err)
	}
	if len(apiErr.Errors) != 1 || apiErr.Errors[0].Entity == nil || apiErr.Errors[0].Entity.Resource != "offer" {
		t.Fatalf("errors = %+v", apiErr.Errors)
	}
	if strings.Contains(err.Error(), key) {
		t.Fatalf("error text leaks key: %s", err.Error())
	}
	if hits.Load() != 1 {
		t.Fatalf("hits = %d", hits.Load())
	}
}

func testDecimal(t *testing.T, raw string) Decimal {
	t.Helper()
	d, err := decimal.NewFromString(raw)
	if err != nil {
		t.Fatal(err)
	}
	return decimalOf(d)
}

func jsonServer(t *testing.T, body string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, body)
	}))
	t.Cleanup(srv.Close)
	return srv
}
