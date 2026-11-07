package ggsel

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// Category — категория v2. Fee и PaymentFee — доли комиссии, не float64.
type Category struct {
	ID          int64   `json:"id"`
	Title       string  `json:"title"`
	ContentType string  `json:"content_type"`
	Fee         Decimal `json:"fee"`
	PaymentFee  Decimal `json:"payment_fee"`
	Tree        string  `json:"tree"`
	HasChildren bool    `json:"has_children"`
}

// CategoryPage — одна страница категорий.
type CategoryPage struct {
	Categories []Category
	Pagination Pagination
}

// CategoryPageParams — page и limit. Нулевые указатели в запрос не попадают.
type CategoryPageParams struct {
	Page  *int64
	Limit *int64
}

// ListCategoriesParams — фильтр списка категорий.
type ListCategoriesParams struct {
	ParentID *int64
	Page     *int64
	Limit    *int64
}

// ListCategories возвращает одну страницу категорий.
func (c *Client) ListCategories(ctx context.Context, params ListCategoriesParams) (CategoryPage, error) {
	q := url.Values{}
	setInt(q, "parent_id", params.ParentID)
	setInt(q, "page", params.Page)
	setInt(q, "limit", params.Limit)
	return c.categoryPage(ctx, "list categories", "/api_sellers/v2/categories", q)
}

// SearchCategories ищет категории по q. Пустой запрос не уходит в сеть.
func (c *Client) SearchCategories(ctx context.Context, query string, params CategoryPageParams) (CategoryPage, error) {
	if strings.TrimSpace(query) == "" {
		return CategoryPage{}, fmt.Errorf("ggsel: search categories: query is required")
	}
	q := url.Values{}
	q.Set("q", query)
	setInt(q, "page", params.Page)
	setInt(q, "limit", params.Limit)
	return c.categoryPage(ctx, "search categories", "/api_sellers/v2/categories/search", q)
}

func (c *Client) categoryPage(ctx context.Context, op, path string, query url.Values) (CategoryPage, error) {
	var env pageEnvelope
	if err := c.do(ctx, call{
		method: http.MethodGet,
		path:   path,
		query:  query,
		header: c.localeHeader(),
		dest:   &env,
		auth:   authV2,
		retry:  true,
	}); err != nil {
		return CategoryPage{}, err
	}
	var items []Category
	if err := decodeData(env.Data, &items); err != nil {
		return CategoryPage{}, fmt.Errorf("ggsel: %s: %w", op, err)
	}
	if items == nil {
		items = []Category{}
	}
	return CategoryPage{Categories: items, Pagination: env.Pagination}, nil
}
