package ggsel

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
)

// OfferOptionType — тип опции оффера v2.
type OfferOptionType string

const (
	OfferOptionText          OfferOptionType = "text"
	OfferOptionMultilineText OfferOptionType = "multiline_text"
	OfferOptionCheckBox      OfferOptionType = "check_box"
	OfferOptionRadioButton   OfferOptionType = "radio_button"
)

// OfferOptionStatus — статус опции. В фильтре видимости archived нет.
type OfferOptionStatus string

const (
	OfferOptionActive   OfferOptionStatus = "active"
	OfferOptionHidden   OfferOptionStatus = "hidden"
	OfferOptionArchived OfferOptionStatus = "archived"
)

// DiscountType — как вариант меняет цену.
type DiscountType string

const (
	DiscountFixed   DiscountType = "fixed"
	DiscountPercent DiscountType = "percent"
)

// ImpactType — направление изменения цены варианта.
type ImpactType string

const (
	ImpactIncrease ImpactType = "increase"
	ImpactDecrease ImpactType = "decrease"
)

// DeleteAll — строковый флаг архивации. В снимке это "true" или "false", не boolean.
type DeleteAll string

const (
	DeleteAllTrue  DeleteAll = "true"
	DeleteAllFalse DeleteAll = "false"
)

// OfferProductStatus — статус ключа на складе оффера.
type OfferProductStatus string

const (
	OfferProductInStock  OfferProductStatus = "in_stock"
	OfferProductSold     OfferProductStatus = "sold"
	OfferProductArchived OfferProductStatus = "archived"
)

// OfferVariant — вариант опции. Остатки заполнены в GetOfferOption.
type OfferVariant struct {
	ID           int64             `json:"id"`
	TitleRU      *string           `json:"title_ru"`
	TitleEN      *string           `json:"title_en"`
	Price        Decimal           `json:"price"`
	DiscountType DiscountType      `json:"discount_type"`
	ImpactType   ImpactType        `json:"impact_type"`
	IsDefault    bool              `json:"is_default"`
	Status       OfferOptionStatus `json:"status"`
	Position     int64             `json:"position"`
	InStock      *int64            `json:"in_stock_products_count"`
	Sold         *int64            `json:"sold_products_count"`
}

// OfferOption — опция оффера v2.
type OfferOption struct {
	ID                    int64             `json:"id"`
	Status                OfferOptionStatus `json:"status"`
	Type                  OfferOptionType   `json:"type"`
	TitleRU               *string           `json:"title_ru"`
	TitleEN               *string           `json:"title_en"`
	CommentRU             *string           `json:"comment_ru"`
	CommentEN             *string           `json:"comment_en"`
	IsRequired            bool              `json:"is_required"`
	IsPriceModifierHidden bool              `json:"is_price_modifier_hidden"`
	HasSplittedProducts   bool              `json:"has_splitted_products"`
	Position              int64             `json:"position"`
	Variants              []OfferVariant    `json:"variants"`
}

// OfferOptionPage — страница опций оффера.
type OfferOptionPage struct {
	Options    []OfferOption
	Pagination Pagination
}

// OfferOptionWrite — создание или обновление опции. Nil-поле в JSON не попадает.
type OfferOptionWrite struct {
	ID                    *int64             `json:"id,omitempty"`
	Type                  *OfferOptionType   `json:"type,omitempty"`
	Status                *OfferOptionStatus `json:"status,omitempty"`
	HasSplittedProducts   *bool              `json:"has_splitted_products,omitempty"`
	TitleRU               *string            `json:"title_ru,omitempty"`
	TitleEN               *string            `json:"title_en,omitempty"`
	CommentRU             *string            `json:"comment_ru,omitempty"`
	CommentEN             *string            `json:"comment_en,omitempty"`
	IsRequired            *bool              `json:"is_required,omitempty"`
	IsPriceModifierHidden *bool              `json:"is_price_modifier_hidden,omitempty"`
	Position              *int64             `json:"position,omitempty"`
}

// OptionVisibility — видимость одной опции. Статус только active или hidden.
type OptionVisibility struct {
	ID     int64             `json:"id"`
	Status OfferOptionStatus `json:"status"`
}

// OfferVariantWrite — создание или обновление варианта. Nil-поле в JSON не попадает.
type OfferVariantWrite struct {
	ID           *int64             `json:"id,omitempty"`
	TitleRU      *string            `json:"title_ru,omitempty"`
	TitleEN      *string            `json:"title_en,omitempty"`
	Price        *Decimal           `json:"price,omitempty"`
	DiscountType *DiscountType      `json:"discount_type,omitempty"`
	ImpactType   *ImpactType        `json:"impact_type,omitempty"`
	IsDefault    *bool              `json:"is_default,omitempty"`
	Status       *OfferOptionStatus `json:"status,omitempty"`
	Position     *int64             `json:"position,omitempty"`
}

// VariantVisibility — видимость одного варианта. Статус только active или hidden.
type VariantVisibility struct {
	ID     int64             `json:"id"`
	Status OfferOptionStatus `json:"status"`
}

// OptionArchive — архив опций. Пустой набор без delete_all в сеть не уходит.
type OptionArchive struct {
	OptionIDs []int64    `json:"option_ids,omitempty"`
	DeleteAll *DeleteAll `json:"delete_all,omitempty"`
}

// VariantArchive — архив вариантов. Ответ содержит job_id, клиент его не опрашивает.
type VariantArchive struct {
	VariantIDs []int64    `json:"option_variant_ids,omitempty"`
	DeleteAll  *DeleteAll `json:"delete_all,omitempty"`
}

// ProductArchive — архив ключей оффера или варианта.
type ProductArchive struct {
	ProductIDs []int64    `json:"product_ids,omitempty"`
	DeleteAll  *DeleteAll `json:"delete_all,omitempty"`
}

// OfferProduct — ключ на складе оффера или варианта.
type OfferProduct struct {
	ID        int64              `json:"id"`
	Value     string             `json:"value"`
	Status    OfferProductStatus `json:"status"`
	CreatedAt string             `json:"created_at"`
}

// OfferProductPage — страница ключей.
type OfferProductPage struct {
	Products   []OfferProduct
	Pagination Pagination
}

// OfferProductInput — значение ключа при создании.
type OfferProductInput struct {
	Value string `json:"value"`
}

// ListOfferProductsParams — фильтр склада. Нулевые указатели в запрос не попадают.
type ListOfferProductsParams struct {
	Status        *OfferProductStatus
	SortColumn    *string
	SortDirection *string
}

// ListOfferOptions возвращает опции оффера, видимые продавцу.
func (c *Client) ListOfferOptions(ctx context.Context, offerID int64) (OfferOptionPage, error) {
	if err := requireID("list offer options", offerID); err != nil {
		return OfferOptionPage{}, err
	}
	var env pageEnvelope
	if err := c.do(ctx, call{
		method: http.MethodGet,
		path:   offerOptionsPath(offerID),
		header: c.localeHeader(),
		dest:   &env,
		auth:   authV2,
		retry:  true,
	}); err != nil {
		return OfferOptionPage{}, err
	}
	items, err := decodeOptions(env.Data)
	if err != nil {
		return OfferOptionPage{}, fmt.Errorf("ggsel: list offer options: %w", err)
	}
	return OfferOptionPage{Options: items, Pagination: env.Pagination}, nil
}

// GetOfferOption возвращает одну опцию вместе с остатками вариантов.
func (c *Client) GetOfferOption(ctx context.Context, offerID, optionID int64) (OfferOption, error) {
	if err := requireID("get offer option", offerID); err != nil {
		return OfferOption{}, err
	}
	if err := requireID("get offer option", optionID); err != nil {
		return OfferOption{}, err
	}
	var env objectEnvelope
	if err := c.do(ctx, call{
		method: http.MethodGet,
		path:   offerOptionsPath(offerID) + "/" + strconv.FormatInt(optionID, 10),
		header: c.localeHeader(),
		dest:   &env,
		auth:   authV2,
		retry:  true,
	}); err != nil {
		return OfferOption{}, err
	}
	var item OfferOption
	if err := decodeData(env.Data, &item); err != nil {
		return OfferOption{}, fmt.Errorf("ggsel: get offer option: %w", err)
	}
	fillVariants(&item)
	return item, nil
}

// CreateOfferOptions создаёт пачку опций. Повтор запроса не выполняется.
func (c *Client) CreateOfferOptions(ctx context.Context, offerID int64, options []OfferOptionWrite) ([]OfferOption, error) {
	if err := requireID("create offer options", offerID); err != nil {
		return nil, err
	}
	if err := validateOptionWrites("create offer options", options); err != nil {
		return nil, err
	}
	var env objectEnvelope
	if err := c.do(ctx, call{
		method: http.MethodPost,
		path:   offerOptionsPath(offerID),
		header: c.localeHeader(),
		body: struct {
			Options []OfferOptionWrite `json:"options"`
		}{Options: options},
		dest: &env,
		auth: authV2,
	}); err != nil {
		return nil, err
	}
	items, err := decodeOptions(env.Data)
	if err != nil {
		return nil, fmt.Errorf("ggsel: create offer options: %w", err)
	}
	return items, nil
}

// ArchiveOfferOptions ставит опции в архив и возвращает job. Клиент job не опрашивает.
func (c *Client) ArchiveOfferOptions(ctx context.Context, offerID int64, body OptionArchive) (Job, error) {
	if err := requireID("archive offer options", offerID); err != nil {
		return Job{}, err
	}
	if err := validateArchive("archive offer options", body.OptionIDs, body.DeleteAll); err != nil {
		return Job{}, err
	}
	return c.postJob(ctx, http.MethodDelete, offerOptionsPath(offerID), body)
}

// UpdateOfferOptionVisibility меняет видимость пачки опций.
func (c *Client) UpdateOfferOptionVisibility(ctx context.Context, offerID int64, options []OptionVisibility) ([]OfferOption, error) {
	if err := requireID("update offer option visibility", offerID); err != nil {
		return nil, err
	}
	if err := validateOptionVisibility(options); err != nil {
		return nil, err
	}
	var env objectEnvelope
	if err := c.do(ctx, call{
		method: http.MethodPatch,
		path:   offerOptionsPath(offerID) + "/batch_visibility",
		header: c.localeHeader(),
		body: struct {
			Options []OptionVisibility `json:"options"`
		}{Options: options},
		dest: &env,
		auth: authV2,
	}); err != nil {
		return nil, err
	}
	items, err := decodeOptions(env.Data)
	if err != nil {
		return nil, fmt.Errorf("ggsel: update offer option visibility: %w", err)
	}
	return items, nil
}

// UpsertOfferVariants создаёт или обновляет варианты опции.
func (c *Client) UpsertOfferVariants(ctx context.Context, offerID, optionID int64, variants []OfferVariantWrite) ([]OfferVariant, error) {
	if err := requireID("upsert offer variants", offerID); err != nil {
		return nil, err
	}
	if err := requireID("upsert offer variants", optionID); err != nil {
		return nil, err
	}
	if err := validateVariantWrites(variants); err != nil {
		return nil, err
	}
	var env objectEnvelope
	if err := c.do(ctx, call{
		method: http.MethodPost,
		path:   offerVariantsPath(offerID, optionID),
		header: c.localeHeader(),
		body: struct {
			Variants []OfferVariantWrite `json:"variants"`
		}{Variants: variants},
		dest: &env,
		auth: authV2,
	}); err != nil {
		return nil, err
	}
	return decodeVariants("upsert offer variants", env.Data)
}

// ArchiveOfferVariants ставит варианты в архив. В снимке это асинхронно: метод возвращает job_id и не ждёт завершения.
func (c *Client) ArchiveOfferVariants(ctx context.Context, offerID, optionID int64, body VariantArchive) (Job, error) {
	if err := requireID("archive offer variants", offerID); err != nil {
		return Job{}, err
	}
	if err := requireID("archive offer variants", optionID); err != nil {
		return Job{}, err
	}
	if err := validateArchive("archive offer variants", body.VariantIDs, body.DeleteAll); err != nil {
		return Job{}, err
	}
	return c.postJob(ctx, http.MethodDelete, offerVariantsPath(offerID, optionID), body)
}

// UpdateOfferVariantVisibility меняет видимость пачки вариантов.
func (c *Client) UpdateOfferVariantVisibility(ctx context.Context, offerID, optionID int64, variants []VariantVisibility) ([]OfferVariant, error) {
	if err := requireID("update offer variant visibility", offerID); err != nil {
		return nil, err
	}
	if err := requireID("update offer variant visibility", optionID); err != nil {
		return nil, err
	}
	if err := validateVariantVisibility(variants); err != nil {
		return nil, err
	}
	var env objectEnvelope
	if err := c.do(ctx, call{
		method: http.MethodPatch,
		path:   offerVariantsPath(offerID, optionID) + "/batch_visibility",
		header: c.localeHeader(),
		body: struct {
			Variants []VariantVisibility `json:"variants"`
		}{Variants: variants},
		dest: &env,
		auth: authV2,
	}); err != nil {
		return nil, err
	}
	return decodeVariants("update offer variant visibility", env.Data)
}

// ListOfferProducts возвращает ключи оффера.
func (c *Client) ListOfferProducts(ctx context.Context, offerID int64, params ListOfferProductsParams) (OfferProductPage, error) {
	if err := requireID("list offer products", offerID); err != nil {
		return OfferProductPage{}, err
	}
	query, err := params.query()
	if err != nil {
		return OfferProductPage{}, err
	}
	return c.productPage(ctx, "list offer products", offerProductsPath(offerID), query)
}

// CreateOfferProducts создаёт ключи оффера. Успех — ответ 204 без тела.
func (c *Client) CreateOfferProducts(ctx context.Context, offerID int64, products []OfferProductInput) error {
	if err := requireID("create offer products", offerID); err != nil {
		return err
	}
	if err := validateProductInputs(products); err != nil {
		return err
	}
	return c.do(ctx, call{
		method: http.MethodPost,
		path:   offerProductsPath(offerID),
		header: c.localeHeader(),
		body: struct {
			Products []OfferProductInput `json:"products"`
		}{Products: products},
		auth: authV2,
	})
}

// ArchiveOfferProducts архивирует ключи оффера и возвращает job.
func (c *Client) ArchiveOfferProducts(ctx context.Context, offerID int64, body ProductArchive) (Job, error) {
	if err := requireID("archive offer products", offerID); err != nil {
		return Job{}, err
	}
	if err := validateArchive("archive offer products", body.ProductIDs, body.DeleteAll); err != nil {
		return Job{}, err
	}
	return c.postJob(ctx, http.MethodDelete, offerProductsPath(offerID), body)
}

// ListSplittedProducts возвращает ключи варианта.
func (c *Client) ListSplittedProducts(ctx context.Context, offerID, variantID int64, params ListOfferProductsParams) (OfferProductPage, error) {
	if err := requireID("list splitted products", offerID); err != nil {
		return OfferProductPage{}, err
	}
	if err := requireID("list splitted products", variantID); err != nil {
		return OfferProductPage{}, err
	}
	query, err := params.query()
	if err != nil {
		return OfferProductPage{}, err
	}
	return c.productPage(ctx, "list splitted products", splittedProductsPath(offerID, variantID), query)
}

// CreateSplittedProducts создаёт ключи варианта. Успех — ответ 204 без тела.
func (c *Client) CreateSplittedProducts(ctx context.Context, offerID, variantID int64, products []OfferProductInput) error {
	if err := requireID("create splitted products", offerID); err != nil {
		return err
	}
	if err := requireID("create splitted products", variantID); err != nil {
		return err
	}
	if err := validateProductInputs(products); err != nil {
		return err
	}
	return c.do(ctx, call{
		method: http.MethodPost,
		path:   splittedProductsPath(offerID, variantID),
		header: c.localeHeader(),
		body: struct {
			Products []OfferProductInput `json:"products"`
		}{Products: products},
		auth: authV2,
	})
}

// ArchiveSplittedProducts архивирует ключи варианта и возвращает job.
func (c *Client) ArchiveSplittedProducts(ctx context.Context, offerID, variantID int64, body ProductArchive) (Job, error) {
	if err := requireID("archive splitted products", offerID); err != nil {
		return Job{}, err
	}
	if err := requireID("archive splitted products", variantID); err != nil {
		return Job{}, err
	}
	if err := validateArchive("archive splitted products", body.ProductIDs, body.DeleteAll); err != nil {
		return Job{}, err
	}
	return c.postJob(ctx, http.MethodDelete, splittedProductsPath(offerID, variantID), body)
}

func (c *Client) productPage(ctx context.Context, op, path string, query url.Values) (OfferProductPage, error) {
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
		return OfferProductPage{}, err
	}
	var items []OfferProduct
	if err := decodeData(env.Data, &items); err != nil {
		return OfferProductPage{}, fmt.Errorf("ggsel: %s: %w", op, err)
	}
	if items == nil {
		items = []OfferProduct{}
	}
	return OfferProductPage{Products: items, Pagination: env.Pagination}, nil
}

func (c *Client) postJob(ctx context.Context, method, path string, body any) (Job, error) {
	var job Job
	if err := c.do(ctx, call{
		method: method,
		path:   path,
		header: c.localeHeader(),
		body:   body,
		dest:   &job,
		auth:   authV2,
	}); err != nil {
		return Job{}, err
	}
	return job, nil
}

func (p ListOfferProductsParams) query() (url.Values, error) {
	q := url.Values{}
	if p.Status != nil {
		if !known(*p.Status, OfferProductInStock, OfferProductSold) {
			return nil, fmt.Errorf("ggsel: list products: invalid status %q", *p.Status)
		}
		q.Set("status", string(*p.Status))
	}
	setString(q, "sort_column", p.SortColumn)
	setString(q, "sort_direction", p.SortDirection)
	return q, nil
}

func decodeOptions(raw json.RawMessage) ([]OfferOption, error) {
	var items []OfferOption
	if err := decodeData(raw, &items); err != nil {
		return nil, err
	}
	if items == nil {
		items = []OfferOption{}
	}
	for i := range items {
		fillVariants(&items[i])
	}
	return items, nil
}

func decodeVariants(op string, raw json.RawMessage) ([]OfferVariant, error) {
	var items []OfferVariant
	if err := decodeData(raw, &items); err != nil {
		return nil, fmt.Errorf("ggsel: %s: %w", op, err)
	}
	if items == nil {
		items = []OfferVariant{}
	}
	return items, nil
}

func fillVariants(item *OfferOption) {
	if item.Variants == nil {
		item.Variants = []OfferVariant{}
	}
}

func offerOptionsPath(offerID int64) string {
	return "/api_sellers/v2/offers/" + strconv.FormatInt(offerID, 10) + "/options"
}

func offerVariantsPath(offerID, optionID int64) string {
	return offerOptionsPath(offerID) + "/" + strconv.FormatInt(optionID, 10) + "/variants"
}

func offerProductsPath(offerID int64) string {
	return "/api_sellers/v2/offers/" + strconv.FormatInt(offerID, 10) + "/products"
}

func splittedProductsPath(offerID, variantID int64) string {
	return "/api_sellers/v2/offers/" + strconv.FormatInt(offerID, 10) + "/variants/" + strconv.FormatInt(variantID, 10) + "/splitted_products"
}

func requireID(op string, id int64) error {
	if id <= 0 {
		return fmt.Errorf("ggsel: %s: id must be positive", op)
	}
	return nil
}

func requireIDs(op string, ids []int64, limit int) error {
	if len(ids) == 0 {
		return fmt.Errorf("ggsel: %s: ids are required", op)
	}
	if limit > 0 && len(ids) > limit {
		return fmt.Errorf("ggsel: %s: at most %d ids", op, limit)
	}
	for _, id := range ids {
		if id <= 0 {
			return fmt.Errorf("ggsel: %s: id must be positive", op)
		}
	}
	return nil
}

func validateArchive(op string, ids []int64, all *DeleteAll) error {
	if all != nil && !known(*all, DeleteAllTrue, DeleteAllFalse) {
		return fmt.Errorf("ggsel: %s: invalid delete_all %q", op, *all)
	}
	if len(ids) == 0 && all == nil {
		return fmt.Errorf("ggsel: %s: ids or delete_all is required", op)
	}
	for _, id := range ids {
		if id <= 0 {
			return fmt.Errorf("ggsel: %s: id must be positive", op)
		}
	}
	return nil
}

func validateOptionWrites(op string, options []OfferOptionWrite) error {
	if len(options) == 0 {
		return fmt.Errorf("ggsel: %s: options are required", op)
	}
	for _, item := range options {
		if item.Type != nil && !known(*item.Type, OfferOptionText, OfferOptionMultilineText, OfferOptionCheckBox, OfferOptionRadioButton) {
			return fmt.Errorf("ggsel: %s: invalid type %q", op, *item.Type)
		}
		if item.Status != nil && !known(*item.Status, OfferOptionActive, OfferOptionHidden, OfferOptionArchived) {
			return fmt.Errorf("ggsel: %s: invalid status %q", op, *item.Status)
		}
		if item.ID != nil && *item.ID <= 0 {
			return fmt.Errorf("ggsel: %s: id must be positive", op)
		}
	}
	return nil
}

func validateVariantWrites(variants []OfferVariantWrite) error {
	if len(variants) == 0 {
		return fmt.Errorf("ggsel: upsert offer variants: variants are required")
	}
	for _, item := range variants {
		if item.DiscountType != nil && !known(*item.DiscountType, DiscountFixed, DiscountPercent) {
			return fmt.Errorf("ggsel: upsert offer variants: invalid discount_type %q", *item.DiscountType)
		}
		if item.ImpactType != nil && !known(*item.ImpactType, ImpactIncrease, ImpactDecrease) {
			return fmt.Errorf("ggsel: upsert offer variants: invalid impact_type %q", *item.ImpactType)
		}
		if item.Status != nil && !known(*item.Status, OfferOptionActive, OfferOptionHidden, OfferOptionArchived) {
			return fmt.Errorf("ggsel: upsert offer variants: invalid status %q", *item.Status)
		}
		if item.ID != nil && *item.ID <= 0 {
			return fmt.Errorf("ggsel: upsert offer variants: id must be positive")
		}
	}
	return nil
}

func validateOptionVisibility(options []OptionVisibility) error {
	if len(options) == 0 {
		return fmt.Errorf("ggsel: update offer option visibility: options are required")
	}
	for _, item := range options {
		if item.ID <= 0 {
			return fmt.Errorf("ggsel: update offer option visibility: id must be positive")
		}
		if !known(item.Status, OfferOptionActive, OfferOptionHidden) {
			return fmt.Errorf("ggsel: update offer option visibility: invalid status %q", item.Status)
		}
	}
	return nil
}

func validateVariantVisibility(variants []VariantVisibility) error {
	if len(variants) == 0 {
		return fmt.Errorf("ggsel: update offer variant visibility: variants are required")
	}
	for _, item := range variants {
		if item.ID <= 0 {
			return fmt.Errorf("ggsel: update offer variant visibility: id must be positive")
		}
		if !known(item.Status, OfferOptionActive, OfferOptionHidden) {
			return fmt.Errorf("ggsel: update offer variant visibility: invalid status %q", item.Status)
		}
	}
	return nil
}

func validateProductInputs(products []OfferProductInput) error {
	if len(products) == 0 {
		return fmt.Errorf("ggsel: create products: products are required")
	}
	for _, item := range products {
		if item.Value == "" {
			return fmt.Errorf("ggsel: create products: value is required")
		}
	}
	return nil
}

func known[T ~string](value T, allowed ...T) bool {
	for _, item := range allowed {
		if value == item {
			return true
		}
	}
	return false
}
