package ggsel

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
)

// Ptr возвращает указатель на v. Нужен, чтобы отправить false, 0 или пустую строку.
func Ptr[T any](v T) *T {
	return &v
}

// OfferWrite — тело создания и частичного обновления оффера.
// Nil-поле в JSON не попадает.
type OfferWrite struct {
	TitleRU              *string                    `json:"title_ru,omitempty"`
	TitleEN              *string                    `json:"title_en,omitempty"`
	DescriptionRU        *string                    `json:"description_ru,omitempty"`
	DescriptionEN        *string                    `json:"description_en,omitempty"`
	InstructionsRU       *string                    `json:"instructions_ru,omitempty"`
	InstructionsEN       *string                    `json:"instructions_en,omitempty"`
	CoverImageRU         *string                    `json:"cover_image_ru,omitempty"`
	CoverImageEN         *string                    `json:"cover_image_en,omitempty"`
	Price                *Decimal                   `json:"price,omitempty"`
	Currency             *Currency                  `json:"currency,omitempty"`
	IsAutoselling        *bool                      `json:"is_autoselling,omitempty"`
	CategoryID           *int64                     `json:"category_id,omitempty"`
	MinQuantity          *int64                     `json:"min_quantity,omitempty"`
	MaxQuantity          *int64                     `json:"max_quantity,omitempty"`
	Quantity             *int64                     `json:"quantity,omitempty"`
	IsUnlimitedQuantity  *bool                      `json:"is_unlimited_quantity,omitempty"`
	PostPaymentURL       *string                    `json:"post_payment_url,omitempty"`
	Delivery             *Delivery                  `json:"delivery,omitempty"`
	PrePaymentSettings   *PrePaymentSettingsWrite   `json:"pre_payment_settings,omitempty"`
	NotificationSettings *NotificationSettingsWrite `json:"notification_settings,omitempty"`
}

// PrePaymentSettingsWrite — частичные настройки проверки до оплаты.
type PrePaymentSettingsWrite struct {
	IsEnabled    *bool   `json:"is_enabled,omitempty"`
	URL          *string `json:"url,omitempty"`
	AllowPayment *bool   `json:"allow_payment,omitempty"`
}

// NotificationSettingsWrite — частичные настройки уведомления о заказе.
type NotificationSettingsWrite struct {
	Type       *NotificationType   `json:"type,omitempty"`
	URL        *string             `json:"url,omitempty"`
	Email      *string             `json:"email,omitempty"`
	HTTPMethod *NotificationMethod `json:"http_method,omitempty"`
	IsDisabled *bool               `json:"is_disabled,omitempty"`
	IsDefault  *bool               `json:"is_default,omitempty"`
}

// CreateOffer создаёт оффер. Повтор запроса не выполняется.
func (c *Client) CreateOffer(ctx context.Context, body OfferWrite) (Offer, error) {
	return c.writeOffer(ctx, http.MethodPost, "/api_sellers/v2/offers", "create offer", body)
}

// UpdateOffer частично обновляет оффер. Повтор запроса не выполняется.
func (c *Client) UpdateOffer(ctx context.Context, id int64, body OfferWrite) (Offer, error) {
	if id <= 0 {
		return Offer{}, fmt.Errorf("ggsel: update offer: id must be positive")
	}
	return c.writeOffer(ctx, http.MethodPatch, "/api_sellers/v2/offers/"+strconv.FormatInt(id, 10), "update offer", body)
}

func (c *Client) writeOffer(ctx context.Context, method, path, op string, body OfferWrite) (Offer, error) {
	if err := body.validate(op); err != nil {
		return Offer{}, err
	}
	var env objectEnvelope
	if err := c.do(ctx, call{
		method: method,
		path:   path,
		header: c.localeHeader(),
		body:   body,
		dest:   &env,
		auth:   authV2,
		retry:  false,
	}); err != nil {
		return Offer{}, err
	}
	var offer Offer
	if err := decodeData(env.Data, &offer); err != nil {
		return Offer{}, fmt.Errorf("ggsel: %s: %w", op, err)
	}
	return offer, nil
}

func (w OfferWrite) validate(op string) error {
	if w.Currency != nil && *w.Currency != CurrencyRUB {
		return fmt.Errorf("ggsel: %s: invalid currency %q", op, *w.Currency)
	}
	if w.Delivery != nil && !validDelivery(*w.Delivery) {
		return fmt.Errorf("ggsel: %s: invalid delivery %q", op, *w.Delivery)
	}
	if w.NotificationSettings != nil {
		if err := w.NotificationSettings.validate(op); err != nil {
			return err
		}
	}
	return nil
}

func (n NotificationSettingsWrite) validate(op string) error {
	if n.Type != nil {
		switch *n.Type {
		case NotificationEmail, NotificationURL:
		default:
			return fmt.Errorf("ggsel: %s: invalid notification type %q", op, *n.Type)
		}
	}
	if n.HTTPMethod != nil {
		switch *n.HTTPMethod {
		case NotificationMethodGET, NotificationMethodPOST:
		default:
			return fmt.Errorf("ggsel: %s: invalid notification method %q", op, *n.HTTPMethod)
		}
	}
	return nil
}
