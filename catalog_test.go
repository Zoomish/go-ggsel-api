package ggsel

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestListOfferOptions(t *testing.T) {
	t.Parallel()
	const body = `{
		"data":[{
			"id":7,
			"status":"active",
			"type":"radio_button",
			"title_ru":"Регион",
			"title_en":null,
			"is_required":true,
			"is_price_modifier_hidden":false,
			"has_splitted_products":false,
			"position":1,
			"variants":[{
				"id":3,
				"title_ru":"EU",
				"title_en":null,
				"price":12.50,
				"discount_type":"fixed",
				"impact_type":"increase",
				"is_default":false,
				"status":"hidden",
				"position":2
			}]
		}],
		"pagination":{"page":1,"limit":100,"has_next_page":false,"has_previous_page":false,"total_pages":1,"total_count":1}
	}`
	var auth, locale string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth = r.Header.Get("Authorization")
		locale = r.Header.Get("locale")
		if r.URL.Path != "/api_sellers/v2/offers/9/options" || r.URL.RawQuery != "" {
			t.Errorf("url = %s?%s", r.URL.Path, r.URL.RawQuery)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, body)
	}))
	t.Cleanup(srv.Close)
	c := newTestClient(t, srv.URL, Options{APIKey: "seller-key", Locale: "en"})
	page, err := c.ListOfferOptions(context.Background(), 9)
	if err != nil {
		t.Fatalf("ListOfferOptions: %v", err)
	}
	if auth != "seller-key" || locale != "en" {
		t.Fatalf("auth=%q locale=%q", auth, locale)
	}
	if len(page.Options) != 1 || page.Options[0].Type != OfferOptionRadioButton || page.Options[0].TitleEN != nil {
		t.Fatalf("option = %+v", page.Options)
	}
	variant := page.Options[0].Variants[0]
	if !variant.Price.Equal(testDecimal(t, "12.5")) || variant.InStock != nil || variant.Status != OfferOptionHidden {
		t.Fatalf("variant = %+v", variant)
	}
}

func TestCreateOfferOptionsOmitsEmpty(t *testing.T) {
	t.Parallel()
	var raw string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf, _ := io.ReadAll(r.Body)
		raw = string(buf)
		if r.Method != http.MethodPost {
			t.Errorf("method = %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"data":[]}`)
	}))
	t.Cleanup(srv.Close)
	c := newTestClient(t, srv.URL, Options{APIKey: "k"})
	kind := OfferOptionText
	_, err := c.CreateOfferOptions(context.Background(), 4, []OfferOptionWrite{{
		Type:    &kind,
		TitleRU: Ptr("Цвет"),
	}})
	if err != nil {
		t.Fatalf("CreateOfferOptions: %v", err)
	}
	if strings.Contains(raw, "title_en") || strings.Contains(raw, "status") || !strings.Contains(raw, `"type":"text"`) {
		t.Fatalf("body = %s", raw)
	}
}

func TestArchiveOfferVariantsReturnsJob(t *testing.T) {
	t.Parallel()
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Method != http.MethodDelete || r.URL.Path != "/api_sellers/v2/offers/4/options/8/variants" {
			t.Errorf("%s %s", r.Method, r.URL.Path)
		}
		buf, _ := io.ReadAll(r.Body)
		if string(buf) != `{"option_variant_ids":[3],"delete_all":"false"}` {
			t.Errorf("body = %s", buf)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"success":true,"job_id":"job-1"}`)
	}))
	t.Cleanup(srv.Close)
	c := newTestClient(t, srv.URL, Options{APIKey: "k"})
	all := DeleteAllFalse
	job, err := c.ArchiveOfferVariants(context.Background(), 4, 8, VariantArchive{
		VariantIDs: []int64{3},
		DeleteAll:  &all,
	})
	if err != nil {
		t.Fatalf("ArchiveOfferVariants: %v", err)
	}
	if calls != 1 || !job.Success || job.ID != "job-1" {
		t.Fatalf("calls=%d job=%+v", calls, job)
	}
}

func TestCreateOfferProductsNoContent(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf, _ := io.ReadAll(r.Body)
		if string(buf) != `{"products":[{"value":"KEY-1"}]}` {
			t.Errorf("body = %s", buf)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(srv.Close)
	c := newTestClient(t, srv.URL, Options{APIKey: "k"})
	if err := c.CreateOfferProducts(context.Background(), 5, []OfferProductInput{{Value: "KEY-1"}}); err != nil {
		t.Fatal(err)
	}
	if err := c.CreateSplittedProducts(context.Background(), 0, 1, []OfferProductInput{{Value: "x"}}); err == nil {
		t.Fatal("expected id error")
	}
}

func TestListOfferProductsQuery(t *testing.T) {
	t.Parallel()
	status := OfferProductInStock
	column := "created_at"
	var query string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"data":[{"id":1,"value":"k","status":"in_stock","created_at":"2026-01-01"}],"pagination":{"page":1,"limit":100,"has_next_page":false,"has_previous_page":false,"total_pages":0,"total_count":null}}`)
	}))
	t.Cleanup(srv.Close)
	c := newTestClient(t, srv.URL, Options{APIKey: "k"})
	page, err := c.ListOfferProducts(context.Background(), 6, ListOfferProductsParams{Status: &status, SortColumn: &column})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(query, "status=in_stock") || !strings.Contains(query, "sort_column=created_at") || strings.Contains(query, "sort_direction") {
		t.Fatalf("query = %s", query)
	}
	if page.Products[0].Status != OfferProductInStock || page.Pagination.TotalCount != nil {
		t.Fatalf("page = %+v", page)
	}
	archived := OfferProductArchived
	if _, err := c.ListSplittedProducts(context.Background(), 6, 2, ListOfferProductsParams{Status: &archived}); err == nil {
		t.Fatal("archived filter passed")
	}
}

func TestCatalogRejectsBadInput(t *testing.T) {
	t.Parallel()
	c := newTestClient(t, "http://127.0.0.1:1", Options{APIKey: "k"})
	if _, err := c.GetOfferOption(context.Background(), 1, 0); err == nil {
		t.Fatal("option id")
	}
	if _, err := c.UpdateOfferOptionVisibility(context.Background(), 1, []OptionVisibility{{ID: 1, Status: OfferOptionArchived}}); err == nil {
		t.Fatal("archived visibility")
	}
	bad := DeleteAll("yes")
	if _, err := c.ArchiveOfferOptions(context.Background(), 1, OptionArchive{DeleteAll: &bad}); err == nil {
		t.Fatal("delete_all")
	}
	if _, err := c.ArchiveOfferProducts(context.Background(), 1, ProductArchive{}); err == nil {
		t.Fatal("empty archive")
	}
}
