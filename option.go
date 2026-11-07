package ggsel

import (
	"net/http"
	"time"
)

const (
	// DefaultBaseURL — база Seller API, если Options.BaseURL пуст.
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
	// BaseURL — база API. Пустое значение равно DefaultBaseURL.
	BaseURL string
	// HTTPClient — свой HTTP-клиент. Nil заменяется клиентом с таймаутом 30 секунд.
	// Переданный клиент не изменяется.
	HTTPClient *http.Client
	// APIKey — ключ продавца. Пустое значение делает New ошибкой.
	APIKey string
	// SellerID — идентификатор продавца для контура v1. Ноль допустим, пока v1 не вызывается.
	SellerID int64
	// Locale — заголовок locale. Пустое значение равно ru.
	Locale string
	// MaxRetries — сколько повторных попыток получает безопасный GET после первой.
	// Ноль равен двум. Отрицательное значение выключает повторы.
	MaxRetries int
	// RetryBackoffMin — нижняя граница паузы между повторами. Ноль равно 100 миллисекундам.
	RetryBackoffMin time.Duration
	// RetryBackoffMax — верхняя граница паузы между повторами. Ноль равно 2 секундам.
	RetryBackoffMax time.Duration
	// Clock — часы для timestamp и срока token. Nil вызывает time.Now.
	Clock func() time.Time
}
