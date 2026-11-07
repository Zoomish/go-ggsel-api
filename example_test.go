package ggsel_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/Zoomish/go-ggsel-api"
)

func ExampleNew() {
	if _, err := ggsel.New(ggsel.Options{}); err == nil {
		panic("empty key")
	}
	client, err := ggsel.New(ggsel.Options{APIKey: "test-key"})
	if err != nil || client == nil {
		panic(err)
	}
	// Output:
}

func ExampleClient_ListOffers() {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{
			"data":[{
				"id":1,
				"status":"active",
				"price":19.99,
				"currency":"RUB",
				"category":{"id":3,"title":"Games","content_type":"key","fee":0,"payment_fee":0,"tree":"Games","has_children":false},
				"has_options":false,
				"has_products":false,
				"has_splitted_products":false,
				"cover_image_ru_url":""
			}],
			"pagination":{"page":1,"limit":100,"has_next_page":false,"has_previous_page":false}
		}`)
	}))
	defer srv.Close()

	client, err := ggsel.New(ggsel.Options{APIKey: "test-key", BaseURL: srv.URL})
	if err != nil {
		panic(err)
	}
	page, err := client.ListOffers(context.Background(), ggsel.ListOffersParams{})
	if err != nil {
		panic(err)
	}
	if len(page.Offers) != 1 || page.Offers[0].Price == nil || page.Offers[0].Price.String() != "19.99" {
		panic(page.Offers)
	}
	// Output:
}

func ExampleClient_UpdateOffer() {
	var raw string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf, err := io.ReadAll(r.Body)
		if err != nil {
			panic(err)
		}
		raw = string(buf)
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{
			"data":{
				"id":15,"status":"draft","is_autoselling":false,
				"title_ru":"","title_en":"","description_ru":"","description_en":"",
				"instructions_ru":"","instructions_en":"","cover_image_ru_url":"",
				"category":{"id":1,"title":"","content_type":"","fee":0,"payment_fee":0,"tree":"","has_children":false},
				"price":19.99,"currency":"RUB","quantity":0,"is_unlimited_quantity":false,
				"sold_splitted_products_count":0,"sold_products_count":0,
				"in_stock_products_count":0,"in_stock_splitted_products_count":0,
				"has_splitted_products":false,"updated_at":"","created_at":""
			}
		}`)
	}))
	defer srv.Close()

	client, err := ggsel.New(ggsel.Options{APIKey: "test-key", BaseURL: srv.URL})
	if err != nil {
		panic(err)
	}
	price, err := ggsel.ParseDecimal("19.99")
	if err != nil {
		panic(err)
	}
	if _, err := client.UpdateOffer(context.Background(), 15, ggsel.OfferWrite{
		IsAutoselling: ggsel.Ptr(false),
		Price:         &price,
	}); err != nil {
		panic(err)
	}
	if raw != `{"price":19.99,"is_autoselling":false}` || strings.Contains(raw, "title_ru") {
		panic(raw)
	}
	// Output:
}

func ExampleClient_GetOrder() {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api_sellers/api/apilogin" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"retval":0,"token":"tok","valid_thru":"2099-01-01T00:00:00Z"}`)
			return
		}
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = io.WriteString(w, `{"retval":-1,"retdesc":"no"}`)
	}))
	defer srv.Close()

	client, err := ggsel.New(ggsel.Options{
		APIKey:   "test-key",
		SellerID: 42,
		BaseURL:  srv.URL,
	})
	if err != nil {
		panic(err)
	}
	_, err = client.GetOrder(context.Background(), 15)
	var apiErr *ggsel.APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusUnauthorized {
		panic(err)
	}
	// Output:
}
