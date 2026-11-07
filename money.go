package ggsel

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/shopspring/decimal"
)

// Decimal — точное десятичное число для полей schema type number.
// В JSON это число, не строка и не float64.
// null в значении — ошибка разбора. null в *Decimal оставляет указатель nil.
type Decimal struct {
	value decimal.Decimal
}

// ParseDecimal собирает Decimal из десятичной строки. Методы shopspring наружу не выходят.
func ParseDecimal(raw string) (Decimal, error) {
	inner, err := decimal.NewFromString(strings.TrimSpace(raw))
	if err != nil {
		return Decimal{}, fmt.Errorf("ggsel: parse decimal: %w", err)
	}
	return Decimal{value: inner}, nil
}

func (d Decimal) MarshalJSON() ([]byte, error) {
	return []byte(d.value.String()), nil
}

func (d *Decimal) UnmarshalJSON(raw []byte) error {
	if d == nil {
		return fmt.Errorf("unmarshal decimal: nil receiver")
	}
	if bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return fmt.Errorf("unmarshal decimal: null")
	}
	var inner decimal.Decimal
	if err := inner.UnmarshalJSON(raw); err != nil {
		return fmt.Errorf("unmarshal decimal: %w", err)
	}
	d.value = inner
	return nil
}

func (d Decimal) Equal(other Decimal) bool {
	return d.value.Equal(other.value)
}

func (d Decimal) String() string {
	return d.value.String()
}

func decimalOf(d decimal.Decimal) Decimal {
	return Decimal{value: d}
}
