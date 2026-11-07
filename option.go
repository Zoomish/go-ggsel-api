package ggsel

import (
	"net/http"
	"time"
)

const (
	DefaultBaseURL = "https://seller.ggsel.com"

	defaultTimeout    = 30 * time.Second
	defaultMaxRetries = 2
	defaultBackoffMin = 100 * time.Millisecond
	defaultBackoffMax = 2 * time.Second
	maxRetryAfter     = 30 * time.Second
	maxResponseBody   = 8 << 20
	maxErrorBody      = 4096
)

// Options задаёт клиент. Нулевые поля, кроме APIKey, безопасны.
// New не выполняет сетевых вызовов.
type Options struct {
	BaseURL    string
	HTTPClient *http.Client
	APIKey     string
	SellerID   int64
	Locale     string
	// MaxRetries — сколько повторных попыток получает безопасный GET после первой.
	// Ноль равен двум. Отрицательное значение выключает повторы.
	MaxRetries      int
	RetryBackoffMin time.Duration
	RetryBackoffMax time.Duration
	Clock           func() time.Time
}
