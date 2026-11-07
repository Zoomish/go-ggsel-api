package ggsel

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

var errEmptyData = errors.New("empty data")

// OfferStatus — статус оффера в карточке и списке.
type OfferStatus string

// Значения статуса оффера.
const (
	OfferStatusDraft    OfferStatus = "draft"
	OfferStatusActive   OfferStatus = "active"
	OfferStatusPaused   OfferStatus = "paused"
	OfferStatusArchived OfferStatus = "archived"
)

// OfferListStatus — фильтр списка. Archived в этот фильтр не входит.
type OfferListStatus string

// Значения фильтра списка. Archived сюда не входит.
const (
	OfferListStatusActive OfferListStatus = "active"
	OfferListStatusPaused OfferListStatus = "paused"
	OfferListStatusDraft  OfferListStatus = "draft"
)

// Delivery — способ выдачи оффера.
type Delivery string

// Значения способа выдачи.
const (
	DeliveryAuto   Delivery = "auto"
	DeliveryManual Delivery = "manual"
)

// Currency — валюта цены оффера v2. В спецификации это RUB.
type Currency string

// Валюта цены v2.
const (
	CurrencyRUB Currency = "RUB"
)

// OfferSort — сортировка списка офферов.
type OfferSort string

// Варианты сортировки списка офферов.
const (
	OfferSortPriceAsc          OfferSort = "price_asc"
	OfferSortPriceDesc         OfferSort = "price_desc"
	OfferSortSalesCountAsc     OfferSort = "sales_count_asc"
	OfferSortSalesCountDesc    OfferSort = "sales_count_desc"
	OfferSortProductsCountAsc  OfferSort = "products_count_asc"
	OfferSortProductsCountDesc OfferSort = "products_count_desc"
)

// NotificationType — канал уведомления продавца о заказе.
type NotificationType string

// Каналы уведомления о заказе.
const (
	NotificationEmail NotificationType = "email"
	NotificationURL   NotificationType = "url"
)

// NotificationMethod — HTTP-метод уведомления.
type NotificationMethod string

// HTTP-методы уведомления.
const (
	NotificationMethodGET  NotificationMethod = "GET"
	NotificationMethodPOST NotificationMethod = "POST"
)

// Pagination — страница v2, включая устаревшие total_pages и total_count.
type Pagination struct {
	Page            int64  `json:"page"`
	Limit           int64  `json:"limit"`
	HasNextPage     bool   `json:"has_next_page"`
	HasPreviousPage bool   `json:"has_previous_page"`
	TotalPages      int64  `json:"total_pages"`
	TotalCount      *int64 `json:"total_count"`
}

// PrePaymentSettings — проверка до оплаты.
type PrePaymentSettings struct {
	IsEnabled    bool    `json:"is_enabled"`
	URL          *string `json:"url"`
	AllowPayment bool    `json:"allow_payment"`
}

// NotificationSettings — уведомление продавца о новом заказе.
type NotificationSettings struct {
	Type       *NotificationType   `json:"type"`
	URL        *string             `json:"url"`
	Email      *string             `json:"email"`
	HTTPMethod *NotificationMethod `json:"http_method"`
	IsDisabled bool                `json:"is_disabled"`
	IsDefault  bool                `json:"is_default"`
}

// OfferSummary — элемент списка офферов.
type OfferSummary struct {
	ID                   int64                 `json:"id"`
	Status               OfferStatus           `json:"status"`
	TitleRU              *string               `json:"title_ru"`
	TitleEN              *string               `json:"title_en"`
	DescriptionRU        *string               `json:"description_ru"`
	DescriptionEN        *string               `json:"description_en"`
	Price                *Decimal              `json:"price"`
	Currency             Currency              `json:"currency"`
	IsAutoselling        *bool                 `json:"is_autoselling"`
	Category             Category              `json:"category"`
	MinQuantity          *int64                `json:"min_quantity"`
	MaxQuantity          *int64                `json:"max_quantity"`
	Quantity             *int64                `json:"quantity"`
	IsUnlimitedQuantity  *bool                 `json:"is_unlimited_quantity"`
	InstructionsRU       *string               `json:"instructions_ru"`
	InstructionsEN       *string               `json:"instructions_en"`
	PostPaymentURL       *string               `json:"post_payment_url"`
	Delivery             *Delivery             `json:"delivery"`
	CoverImageRUURL      string                `json:"cover_image_ru_url"`
	CoverImageENURL      *string               `json:"cover_image_en_url"`
	HasOptions           bool                  `json:"has_options"`
	HasProducts          bool                  `json:"has_products"`
	HasSplittedProducts  bool                  `json:"has_splitted_products"`
	CreatedAt            string                `json:"created_at"`
	UpdatedAt            string                `json:"updated_at"`
	PrePaymentSettings   *PrePaymentSettings   `json:"pre_payment_settings"`
	NotificationSettings *NotificationSettings `json:"notification_settings"`
}

// Offer — карточка оффера. Её же возвращают создание и обновление.
type Offer struct {
	ID                           int64                 `json:"id"`
	Status                       OfferStatus           `json:"status"`
	IsAutoselling                bool                  `json:"is_autoselling"`
	Delivery                     *Delivery             `json:"delivery"`
	TitleRU                      string                `json:"title_ru"`
	TitleEN                      string                `json:"title_en"`
	DescriptionRU                string                `json:"description_ru"`
	DescriptionEN                string                `json:"description_en"`
	InstructionsRU               string                `json:"instructions_ru"`
	InstructionsEN               string                `json:"instructions_en"`
	CoverImageRUURL              string                `json:"cover_image_ru_url"`
	CoverImageENURL              *string               `json:"cover_image_en_url"`
	Category                     Category              `json:"category"`
	Price                        Decimal               `json:"price"`
	Currency                     Currency              `json:"currency"`
	Quantity                     int64                 `json:"quantity"`
	IsUnlimitedQuantity          bool                  `json:"is_unlimited_quantity"`
	MinQuantity                  *int64                `json:"min_quantity"`
	MaxQuantity                  *int64                `json:"max_quantity"`
	SoldSplittedProductsCount    int64                 `json:"sold_splitted_products_count"`
	SoldProductsCount            int64                 `json:"sold_products_count"`
	InStockProductsCount         int64                 `json:"in_stock_products_count"`
	InStockSplittedProductsCount int64                 `json:"in_stock_splitted_products_count"`
	HasSplittedProducts          bool                  `json:"has_splitted_products"`
	UpdatedAt                    string                `json:"updated_at"`
	CreatedAt                    string                `json:"created_at"`
	PostPaymentURL               *string               `json:"post_payment_url"`
	PrePaymentSettings           *PrePaymentSettings   `json:"pre_payment_settings"`
	NotificationSettings         *NotificationSettings `json:"notification_settings"`
}

// OfferPage — одна страница списка офферов.
type OfferPage struct {
	Offers     []OfferSummary
	Pagination Pagination
}

// ListOffersParams — фильтр списка. Нулевые указатели в запрос не попадают.
type ListOffersParams struct {
	Page          *int64
	Limit         *int64
	Search        *string
	CategoryID    *int64
	Status        *OfferListStatus
	Sort          *OfferSort
	Delivery      *Delivery
	UpdatedAtFrom *time.Time
}

// ListOffers возвращает одну страницу офферов.
func (c *Client) ListOffers(ctx context.Context, params ListOffersParams) (OfferPage, error) {
	query, err := params.query()
	if err != nil {
		return OfferPage{}, err
	}
	var env pageEnvelope
	if err := c.do(ctx, call{
		method: http.MethodGet,
		path:   "/api_sellers/v2/offers",
		query:  query,
		header: c.localeHeader(),
		dest:   &env,
		auth:   authV2,
		retry:  true,
	}); err != nil {
		return OfferPage{}, err
	}
	var items []OfferSummary
	if err := decodeData(env.Data, &items); err != nil {
		return OfferPage{}, fmt.Errorf("ggsel: list offers: %w", err)
	}
	if items == nil {
		items = []OfferSummary{}
	}
	return OfferPage{Offers: items, Pagination: env.Pagination}, nil
}

// GetOffer возвращает карточку оффера.
func (c *Client) GetOffer(ctx context.Context, id int64) (Offer, error) {
	if id <= 0 {
		return Offer{}, fmt.Errorf("ggsel: get offer: id must be positive")
	}
	var env objectEnvelope
	if err := c.do(ctx, call{
		method: http.MethodGet,
		path:   "/api_sellers/v2/offers/" + strconv.FormatInt(id, 10),
		header: c.localeHeader(),
		dest:   &env,
		auth:   authV2,
		retry:  true,
	}); err != nil {
		return Offer{}, err
	}
	var offer Offer
	if err := decodeData(env.Data, &offer); err != nil {
		return Offer{}, fmt.Errorf("ggsel: get offer: %w", err)
	}
	return offer, nil
}

func (p ListOffersParams) query() (url.Values, error) {
	q := url.Values{}
	setInt(q, "page", p.Page)
	setInt(q, "limit", p.Limit)
	setString(q, "search", p.Search)
	setInt(q, "category_id", p.CategoryID)
	if p.Status != nil {
		if !validOfferListStatus(*p.Status) {
			return nil, fmt.Errorf("ggsel: list offers: invalid status %q", *p.Status)
		}
		q.Set("status", string(*p.Status))
	}
	if p.Sort != nil {
		if !validOfferSort(*p.Sort) {
			return nil, fmt.Errorf("ggsel: list offers: invalid sort %q", *p.Sort)
		}
		q.Set("sort", string(*p.Sort))
	}
	if p.Delivery != nil {
		if !validDelivery(*p.Delivery) {
			return nil, fmt.Errorf("ggsel: list offers: invalid delivery %q", *p.Delivery)
		}
		q.Set("delivery", string(*p.Delivery))
	}
	if p.UpdatedAtFrom != nil {
		q.Set("updated_at_from", p.UpdatedAtFrom.Format(time.RFC3339))
	}
	return q, nil
}

func validOfferListStatus(s OfferListStatus) bool {
	switch s {
	case OfferListStatusActive, OfferListStatusPaused, OfferListStatusDraft:
		return true
	default:
		return false
	}
}

func validOfferSort(s OfferSort) bool {
	switch s {
	case OfferSortPriceAsc, OfferSortPriceDesc, OfferSortSalesCountAsc, OfferSortSalesCountDesc, OfferSortProductsCountAsc, OfferSortProductsCountDesc:
		return true
	default:
		return false
	}
}

func validDelivery(d Delivery) bool {
	switch d {
	case DeliveryAuto, DeliveryManual:
		return true
	default:
		return false
	}
}

func (c *Client) localeHeader() http.Header {
	h := make(http.Header)
	h.Set("locale", c.locale)
	return h
}

type pageEnvelope struct {
	Data       json.RawMessage `json:"data"`
	Pagination Pagination      `json:"pagination"`
}

type objectEnvelope struct {
	Data json.RawMessage `json:"data"`
}

func decodeData(raw json.RawMessage, dest any) error {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		return errEmptyData
	}
	if err := json.Unmarshal(trimmed, dest); err != nil {
		return fmt.Errorf("decode data: %w", err)
	}
	return nil
}

func setInt(q url.Values, key string, v *int64) {
	if v != nil {
		q.Set(key, strconv.FormatInt(*v, 10))
	}
}

func setString(q url.Values, key string, v *string) {
	if v != nil {
		q.Set(key, *v)
	}
}
