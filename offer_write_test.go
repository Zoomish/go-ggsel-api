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

	"github.com/shopspring/decimal"
)

func TestOfferWriteJSON(t *testing.T) {
	t.Parallel()
	price := decimalOf(decimal.RequireFromString("19.99"))
	zero := decimalOf(decimal.NewFromInt(0))
	tests := []struct {
		name   string
		method string
		id     int64
		body   OfferWrite
		want   string
	}{
		{
			name:   "create partial price",
			method: http.MethodPost,
			body: OfferWrite{
				TitleRU:  Ptr("Товар"),
				Price:    &price,
				Currency: Ptr(CurrencyRUB),
			},
			want: `{"title_ru":"Товар","price":19.99,"currency":"RUB"}`,
		},
		{
			name:   "patch false only",
			method: http.MethodPatch,
			id:     15,
			body:   OfferWrite{IsAutoselling: Ptr(false)},
			want:   `{"is_autoselling":false}`,
		},
		{
			name:   "patch zero price",
			method: http.MethodPatch,
			id:     15,
			body:   OfferWrite{Price: &zero},
			want:   `{"price":0}`,
		},
		{
			name:   "patch empty",
			method: http.MethodPatch,
			id:     15,
			body:   OfferWrite{},
			want:   `{}`,
		},
		{
			name:   "empty nested object",
			method: http.MethodPatch,
			id:     15,
			body:   OfferWrite{PrePaymentSettings: &PrePaymentSettingsWrite{}},
			want:   `{"pre_payment_settings":{}}`,
		},
		{
			name:   "nested false",
			method: http.MethodPatch,
			id:     15,
			body: OfferWrite{PrePaymentSettings: &PrePaymentSettingsWrite{
				AllowPayment: Ptr(false),
			}},
			want: `{"pre_payment_settings":{"allow_payment":false}}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var method, path, raw string
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				method = r.Method
				path = r.URL.RequestURI()
				buf, err := io.ReadAll(r.Body)
				if err != nil {
					t.Errorf("read body: %v", err)
				}
				raw = string(buf)
				if r.Header.Get("Authorization") != "seller-key" || strings.Contains(r.Header.Get("Authorization"), "Bearer") {
					t.Errorf("Authorization = %q", r.Header.Get("Authorization"))
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = io.WriteString(w, `{"data":{"id":15,"status":"draft","is_autoselling":false,"title_ru":"Товар","title_en":"","description_ru":"","description_en":"","instructions_ru":"","instructions_en":"","cover_image_ru_url":"","category":{"id":1,"title":"","content_type":"","fee":0,"payment_fee":0,"tree":"","has_children":false},"price":19.99,"currency":"RUB","quantity":0,"is_unlimited_quantity":false,"sold_splitted_products_count":0,"sold_products_count":0,"in_stock_products_count":0,"in_stock_splitted_products_count":0,"has_splitted_products":false,"updated_at":"","created_at":"","extra":true}}`)
			}))
			t.Cleanup(srv.Close)
			c := newTestClient(t, srv.URL, Options{APIKey: "seller-key"})
			var (
				offer Offer
				err   error
			)
			if tt.method == http.MethodPost {
				offer, err = c.CreateOffer(context.Background(), tt.body)
			} else {
				offer, err = c.UpdateOffer(context.Background(), tt.id, tt.body)
			}
			if err != nil {
				t.Fatalf("write: %v", err)
			}
			if method != tt.method {
				t.Fatalf("method = %s", method)
			}
			wantPath := "/api_sellers/v2/offers"
			if tt.method == http.MethodPatch {
				wantPath += "/15"
			}
			if path != wantPath {
				t.Fatalf("path = %s", path)
			}
			if raw != tt.want {
				t.Fatalf("body = %s", raw)
			}
			if strings.Contains(raw, "title_ru") && tt.name == "patch false only" {
				t.Fatalf("unexpected title: %s", raw)
			}
			if !offer.Price.Equal(testDecimal(t, "19.99")) || offer.ID != 15 {
				t.Fatalf("offer = %+v", offer)
			}
		})
	}
}

func TestOfferWriteRejectsEnums(t *testing.T) {
	t.Parallel()
	usd := Currency("USD")
	badDelivery := Delivery("courier")
	badType := NotificationType("sms")
	badMethod := NotificationMethod("PUT")
	tests := []OfferWrite{
		{Currency: &usd},
		{Delivery: &badDelivery},
		{NotificationSettings: &NotificationSettingsWrite{Type: &badType}},
		{NotificationSettings: &NotificationSettingsWrite{HTTPMethod: &badMethod}},
	}
	for _, body := range tests {
		c := newTestClient(t, "", Options{
			APIKey:  "k",
			BaseURL: "https://seller.ggsel.com",
			HTTPClient: &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
				t.Fatal("request was sent")
				return nil, nil
			})},
		})
		if _, err := c.CreateOffer(context.Background(), body); err == nil {
			t.Fatalf("create accepted %+v", body)
		}
		if _, err := c.UpdateOffer(context.Background(), 15, body); err == nil {
			t.Fatalf("update accepted %+v", body)
		}
	}
}

func TestUpdateOfferRejectsID(t *testing.T) {
	t.Parallel()
	c := newTestClient(t, "", Options{
		APIKey:  "k",
		BaseURL: "https://seller.ggsel.com",
		HTTPClient: &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			t.Fatal("request was sent")
			return nil, nil
		})},
	})
	for _, id := range []int64{0, -3} {
		if _, err := c.UpdateOffer(context.Background(), id, OfferWrite{}); err == nil {
			t.Fatalf("id %d was accepted", id)
		}
	}
}

func TestOfferWriteAPIError(t *testing.T) {
	t.Parallel()
	const key = "super-secret-key"
	tests := []struct {
		name   string
		status int
		body   string
		hits   int32
	}{
		{
			name:   "unprocessable",
			status: http.StatusUnprocessableEntity,
			body:   `{"errors":[{"code":"INVALID","entity":{"id":15,"resource":"offer","description":"price"}}]}`,
			hits:   1,
		},
		{
			name:   "unauthorized",
			status: http.StatusUnauthorized,
			body:   `{"errors":[{"code":"UNAUTHORIZED"}]}`,
			hits:   1,
		},
		{
			name:   "unavailable",
			status: http.StatusServiceUnavailable,
			body:   `{"errors":[{"code":"UNAVAILABLE"}]}`,
			hits:   1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			for _, method := range []string{http.MethodPost, http.MethodPatch} {
				var hits atomic.Int32
				srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					hits.Add(1)
					w.WriteHeader(tt.status)
					_, _ = io.WriteString(w, tt.body)
				}))
				t.Cleanup(srv.Close)
				c := newTestClient(t, srv.URL, Options{APIKey: key})
				var err error
				if method == http.MethodPost {
					_, err = c.CreateOffer(context.Background(), OfferWrite{})
				} else {
					_, err = c.UpdateOffer(context.Background(), 15, OfferWrite{IsAutoselling: Ptr(false)})
				}
				var apiErr *APIError
				if !errors.As(err, &apiErr) || apiErr.StatusCode != tt.status || len(apiErr.Errors) != 1 {
					t.Fatalf("%s error = %v", method, err)
				}
				if tt.status == http.StatusUnprocessableEntity {
					item := apiErr.Errors[0]
					if item.Code != "INVALID" || item.Entity == nil || item.Entity.ID == nil || *item.Entity.ID != 15 || item.Entity.Resource != "offer" || item.Entity.Description != "price" {
						t.Fatalf("entity = %+v", item.Entity)
					}
				}
				if strings.Contains(err.Error(), key) {
					t.Fatalf("error text leaks key: %s", err.Error())
				}
				if hits.Load() != tt.hits {
					t.Fatalf("%s hits = %d", method, hits.Load())
				}
			}
		})
	}
}
