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

func TestCatalogAndMoney(t *testing.T) {
	t.Parallel()
	var lang, prices string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/api_sellers/api/apilogin" {
			_, _ = io.WriteString(w, `{"retval":0,"token":"tok","valid_thru":"2026-01-02T05:04:05Z"}`)
			return
		}
		if r.Header.Get("Authorization") != "" || !strings.Contains(r.URL.RawQuery, "token=tok") && r.Method == http.MethodGet {
			t.Errorf("auth=%q query=%s", r.Header.Get("Authorization"), r.URL.RawQuery)
		}
		switch r.URL.Path {
		case "/api_sellers/api/categories":
			lang = r.Header.Get("lang")
			if r.Header.Get("locale") != "" {
				t.Error("categories sent locale")
			}
			_, _ = io.WriteString(w, `{"retval":0,"retdesc":"ok","category":[{"id":1,"name":"Игры","sub":[2],"cnt":null}]}`)
		case "/api_sellers/api/products/list":
			_, _ = io.WriteString(w, `{"retval":0,"page":1,"count":10,"has_next_page":false,"has_previous_page":false,"total_count":1,"total_pages":1,"rows":[{"id_goods":8,"name_goods":"Ключ","price":"19.99","price_usd":1.1,"price_eur":1.2,"price_rur":19.99,"price_uah":null,"in_stock":1,"sale_info":{"common_base_price":10.5,"sale_percent":"15"}}]}`)
		case "/api_sellers/api/products/15/data":
			_, _ = io.WriteString(w, `{"retval":0,"product":{"id":15,"price":19.99,"prices":{"initial":{"RUB":19.99}},"options":[{"name":4,"label":"Регион","variants":[]}],"payment_methods":[],"seller":{"id":1,"name":"shop"},"statistics":{"sales":3}}}`)
		case "/api_sellers/api/sellers/account/balance/info":
			_, _ = io.WriteString(w, `{"retval":0,"content":{"amount_t_lock":1.50,"amount_t_free":20.00,"amount_t_plus":null}}`)
		case "/api_sellers/api/sellers/account/receipts":
			if r.URL.Query().Get("type") != "product_sales" || r.URL.Query().Get("currency") != "" {
				t.Errorf("receipts query = %s", r.URL.RawQuery)
			}
			_, _ = io.WriteString(w, `{"retval":0,"content":{"page":1,"count":5,"has_next_page":false,"has_previous_page":false,"total_count":1,"total_pages":1,"items":[{"account_operation_id":9,"operation":{"id":3,"type":"product_sales","datetime":"2026-01-01","percent":5,"price":19.99,"currency":"WMT","on_account":18.00},"product":{"id":8,"name":[{"locale":"ru-RU","value":"Ключ"}],"deleted":false}}]}}`)
		case "/api_sellers/api/product/edit/prices":
			buf, _ := io.ReadAll(r.Body)
			prices = string(buf)
			_, _ = io.WriteString(w, `{"taskId":"task-1"}`)
		case "/api_sellers/api/seller-goods":
			buf, _ := io.ReadAll(r.Body)
			if strings.Contains(string(buf), "owner_id") || !strings.Contains(string(buf), `"order_col":"cntsell"`) {
				t.Errorf("goods body = %s", buf)
			}
			_, _ = io.WriteString(w, `{"retval":0,"id_seller":4,"name_seller":"shop","cnt_goods":1,"pages":1,"page":1,"rows":[]}`)
		default:
			t.Errorf("path = %s", r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	c := newTestClient(t, srv.URL, Options{APIKey: "secret", SellerID: 4, Locale: "en", Clock: func() time.Time { return now }})
	ctx := context.Background()

	categories, err := c.ListCatalogCategories(ctx, ListCatalogCategoriesParams{Page: 1, Count: 10})
	if err != nil || lang != "ru-RU" || categories.Categories[0].Name != "Игры" || categories.Categories[0].Count != nil {
		t.Fatalf("categories lang=%s err=%v page=%+v", lang, err, categories)
	}
	products, err := c.ListCatalogProducts(ctx, ListCatalogProductsParams{Lang: LangEN, Page: 1, Count: 10})
	if err != nil || !products.Products[0].PriceRUR.Equal(testDecimal(t, "19.99")) || products.Products[0].PriceUAH != nil {
		t.Fatalf("products err=%v row=%+v", err, products.Products)
	}
	if products.Products[0].Sale.BasePrice == nil || !products.Products[0].Sale.BasePrice.Equal(testDecimal(t, "10.5")) {
		t.Fatalf("sale = %+v", products.Products[0].Sale)
	}
	card, err := c.GetCatalogProduct(ctx, "15")
	if err != nil || card.Price == nil || !card.Price.Equal(testDecimal(t, "19.99")) || card.Options[0].Name != 4 {
		t.Fatalf("card err=%v %+v", err, card)
	}
	balance, err := c.GetBalance(ctx)
	if err != nil || balance.Free == nil || !balance.Free.Equal(testDecimal(t, "20")) || balance.Plus != nil {
		t.Fatalf("balance err=%v %+v", err, balance)
	}
	kind := ReceiptProductSales
	receipts, err := c.ListReceipts(ctx, ListReceiptsParams{Page: 1, Count: 5, Type: &kind})
	if err != nil || !receipts.Items[0].Operation.Price.Equal(testDecimal(t, "19.99")) || receipts.Items[0].Product.Name[0].Value != "Ключ" {
		t.Fatalf("receipts err=%v %+v", err, receipts)
	}
	rate := testDecimal(t, "10")
	change := PricePercentPlus
	taskID, err := c.UpdateProductPrices(ctx, []ProductPriceChange{{
		ProductID: Ptr(int64(8)),
		Price:     Ptr(testDecimal(t, "19.99")),
		Variants:  []VariantPriceChange{{VariantID: Ptr(int64(3)), Rate: &rate, Type: &change}},
	}})
	if err != nil || taskID != "task-1" {
		t.Fatalf("task=%s err=%v", taskID, err)
	}
	if strings.Contains(prices, `"19.99"`) || !strings.Contains(prices, "19.99") || !strings.HasPrefix(prices, "[") {
		t.Fatalf("prices body = %s", prices)
	}
	order := GoodsOrderSales
	goods, err := c.ListSellerGoods(ctx, SellerGoodsQuery{SellerID: Ptr(int64(4)), Order: &order})
	if err != nil || goods.SellerName != "shop" || goods.Count != 1 {
		t.Fatalf("goods err=%v %+v", err, goods)
	}
}

func TestSellerRetvalAndValidation(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/api_sellers/api/apilogin" {
			_, _ = io.WriteString(w, `{"retval":0,"token":"tok","valid_thru":"2026-01-02T05:04:05Z"}`)
			return
		}
		_, _ = io.WriteString(w, `{"retval":3,"retdesc":"no"}`)
	}))
	t.Cleanup(srv.Close)
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	c := newTestClient(t, srv.URL, Options{APIKey: "secret", SellerID: 4, Clock: func() time.Time { return now }})
	_, err := c.GetBalance(context.Background())
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.RetVal == nil || *apiErr.RetVal != 3 || apiErr.StatusCode != http.StatusOK {
		t.Fatalf("err = %v", err)
	}
	if _, err := c.ListCatalogCategories(context.Background(), ListCatalogCategoriesParams{Page: 0, Count: 10}); err == nil {
		t.Fatal("page 0 passed")
	}
	if _, err := c.UpdateProductPrices(context.Background(), nil); err == nil {
		t.Fatal("empty prices passed")
	}
	bad := PriceChangeType("double")
	if _, err := c.UpdateProductPrices(context.Background(), []ProductPriceChange{{Variants: []VariantPriceChange{{Type: &bad}}}}); err == nil {
		t.Fatal("bad price type passed")
	}
}
