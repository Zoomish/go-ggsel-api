package ggsel

import (
	"context"
	"errors"
	"fmt"
)

// ErrPageLimit — обход остановился: HasNextPage всё ещё true, а maxPages уже выбран.
var ErrPageLimit = errors.New("page limit reached")

// WalkOfferPages вызывает fn для каждой страницы офферов, пока HasNextPage.
// maxPages — сколько страниц можно запросить. Ноль и меньше в сеть не уходят.
// Если после последней разрешённой страницы следующая всё ещё есть, возвращается ErrPageLimit.
// total_pages не используется. Page в params задаёт первую страницу, по умолчанию 1.
func (c *Client) WalkOfferPages(ctx context.Context, params ListOffersParams, maxPages int, fn func(OfferPage) error) error {
	return walkPages(ctx, params.Page, maxPages, func(ctx context.Context, page int64) (OfferPage, Pagination, error) {
		params.Page = &page
		got, err := c.ListOffers(ctx, params)
		if err != nil {
			return OfferPage{}, Pagination{}, err
		}
		return got, got.Pagination, nil
	}, fn)
}

// WalkCategoryPages вызывает fn для каждой страницы категорий v2, пока HasNextPage.
// maxPages ограничивает число запросов так же, как в WalkOfferPages.
func (c *Client) WalkCategoryPages(ctx context.Context, params ListCategoriesParams, maxPages int, fn func(CategoryPage) error) error {
	return walkPages(ctx, params.Page, maxPages, func(ctx context.Context, page int64) (CategoryPage, Pagination, error) {
		params.Page = &page
		got, err := c.ListCategories(ctx, params)
		if err != nil {
			return CategoryPage{}, Pagination{}, err
		}
		return got, got.Pagination, nil
	}, fn)
}

// WalkCategorySearch вызывает fn для каждой страницы поиска категорий, пока HasNextPage.
// maxPages ограничивает число запросов так же, как в WalkOfferPages.
func (c *Client) WalkCategorySearch(ctx context.Context, query string, params CategoryPageParams, maxPages int, fn func(CategoryPage) error) error {
	return walkPages(ctx, params.Page, maxPages, func(ctx context.Context, page int64) (CategoryPage, Pagination, error) {
		params.Page = &page
		got, err := c.SearchCategories(ctx, query, params)
		if err != nil {
			return CategoryPage{}, Pagination{}, err
		}
		return got, got.Pagination, nil
	}, fn)
}

func walkPages[T any](ctx context.Context, start *int64, maxPages int, fetch func(context.Context, int64) (T, Pagination, error), fn func(T) error) error {
	if fn == nil {
		return fmt.Errorf("ggsel: walk pages: fn is required")
	}
	if maxPages <= 0 {
		return fmt.Errorf("ggsel: walk pages: max pages must be positive")
	}
	page := int64(1)
	if start != nil {
		if *start <= 0 {
			return fmt.Errorf("ggsel: walk pages: page must be positive")
		}
		page = *start
	}
	seen := 0
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		item, pagination, err := fetch(ctx, page)
		if err != nil {
			return err
		}
		seen++
		if err := fn(item); err != nil {
			return err
		}
		if !pagination.HasNextPage {
			return nil
		}
		if seen >= maxPages {
			return fmt.Errorf("ggsel: walk pages: stopped after %d pages: %w", seen, ErrPageLimit)
		}
		page++
	}
}
