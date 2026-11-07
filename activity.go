package ggsel

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
)

// ReviewFilter — фильтр отзывов v1.
type ReviewFilter string

const (
	ReviewGood ReviewFilter = "good"
	ReviewBad  ReviewFilter = "bad"
	ReviewAll  ReviewFilter = "all"
)

// Review — отзыв v1. InvoiceID в снимке number, дата остаётся строкой.
type Review struct {
	ID        int64   `json:"id"`
	Info      string  `json:"info"`
	Good      int64   `json:"good"`
	Type      string  `json:"type"`
	Date      string  `json:"date"`
	InvoiceID Decimal `json:"invoice_id"`
	Name      string  `json:"name"`
	Comment   *string `json:"comment"`
	OwnerID   int64   `json:"owner_id"`
}

// ReviewPage — страница отзывов.
type ReviewPage struct {
	TotalPages int64
	TotalItems int64
	TotalGood  int64
	TotalBad   int64
	Reviews    []Review
}

// ListReviewsParams — фильтр отзывов. Page и Count обязательны.
// Lang пустой становится ru-RU: заголовок locale здесь не совпадает с locale клиента.
type ListReviewsParams struct {
	Lang      Lang
	Type      *ReviewFilter
	Page      int64
	Count     int64
	ProductID *int64
}

// LastSaleProduct — товар в последней продаже.
type LastSaleProduct struct {
	ID       int64    `json:"id"`
	Name     string   `json:"name"`
	PriceRUB Decimal  `json:"price_rub"`
	PriceUSD Decimal  `json:"price_usd"`
	PriceEUR Decimal  `json:"price_eur"`
	PriceUAH *Decimal `json:"price_uah"`
}

// LastSale — одна из последних продаж.
type LastSale struct {
	InvoiceID int64           `json:"invoice_id"`
	Date      string          `json:"date"`
	Product   LastSaleProduct `json:"product"`
}

// ListLastSalesParams — фильтр последних продаж. Нулевые указатели в запрос не попадают.
type ListLastSalesParams struct {
	SellerID *int64
	Group    *bool
	Top      *int64
}

func (c *Client) ListReviews(ctx context.Context, params ListReviewsParams) (ReviewPage, error) {
	lang, err := requiredLang("list reviews", params.Lang)
	if err != nil {
		return ReviewPage{}, err
	}
	if err := requirePage("list reviews", params.Page, params.Count); err != nil {
		return ReviewPage{}, err
	}
	if params.Type != nil && !known(*params.Type, ReviewGood, ReviewBad, ReviewAll) {
		return ReviewPage{}, fmt.Errorf("ggsel: list reviews: invalid type %q", *params.Type)
	}
	if params.ProductID != nil && *params.ProductID <= 0 {
		return ReviewPage{}, fmt.Errorf("ggsel: list reviews: id must be positive")
	}
	q := url.Values{}
	q.Set("page", strconv.FormatInt(params.Page, 10))
	q.Set("count", strconv.FormatInt(params.Count, 10))
	if params.Type != nil {
		q.Set("type", string(*params.Type))
	}
	setInt(q, "product_id", params.ProductID)
	header := make(http.Header)
	header.Set("locale", string(lang))
	var env struct {
		TotalPages int64    `json:"totalPages"`
		TotalItems int64    `json:"totalItems"`
		TotalGood  int64    `json:"totalGood"`
		TotalBad   int64    `json:"totalBad"`
		Reviews    []Review `json:"reviews"`
	}
	if err := c.do(ctx, call{
		method: http.MethodGet,
		path:   "/api_sellers/api/reviews",
		query:  q,
		header: header,
		dest:   &env,
		auth:   authV1,
		retry:  true,
	}); err != nil {
		return ReviewPage{}, err
	}
	if env.Reviews == nil {
		env.Reviews = []Review{}
	}
	return ReviewPage{
		TotalPages: env.TotalPages,
		TotalItems: env.TotalItems,
		TotalGood:  env.TotalGood,
		TotalBad:   env.TotalBad,
		Reviews:    env.Reviews,
	}, nil
}

// ListLastSales возвращает последние продажи. Заголовок locale берётся у клиента: в снимке значение по умолчанию ru.
func (c *Client) ListLastSales(ctx context.Context, params ListLastSalesParams) ([]LastSale, error) {
	if params.SellerID != nil && *params.SellerID <= 0 {
		return nil, fmt.Errorf("ggsel: list last sales: id must be positive")
	}
	if params.Top != nil && *params.Top <= 0 {
		return nil, fmt.Errorf("ggsel: list last sales: top must be positive")
	}
	q := url.Values{}
	setInt(q, "seller_id", params.SellerID)
	if params.Group != nil {
		q.Set("group", strconv.FormatBool(*params.Group))
	}
	setInt(q, "top", params.Top)
	var env struct {
		Sales []LastSale `json:"sales"`
	}
	if err := c.do(ctx, call{
		method: http.MethodGet,
		path:   "/api_sellers/api/seller-last-sales",
		query:  q,
		header: c.localeHeader(),
		dest:   &env,
		auth:   authV1,
		retry:  true,
	}); err != nil {
		return nil, err
	}
	if env.Sales == nil {
		env.Sales = []LastSale{}
	}
	return env.Sales, nil
}
