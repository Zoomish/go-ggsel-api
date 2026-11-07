package ggsel

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWalkOfferPagesUsesHasNext(t *testing.T) {
	t.Parallel()
	var pages []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pages = append(pages, r.URL.Query().Get("page"))
		w.Header().Set("Content-Type", "application/json")
		hasNext := "true"
		if len(pages) == 2 {
			hasNext = "false"
		}
		_, _ = io.WriteString(w, `{"data":[],"pagination":{"page":1,"limit":100,"has_next_page":`+hasNext+`,"has_previous_page":false,"total_pages":1,"total_count":0}}`)
	}))
	t.Cleanup(srv.Close)
	c := newTestClient(t, srv.URL, Options{APIKey: "k"})
	start := int64(1)
	calls := 0
	err := c.WalkOfferPages(context.Background(), ListOffersParams{Page: &start}, 5, func(OfferPage) error {
		calls++
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if calls != 2 || len(pages) != 2 || pages[0] != "1" || pages[1] != "2" {
		t.Fatalf("calls=%d pages=%v", calls, pages)
	}
	if start != 1 {
		t.Fatalf("caller page changed to %d", start)
	}
	stop := errors.New("stop")
	err = c.WalkCategoryPages(context.Background(), ListCategoriesParams{}, 1, func(CategoryPage) error {
		return stop
	})
	if !errors.Is(err, stop) {
		t.Fatalf("walk error = %v", err)
	}
	if err := c.WalkCategorySearch(context.Background(), "q", CategoryPageParams{}, 1, nil); err == nil {
		t.Fatal("nil fn passed")
	}
}

func TestWalkOfferPagesStopsAtLimit(t *testing.T) {
	t.Parallel()
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"data":[],"pagination":{"page":1,"limit":100,"has_next_page":true,"has_previous_page":false,"total_pages":1,"total_count":0}}`)
	}))
	t.Cleanup(srv.Close)
	c := newTestClient(t, srv.URL, Options{APIKey: "k"})
	err := c.WalkOfferPages(context.Background(), ListOffersParams{}, 2, func(OfferPage) error { return nil })
	if !errors.Is(err, ErrPageLimit) {
		t.Fatalf("error = %v", err)
	}
	if calls != 2 {
		t.Fatalf("calls = %d", calls)
	}
	calls = 0
	if err := c.WalkOfferPages(context.Background(), ListOffersParams{}, 0, func(OfferPage) error { return nil }); err == nil || calls != 0 {
		t.Fatalf("maxPages 0: calls=%d err=%v", calls, err)
	}
}
