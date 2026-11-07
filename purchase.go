package ggsel

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// InvoiceState — состояние счёта v1.
type InvoiceState int

// Состояния счёта: 1 создан, 2 отменён, 3 оплачен, 4 выполнен, 5 возвращён.
const (
	InvoiceStateCreated   InvoiceState = 1
	InvoiceStateCanceled  InvoiceState = 2
	InvoiceStatePaid      InvoiceState = 3
	InvoiceStateCompleted InvoiceState = 4
	InvoiceStateRefunded  InvoiceState = 5
)

// FeedbackType — тип отзыва покупателя.
type FeedbackType string

// Значения типа отзыва.
const (
	FeedbackPositive FeedbackType = "positive"
	FeedbackNegative FeedbackType = "negative"
)

// Order — содержание заказа из purchase/info.
type Order struct {
	ItemID          *int64         `json:"item_id"`
	ContentID       int64          `json:"content_id"`
	CartUID         *string        `json:"cart_uid"`
	Name            *string        `json:"name"`
	Amount          Decimal        `json:"amount"`
	CurrencyType    string         `json:"currency_type"`
	InvoiceState    InvoiceState   `json:"invoice_state"`
	PurchaseDate    string         `json:"purchase_date"`
	DatePay         *string        `json:"date_pay"`
	AgentID         *int64         `json:"agent_id"`
	AgentPercent    *Decimal       `json:"agent_percent"`
	AgentFee        Decimal        `json:"agent_fee"`
	QueryString     *string        `json:"query_string"`
	UnitGoods       *string        `json:"unit_goods"`
	CountGoods      *string        `json:"cnt_goods"`
	PromoCode       *string        `json:"promo_code"`
	BonusCode       *string        `json:"bonus_code"`
	Feedback        *OrderFeedback `json:"feedback"`
	UniqueCodeState OrderCodeState `json:"unique_code_state"`
	Options         []OrderOption  `json:"options"`
	Buyer           Buyer          `json:"buyer_info"`
	Owner           int64          `json:"owner"`
	DayLock         int64          `json:"day_lock"`
	LockState       string         `json:"lock_state"`
	Profit          Decimal        `json:"profit"`
	ExternalOrderID *string        `json:"external_order_id"`
}

// OrderFeedback — отзыв в карточке заказа.
type OrderFeedback struct {
	Deleted  bool         `json:"deleted"`
	Feedback string       `json:"feedback"`
	Type     FeedbackType `json:"feedback_type"`
	Comment  *string      `json:"comment"`
}

// OrderCodeState — состояние уникального кода в заказе.
type OrderCodeState struct {
	State         *int64  `json:"state"`
	DateCheck     *string `json:"date_check"`
	DateDelivery  *string `json:"date_delivery"`
	DateConfirmed *string `json:"date_confirmed"`
	DateRefuted   *string `json:"date_refuted"`
}

// OrderOption — опция товара в заказе.
type OrderOption struct {
	ID         int64   `json:"id"`
	Name       *string `json:"name"`
	UserData   string  `json:"user_data"`
	UserDataID *int64  `json:"user_data_id"`
}

// Buyer — покупатель заказа.
type Buyer struct {
	PaymentMethod     string  `json:"payment_method"`
	Account           string  `json:"account"`
	Email             *string `json:"email"`
	Phone             *string `json:"phone"`
	Skype             *string `json:"skype"`
	WhatsApp          *string `json:"whatsapp"`
	IPAddress         *string `json:"ip_address"`
	PaymentAggregator string  `json:"payment_aggregator"`
}

// UniqueCode — результат проверки уникального кода.
// AgentPercent здесь строка: в этой схеме тип другой, чем у заказа.
type UniqueCode struct {
	InvoiceID       int64              `json:"inv"`
	GoodsID         int64              `json:"id_goods"`
	Amount          Decimal            `json:"amount"`
	CurrencyType    string             `json:"type_curr"`
	AmountUSD       Decimal            `json:"amount_usd"`
	Profit          Decimal            `json:"profit"`
	DatePay         string             `json:"date_pay"`
	Email           *string            `json:"email"`
	Name            string             `json:"name_invoice"`
	Lang            *string            `json:"lang"`
	AgentID         *int64             `json:"agent_id"`
	AgentPercent    *string            `json:"agent_percent"`
	QueryString     *string            `json:"query_string"`
	UnitGoods       *string            `json:"unit_goods"`
	CountGoods      *string            `json:"cnt_goods"`
	PromoCode       *string            `json:"promo_code"`
	BonusCode       *string            `json:"bonus_code"`
	CartUID         *string            `json:"cart_uid"`
	UniqueCodeState UniqueCodeState    `json:"unique_code_state"`
	Options         []UniqueCodeOption `json:"options"`
}

// UniqueCodeState — состояние уникального кода при проверке кода.
type UniqueCodeState struct {
	State         int64   `json:"state"`
	DateCheck     *string `json:"date_check"`
	DateDelivery  *string `json:"date_delivery"`
	DateConfirmed *string `json:"date_confirmed"`
	DateRefuted   *string `json:"date_refuted"`
}

// UniqueCodeOption — опция товара в ответе уникального кода.
type UniqueCodeOption struct {
	ID        *int64  `json:"id"`
	Name      *string `json:"name"`
	Value     *string `json:"value"`
	VariantID *int64  `json:"variant_id"`
}

// GetOrder возвращает заказ по номеру счёта.
func (c *Client) GetOrder(ctx context.Context, invoiceID int64) (Order, error) {
	if invoiceID <= 0 {
		return Order{}, fmt.Errorf("ggsel: get order: id must be positive")
	}
	var env struct {
		Content Order `json:"content"`
	}
	if err := c.do(ctx, call{
		method: http.MethodGet,
		path:   "/api_sellers/api/purchase/info/" + strconv.FormatInt(invoiceID, 10),
		header: c.localeHeader(),
		dest:   &env,
		auth:   authV1,
		retry:  true,
	}); err != nil {
		return Order{}, err
	}
	return env.Content, nil
}

// CheckUniqueCode проверяет уникальный код заказа.
func (c *Client) CheckUniqueCode(ctx context.Context, code string) (UniqueCode, error) {
	if strings.TrimSpace(code) == "" {
		return UniqueCode{}, fmt.Errorf("ggsel: check unique code: code is required")
	}
	escaped := url.PathEscape(code)
	if escaped == "." || escaped == ".." {
		return UniqueCode{}, fmt.Errorf("ggsel: check unique code: invalid code")
	}
	var out UniqueCode
	if err := c.do(ctx, call{
		method:    http.MethodGet,
		path:      "/api_sellers/api/purchases/unique-code/" + escaped,
		errorPath: "/api_sellers/api/purchases/unique-code/{unique_code}",
		dest:      &out,
		auth:      authV1,
		retry:     true,
	}); err != nil {
		return UniqueCode{}, err
	}
	return out, nil
}
