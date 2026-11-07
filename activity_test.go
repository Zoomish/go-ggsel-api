package ggsel

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestReviewsAndSales(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/api_sellers/api/apilogin" {
			_, _ = io.WriteString(w, `{"retval":0,"token":"tok","valid_thru":"2026-01-02T05:04:05Z"}`)
			return
		}
		switch r.URL.Path {
		case "/api_sellers/api/reviews":
			if r.Header.Get("locale") != "ru-RU" || r.URL.Query().Get("type") != "good" {
				t.Errorf("locale=%q query=%s", r.Header.Get("locale"), r.URL.RawQuery)
			}
			_, _ = io.WriteString(w, `{"retval":0,"totalPages":1,"totalItems":1,"totalGood":1,"totalBad":0,"reviews":[{"id":5,"info":"ok","good":1,"type":"good","date":"2026-01-01","invoice_id":1001.0,"name":"buyer","comment":null,"owner_id":4}]}`)
		case "/api_sellers/api/seller-last-sales":
			if r.Header.Get("locale") != "en" || r.URL.Query().Get("group") != "false" || r.URL.Query().Get("top") != "" {
				t.Errorf("locale=%q query=%s", r.Header.Get("locale"), r.URL.RawQuery)
			}
			_, _ = io.WriteString(w, `{"retval":0,"sales":[{"invoice_id":9,"date":"2026-01-01","product":{"id":8,"name":"Ключ","price_rub":19.99,"price_usd":1,"price_eur":1,"price_uah":null}}]}`)
		default:
			t.Errorf("path = %s", r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	c := newTestClient(t, srv.URL, Options{APIKey: "secret", SellerID: 4, Locale: "en", Clock: func() time.Time { return now }})
	kind := ReviewGood
	page, err := c.ListReviews(context.Background(), ListReviewsParams{Type: &kind, Page: 1, Count: 10})
	if err != nil || !page.Reviews[0].InvoiceID.Equal(testDecimal(t, "1001")) || page.Reviews[0].Comment != nil {
		t.Fatalf("reviews err=%v %+v", err, page)
	}
	group := false
	sales, err := c.ListLastSales(context.Background(), ListLastSalesParams{Group: &group})
	if err != nil || !sales[0].Product.PriceRUB.Equal(testDecimal(t, "19.99")) || sales[0].Product.PriceUAH != nil {
		t.Fatalf("sales err=%v %+v", err, sales)
	}
}
