package ggsel

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestListCategories(t *testing.T) {
	t.Parallel()
	parent := int64(4)
	page := int64(2)
	var query string
	var locale string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query = r.URL.RawQuery
		locale = r.Header.Get("locale")
		if r.Header.Get("Authorization") != "seller-key" {
			t.Errorf("Authorization = %q", r.Header.Get("Authorization"))
		}
		if _, ok := r.URL.Query()["q"]; ok {
			t.Errorf("unexpected q")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{
			"data":[{"id":8,"title":"Keys","content_type":"key","fee":0.05,"payment_fee":0.02,"tree":"Games / Keys","has_children":false,"extra":1}],
			"pagination":{"page":2,"limit":10,"has_next_page":false,"has_previous_page":true,"total_pages":2,"total_count":null}
		}`)
	}))
	t.Cleanup(srv.Close)
	c := newTestClient(t, srv.URL, Options{APIKey: "seller-key"})
	got, err := c.ListCategories(context.Background(), ListCategoriesParams{ParentID: &parent, Page: &page})
	if err != nil {
		t.Fatalf("ListCategories: %v", err)
	}
	if locale != "ru" {
		t.Fatalf("locale = %q", locale)
	}
	if query != "page=2&parent_id=4" && query != "parent_id=4&page=2" {
		t.Fatalf("query = %s", query)
	}
	if len(got.Categories) != 1 || got.Categories[0].Title != "Keys" || got.Categories[0].HasChildren {
		t.Fatalf("categories = %+v", got.Categories)
	}
	if !got.Categories[0].Fee.Equal(testDecimal(t, "0.05")) || !got.Categories[0].PaymentFee.Equal(testDecimal(t, "0.02")) {
		t.Fatalf("fees = %s %s", got.Categories[0].Fee, got.Categories[0].PaymentFee)
	}
	if got.Pagination.TotalCount != nil || !got.Pagination.HasPreviousPage || got.Pagination.TotalPages != 2 {
		t.Fatalf("pagination = %+v", got.Pagination)
	}
}

func TestSearchCategories(t *testing.T) {
	t.Parallel()
	limit := int64(5)
	var query string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"data":[],"pagination":{"page":1,"limit":5,"has_next_page":false,"has_previous_page":false,"total_pages":0,"total_count":0}}`)
	}))
	t.Cleanup(srv.Close)
	c := newTestClient(t, srv.URL, Options{APIKey: "seller-key", Locale: "en"})
	got, err := c.SearchCategories(context.Background(), "steam keys", CategoryPageParams{Limit: &limit})
	if err != nil {
		t.Fatalf("SearchCategories: %v", err)
	}
	if len(got.Categories) != 0 {
		t.Fatalf("categories = %+v", got.Categories)
	}
	if query != "limit=5&q=steam+keys" && query != "q=steam+keys&limit=5" {
		t.Fatalf("query = %s", query)
	}
}

func TestSearchCategoriesRequiresQuery(t *testing.T) {
	t.Parallel()
	c := newTestClient(t, "", Options{
		APIKey:  "k",
		BaseURL: "https://seller.ggsel.com",
		HTTPClient: &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			t.Fatal("request was sent")
			return nil, nil
		})},
	})
	for _, query := range []string{"", "   "} {
		if _, err := c.SearchCategories(context.Background(), query, CategoryPageParams{}); err == nil {
			t.Fatalf("query %q was accepted", query)
		}
	}
}
