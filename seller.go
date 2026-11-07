package ggsel

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// Lang — язык заголовков v1. Это ru-RU и en-US, не locale клиента.
type Lang string

const (
	LangRU Lang = "ru-RU"
	LangEN Lang = "en-US"
)

// GoodsCurrency — валюта списка товаров продавца.
type GoodsCurrency string

const (
	GoodsCurrencyUSD GoodsCurrency = "USD"
	GoodsCurrencyEUR GoodsCurrency = "EUR"
	GoodsCurrencyRUB GoodsCurrency = "RUB"
)

// GoodsOrder — колонка сортировки товаров продавца.
type GoodsOrder string

const (
	GoodsOrderSales GoodsOrder = "cntsell"
)

// SortDirection — направление сортировки.
type SortDirection string

const (
	SortAsc  SortDirection = "asc"
	SortDesc SortDirection = "desc"
)

// PriceChangeType — как массовая правка меняет цену варианта.
type PriceChangeType string

const (
	PricePercentPlus  PriceChangeType = "percentplus"
	PricePercentMinus PriceChangeType = "percentminus"
	PriceMinus        PriceChangeType = "priceminus"
	PricePlus         PriceChangeType = "priceplus"
)

// ReceiptType — тип операции в чеках.
type ReceiptType string

const (
	ReceiptAgentAccruals      ReceiptType = "agent_accurals"
	ReceiptProductSales       ReceiptType = "product_sales"
	ReceiptAddFunds           ReceiptType = "add_funds"
	ReceiptExchangeResponse   ReceiptType = "exchange_response"
	ReceiptExchangeRequest    ReceiptType = "exchange_request"
	ReceiptRefund             ReceiptType = "refund"
	ReceiptAdvGoods           ReceiptType = "adv_goods"
	ReceiptExternalCommission ReceiptType = "external_commissions"
	ReceiptDiskRent           ReceiptType = "hard_disk_rent"
	ReceiptPartnerSpace       ReceiptType = "extra_partner_space"
	ReceiptGiftCertificates   ReceiptType = "gift_certificates"
	ReceiptWalletTransfer     ReceiptType = "transfer_to_wallet"
)

// ReceiptCodeFilter — фильтр проверки кода в чеках.
type ReceiptCodeFilter string

const (
	ReceiptOnlyWaitingCode ReceiptCodeFilter = "only_waiting_code_check"
	ReceiptHideWaitingCode ReceiptCodeFilter = "hide_waiting_code_check"
)

// CatalogCategory — категория v1.
type CatalogCategory struct {
	ID       int64   `json:"id"`
	Name     string  `json:"name"`
	Children []int64 `json:"sub"`
	Count    *int64  `json:"cnt"`
}

// CatalogCategoryPage — страница категорий v1.
type CatalogCategoryPage struct {
	Categories []CatalogCategory
}

// ListCatalogCategoriesParams — страница категорий. Page и Count обязательны.
type ListCatalogCategoriesParams struct {
	Lang       Lang
	Page       int64
	Count      int64
	CategoryID *string
}

// SaleInfo — общая акция товара v1. Даты остаются строками.
type SaleInfo struct {
	BasePrice *Decimal `json:"common_base_price"`
	PriceUSD  *Decimal `json:"common_price_usd"`
	PriceEUR  *Decimal `json:"common_price_eur"`
	PriceRUR  *Decimal `json:"common_price_rur"`
	End       *string  `json:"sale_end"`
	Percent   *string  `json:"sale_percent"`
}

// CatalogProduct — строка списка товаров v1 и товаров продавца.
type CatalogProduct struct {
	ID            int64    `json:"id_goods"`
	Name          string   `json:"name_goods"`
	Info          string   `json:"info_goods"`
	Price         string   `json:"price"`
	Currency      string   `json:"currency"`
	AddInfo       *string  `json:"add_info"`
	SellCount     int64    `json:"cnt_sell"`
	ReturnCount   int64    `json:"cnt_return"`
	GoodResponses int64    `json:"cnt_goodresponses"`
	BadResponses  int64    `json:"cnt_badresponses"`
	PriceUSD      Decimal  `json:"price_usd"`
	PriceEUR      Decimal  `json:"price_eur"`
	PriceUAH      *Decimal `json:"price_uah"`
	PriceRUR      Decimal  `json:"price_rur"`
	InStock       int64    `json:"in_stock"`
	NumInStock    int64    `json:"num_in_stock"`
	Visible       int64    `json:"visible"`
	AgentFee      *string  `json:"commiss_agent"`
	HasDiscount   *bool    `json:"has_discount"`
	OptionCount   int64    `json:"num_options"`
	Sale          SaleInfo `json:"sale_info"`
}

// CatalogProductPage — страница списка товаров v1.
type CatalogProductPage struct {
	Page            int64
	Count           int64
	HasNextPage     bool
	HasPreviousPage bool
	TotalCount      int64
	TotalPages      int64
	Products        []CatalogProduct
}

// ListCatalogProductsParams — фильтр списка товаров. Page и Count обязательны.
type ListCatalogProductsParams struct {
	Lang  Lang
	Page  int64
	Count int64
	IDs   *string
}

// MoneyTrio — цены RUB, USD и EUR.
type MoneyTrio struct {
	RUB *Decimal `json:"RUB"`
	USD *Decimal `json:"USD"`
	EUR *Decimal `json:"EUR"`
}

// ProductPrices — начальная и обычная цена карточки товара.
type ProductPrices struct {
	Initial *MoneyTrio `json:"initial"`
	Default *MoneyTrio `json:"default"`
}

type PayLimit struct {
	Min *Decimal `json:"min"`
	Max *Decimal `json:"max"`
}

// PayCurrency — валюта способа оплаты.
type PayCurrency struct {
	Currency *string  `json:"currency"`
	Code     *string  `json:"code"`
	Price    *Decimal `json:"price"`
	Limit    PayLimit `json:"limit"`
}

// PayMethod — способ оплаты в карточке товара.
type PayMethod struct {
	Name       *string       `json:"name"`
	Currencies []PayCurrency `json:"currencies"`
}

// ProductUnit — продажа единицами.
type ProductUnit struct {
	Name     *string  `json:"unit_name"`
	Amount   *Decimal `json:"unit_amount"`
	AmountTo *string  `json:"unit_amount_desc"`
	Currency *string  `json:"unit_currency"`
	Count    *Decimal `json:"unit_cnt"`
	Min      *Decimal `json:"unit_cnt_min"`
	Max      *Decimal `json:"unit_cnt_max"`
	CountTo  *string  `json:"unit_cnt_desc"`
	Fixed    *bool    `json:"unit_fixed"`
	OnlyInt  *bool    `json:"unit_only_int"`
}

// ProductImage — превью товара.
type ProductImage struct {
	URL    *string `json:"url"`
	Width  *int64  `json:"width"`
	Height *int64  `json:"height"`
}

type ProductCrumb struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

// ProductOptionVariant — вариант опции в карточке v1. Суммы модификатора в снимке строки.
type ProductOptionVariant struct {
	Value         *int64  `json:"value"`
	Text          *string `json:"text"`
	Default       *int64  `json:"default"`
	Modify        *string `json:"modify"`
	ModifyType    *string `json:"modify_type"`
	ModifyValue   *string `json:"modify_value"`
	ModifyDefault *string `json:"modify_value_default"`
	InStock       *int64  `json:"num_in_stock"`
	Visible       *int64  `json:"visible"`
}

// ProductOption — опция карточки v1. Name в снимке integer.
type ProductOption struct {
	Name            int64                  `json:"name"`
	Label           string                 `json:"label"`
	Type            *string                `json:"type"`
	SeparateContent *int64                 `json:"separate_content"`
	Required        *int64                 `json:"required"`
	ModifierVisible *int64                 `json:"modifier_visible"`
	Hidden          *int64                 `json:"hidden"`
	Variants        []ProductOptionVariant `json:"variants"`
}

// ProductStats — счётчики карточки товара.
type ProductStats struct {
	Sales       int64 `json:"sales"`
	Refunds     int64 `json:"refunds"`
	GoodReviews int64 `json:"good_reviews"`
	BadReviews  int64 `json:"bad_reviews"`
}

// ProductSeller — продавец в карточке товара.
type ProductSeller struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

// ProductCard — карточка товара v1.
type ProductCard struct {
	ID               *int64            `json:"id"`
	PrevID           *int64            `json:"id_prev"`
	NextID           *int64            `json:"id_next"`
	Name             *string           `json:"name"`
	Price            *Decimal          `json:"price"`
	Currency         *string           `json:"currency"`
	URL              *string           `json:"url"`
	Info             *string           `json:"info"`
	AddInfo          *string           `json:"add_info"`
	ReleaseDate      *string           `json:"release_date"`
	AgencyFee        *string           `json:"agency_fee"`
	AgencySum        *string           `json:"agency_sum"`
	AgencyID         *int64            `json:"agency_id"`
	Collection       *string           `json:"collection"`
	Property         *int64            `json:"propertygood"`
	Available        *int64            `json:"is_available"`
	ShowRest         *int64            `json:"show_rest"`
	InStock          *int64            `json:"num_in_stock"`
	InLock           *int64            `json:"num_in_lock"`
	Prices           ProductPrices     `json:"prices"`
	PaymentMethods   []PayMethod       `json:"payment_methods"`
	Unit             *ProductUnit      `json:"prices_unit"`
	CodeVerification json.RawMessage   `json:"unique_code_verification"`
	Images           []ProductImage    `json:"preview_imgs"`
	Videos           []ProductImage    `json:"preview_videos"`
	Type             *string           `json:"type"`
	Text             *string           `json:"text"`
	File             *string           `json:"file"`
	CategoryID       *int64            `json:"category_id"`
	Breadcrumbs      []ProductCrumb    `json:"breadcrumbs"`
	Discounts        []json.RawMessage `json:"discounts"`
	Units            []json.RawMessage `json:"units"`
	Present          json.RawMessage   `json:"present"`
	GiftFee          *string           `json:"gift_commiss"`
	Options          []ProductOption   `json:"options"`
	OptionsCheck     *int64            `json:"options_check"`
	Statistics       ProductStats      `json:"statistics"`
	Seller           ProductSeller     `json:"seller"`
	Sale             SaleInfo          `json:"sale_info"`
}

// SellerGoodsQuery — тело POST seller-goods. Nil-поле в JSON не попадает.
type SellerGoodsQuery struct {
	SellerID   *int64         `json:"id_seller,omitempty"`
	Order      *GoodsOrder    `json:"order_col,omitempty"`
	Direction  *SortDirection `json:"order_dir,omitempty"`
	Rows       *int64         `json:"rows,omitempty"`
	Page       *int64         `json:"page,omitempty"`
	Currency   *GoodsCurrency `json:"currency,omitempty"`
	Lang       *Lang          `json:"lang,omitempty"`
	ShowHidden *int64         `json:"show_hidden,omitempty"`
	OwnerID    *int64         `json:"owner_id,omitempty"`
}

// SellerGoods — страница товаров продавца.
type SellerGoods struct {
	SellerID   int64
	SellerName string
	Count      int64
	Pages      int64
	Page       int64
	Order      string
	Direction  string
	Products   []CatalogProduct
}

// Balance — свободные, заблокированные и плюсовые суммы счёта.
type Balance struct {
	Locked *Decimal `json:"amount_t_lock"`
	Free   *Decimal `json:"amount_t_free"`
	Plus   *Decimal `json:"amount_t_plus"`
}

type LocalizedText struct {
	Locale string `json:"locale"`
	Value  string `json:"value"`
}

// ReceiptOperation — движение в чеке.
type ReceiptOperation struct {
	ID        int64   `json:"id"`
	Type      string  `json:"type"`
	DateTime  string  `json:"datetime"`
	Percent   int64   `json:"percent"`
	Price     Decimal `json:"price"`
	Currency  string  `json:"currency"`
	OnAccount Decimal `json:"on_account"`
}

// ReceiptProduct — товар в чеке.
type ReceiptProduct struct {
	ID      int64           `json:"id"`
	Name    []LocalizedText `json:"name"`
	Deleted bool            `json:"deleted"`
}

// Receipt — одна операция счёта.
type Receipt struct {
	AccountOperationID int64            `json:"account_operation_id"`
	Operation          ReceiptOperation `json:"operation"`
	OwnerID            int64            `json:"owner_id"`
	Product            ReceiptProduct   `json:"product"`
	CodeCheckAt        *string          `json:"code_check_datetime"`
	FreeAt             *string          `json:"date_free"`
	FreeDescription    *string          `json:"free_description"`
	Response           *string          `json:"response"`
}

// ReceiptPage — страница чеков.
type ReceiptPage struct {
	Page            int64
	Count           int64
	HasNextPage     bool
	HasPreviousPage bool
	TotalCount      int64
	TotalPages      int64
	Items           []Receipt
}

// ListReceiptsParams — фильтр чеков. Page и Count обязательны.
type ListReceiptsParams struct {
	Page       int64
	Count      int64
	Currency   *string
	Type       *ReceiptType
	CodeFilter *ReceiptCodeFilter
	AllowType  *string
	Start      *string
	Finish     *string
}

// VariantPriceChange — правка цены одного варианта. Nil-поле в JSON не попадает.
type VariantPriceChange struct {
	VariantID *int64           `json:"variant_id,omitempty"`
	Rate      *Decimal         `json:"rate,omitempty"`
	Type      *PriceChangeType `json:"type,omitempty"`
}

// ProductPriceChange — правка цены товара и его вариантов.
type ProductPriceChange struct {
	ProductID *int64               `json:"product_id,omitempty"`
	Price     *Decimal             `json:"price,omitempty"`
	Variants  []VariantPriceChange `json:"variants,omitempty"`
}

func (c *Client) ListCatalogCategories(ctx context.Context, params ListCatalogCategoriesParams) (CatalogCategoryPage, error) {
	lang, err := requiredLang("list catalog categories", params.Lang)
	if err != nil {
		return CatalogCategoryPage{}, err
	}
	if err := requirePage("list catalog categories", params.Page, params.Count); err != nil {
		return CatalogCategoryPage{}, err
	}
	q := url.Values{}
	q.Set("page", strconv.FormatInt(params.Page, 10))
	q.Set("count", strconv.FormatInt(params.Count, 10))
	setString(q, "category_id", params.CategoryID)
	var env struct {
		Category []CatalogCategory `json:"category"`
	}
	if err := c.do(ctx, call{
		method: http.MethodGet,
		path:   "/api_sellers/api/categories",
		query:  q,
		header: langHeader(lang),
		dest:   &env,
		auth:   authV1,
		retry:  true,
	}); err != nil {
		return CatalogCategoryPage{}, err
	}
	if env.Category == nil {
		env.Category = []CatalogCategory{}
	}
	for i := range env.Category {
		if env.Category[i].Children == nil {
			env.Category[i].Children = []int64{}
		}
	}
	return CatalogCategoryPage{Categories: env.Category}, nil
}

func (c *Client) ListCatalogProducts(ctx context.Context, params ListCatalogProductsParams) (CatalogProductPage, error) {
	lang, err := requiredLang("list catalog products", params.Lang)
	if err != nil {
		return CatalogProductPage{}, err
	}
	if err := requirePage("list catalog products", params.Page, params.Count); err != nil {
		return CatalogProductPage{}, err
	}
	q := url.Values{}
	q.Set("page", strconv.FormatInt(params.Page, 10))
	q.Set("count", strconv.FormatInt(params.Count, 10))
	setString(q, "ids", params.IDs)
	var env struct {
		Page            int64            `json:"page"`
		Count           int64            `json:"count"`
		HasNextPage     bool             `json:"has_next_page"`
		HasPreviousPage bool             `json:"has_previous_page"`
		TotalCount      int64            `json:"total_count"`
		TotalPages      int64            `json:"total_pages"`
		Rows            []CatalogProduct `json:"rows"`
	}
	if err := c.do(ctx, call{
		method: http.MethodGet,
		path:   "/api_sellers/api/products/list",
		query:  q,
		header: langHeader(lang),
		dest:   &env,
		auth:   authV1,
		retry:  true,
	}); err != nil {
		return CatalogProductPage{}, err
	}
	if env.Rows == nil {
		env.Rows = []CatalogProduct{}
	}
	return CatalogProductPage{
		Page:            env.Page,
		Count:           env.Count,
		HasNextPage:     env.HasNextPage,
		HasPreviousPage: env.HasPreviousPage,
		TotalCount:      env.TotalCount,
		TotalPages:      env.TotalPages,
		Products:        env.Rows,
	}, nil
}

func (c *Client) GetCatalogProduct(ctx context.Context, productID string) (ProductCard, error) {
	productID = strings.TrimSpace(productID)
	if productID == "" || productID == "." || productID == ".." {
		return ProductCard{}, fmt.Errorf("ggsel: get catalog product: id is required")
	}
	var env struct {
		Product ProductCard `json:"product"`
	}
	if err := c.do(ctx, call{
		method:    http.MethodGet,
		path:      "/api_sellers/api/products/" + url.PathEscape(productID) + "/data",
		errorPath: "/api_sellers/api/products/{product_id}/data",
		dest:      &env,
		auth:      authV1,
		retry:     true,
	}); err != nil {
		return ProductCard{}, err
	}
	card := env.Product
	if card.PaymentMethods == nil {
		card.PaymentMethods = []PayMethod{}
	}
	if card.Images == nil {
		card.Images = []ProductImage{}
	}
	if card.Breadcrumbs == nil {
		card.Breadcrumbs = []ProductCrumb{}
	}
	if card.Options == nil {
		card.Options = []ProductOption{}
	}
	for i := range card.Options {
		if card.Options[i].Variants == nil {
			card.Options[i].Variants = []ProductOptionVariant{}
		}
	}
	return card, nil
}

// ListSellerGoods возвращает товары продавца. Повтор POST не выполняется.
func (c *Client) ListSellerGoods(ctx context.Context, query SellerGoodsQuery) (SellerGoods, error) {
	if err := query.validate(); err != nil {
		return SellerGoods{}, err
	}
	var env struct {
		SellerID   int64            `json:"id_seller"`
		SellerName string           `json:"name_seller"`
		Count      int64            `json:"cnt_goods"`
		Pages      int64            `json:"pages"`
		Page       int64            `json:"page"`
		Order      string           `json:"order_col"`
		Direction  string           `json:"order_dir"`
		Rows       []CatalogProduct `json:"rows"`
	}
	if err := c.do(ctx, call{
		method: http.MethodPost,
		path:   "/api_sellers/api/seller-goods",
		body:   query,
		dest:   &env,
		auth:   authV1,
	}); err != nil {
		return SellerGoods{}, err
	}
	if env.Rows == nil {
		env.Rows = []CatalogProduct{}
	}
	return SellerGoods{
		SellerID:   env.SellerID,
		SellerName: env.SellerName,
		Count:      env.Count,
		Pages:      env.Pages,
		Page:       env.Page,
		Order:      env.Order,
		Direction:  env.Direction,
		Products:   env.Rows,
	}, nil
}

func (c *Client) GetBalance(ctx context.Context) (Balance, error) {
	var env struct {
		Content Balance `json:"content"`
	}
	if err := c.do(ctx, call{
		method: http.MethodGet,
		path:   "/api_sellers/api/sellers/account/balance/info",
		dest:   &env,
		auth:   authV1,
		retry:  true,
	}); err != nil {
		return Balance{}, err
	}
	return env.Content, nil
}

func (c *Client) ListReceipts(ctx context.Context, params ListReceiptsParams) (ReceiptPage, error) {
	if err := requirePage("list receipts", params.Page, params.Count); err != nil {
		return ReceiptPage{}, err
	}
	q, err := params.query()
	if err != nil {
		return ReceiptPage{}, err
	}
	var env struct {
		Content struct {
			Page            int64     `json:"page"`
			Count           int64     `json:"count"`
			HasNextPage     bool      `json:"has_next_page"`
			HasPreviousPage bool      `json:"has_previous_page"`
			TotalCount      int64     `json:"total_count"`
			TotalPages      int64     `json:"total_pages"`
			Items           []Receipt `json:"items"`
		} `json:"content"`
	}
	if err := c.do(ctx, call{
		method: http.MethodGet,
		path:   "/api_sellers/api/sellers/account/receipts",
		query:  q,
		dest:   &env,
		auth:   authV1,
		retry:  true,
	}); err != nil {
		return ReceiptPage{}, err
	}
	items := env.Content.Items
	if items == nil {
		items = []Receipt{}
	}
	for i := range items {
		if items[i].Product.Name == nil {
			items[i].Product.Name = []LocalizedText{}
		}
	}
	return ReceiptPage{
		Page:            env.Content.Page,
		Count:           env.Content.Count,
		HasNextPage:     env.Content.HasNextPage,
		HasPreviousPage: env.Content.HasPreviousPage,
		TotalCount:      env.Content.TotalCount,
		TotalPages:      env.Content.TotalPages,
		Items:           items,
	}, nil
}

// UpdateProductPrices ставит массовую правку цен. Тело — JSON-массив. Метод возвращает taskId и не ждёт задачу.
func (c *Client) UpdateProductPrices(ctx context.Context, changes []ProductPriceChange) (string, error) {
	if len(changes) == 0 {
		return "", fmt.Errorf("ggsel: update product prices: changes are required")
	}
	for _, change := range changes {
		if change.ProductID != nil && *change.ProductID <= 0 {
			return "", fmt.Errorf("ggsel: update product prices: id must be positive")
		}
		for _, variant := range change.Variants {
			if variant.VariantID != nil && *variant.VariantID <= 0 {
				return "", fmt.Errorf("ggsel: update product prices: id must be positive")
			}
			if variant.Type != nil && !known(*variant.Type, PricePercentPlus, PricePercentMinus, PriceMinus, PricePlus) {
				return "", fmt.Errorf("ggsel: update product prices: invalid type %q", *variant.Type)
			}
		}
	}
	var env struct {
		TaskID string `json:"taskId"`
	}
	if err := c.do(ctx, call{
		method: http.MethodPost,
		path:   "/api_sellers/api/product/edit/prices",
		body:   changes,
		dest:   &env,
		auth:   authV1,
	}); err != nil {
		return "", err
	}
	if strings.TrimSpace(env.TaskID) == "" {
		return "", fmt.Errorf("ggsel: update product prices: empty task id")
	}
	return env.TaskID, nil
}

func (q SellerGoodsQuery) validate() error {
	if q.Order != nil && !known(*q.Order, GoodsOrderSales) {
		return fmt.Errorf("ggsel: list seller goods: invalid order %q", *q.Order)
	}
	if q.Direction != nil && !known(*q.Direction, SortAsc, SortDesc) {
		return fmt.Errorf("ggsel: list seller goods: invalid direction %q", *q.Direction)
	}
	if q.Currency != nil && !known(*q.Currency, GoodsCurrencyUSD, GoodsCurrencyEUR, GoodsCurrencyRUB) {
		return fmt.Errorf("ggsel: list seller goods: invalid currency %q", *q.Currency)
	}
	if q.Lang != nil && !known(*q.Lang, LangRU, LangEN) {
		return fmt.Errorf("ggsel: list seller goods: invalid lang %q", *q.Lang)
	}
	if q.SellerID != nil && *q.SellerID <= 0 {
		return fmt.Errorf("ggsel: list seller goods: id must be positive")
	}
	if q.Page != nil && *q.Page <= 0 {
		return fmt.Errorf("ggsel: list seller goods: page must be positive")
	}
	if q.Rows != nil && *q.Rows <= 0 {
		return fmt.Errorf("ggsel: list seller goods: rows must be positive")
	}
	return nil
}

func (p ListReceiptsParams) query() (url.Values, error) {
	if p.Currency != nil && *p.Currency != "WMT" {
		return nil, fmt.Errorf("ggsel: list receipts: invalid currency %q", *p.Currency)
	}
	if p.Type != nil && !knownReceipt(*p.Type) {
		return nil, fmt.Errorf("ggsel: list receipts: invalid type %q", *p.Type)
	}
	if p.CodeFilter != nil && !known(*p.CodeFilter, ReceiptOnlyWaitingCode, ReceiptHideWaitingCode) {
		return nil, fmt.Errorf("ggsel: list receipts: invalid code filter %q", *p.CodeFilter)
	}
	q := url.Values{}
	q.Set("page", strconv.FormatInt(p.Page, 10))
	q.Set("count", strconv.FormatInt(p.Count, 10))
	setString(q, "currency", p.Currency)
	if p.Type != nil {
		q.Set("type", string(*p.Type))
	}
	if p.CodeFilter != nil {
		q.Set("codeFilter", string(*p.CodeFilter))
	}
	setString(q, "allowType", p.AllowType)
	setString(q, "start", p.Start)
	setString(q, "finish", p.Finish)
	return q, nil
}

func knownReceipt(value ReceiptType) bool {
	return known(value,
		ReceiptAgentAccruals, ReceiptProductSales, ReceiptAddFunds, ReceiptExchangeResponse,
		ReceiptExchangeRequest, ReceiptRefund, ReceiptAdvGoods, ReceiptExternalCommission,
		ReceiptDiskRent, ReceiptPartnerSpace, ReceiptGiftCertificates, ReceiptWalletTransfer,
	)
}

func requiredLang(op string, lang Lang) (Lang, error) {
	if lang == "" {
		return LangRU, nil
	}
	if !known(lang, LangRU, LangEN) {
		return "", fmt.Errorf("ggsel: %s: invalid lang %q", op, lang)
	}
	return lang, nil
}

func requirePage(op string, page, count int64) error {
	if page <= 0 || count <= 0 {
		return fmt.Errorf("ggsel: %s: page and count must be positive", op)
	}
	return nil
}

func langHeader(lang Lang) http.Header {
	h := make(http.Header)
	h.Set("lang", string(lang))
	return h
}
