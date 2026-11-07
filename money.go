package ggsel

import (
	"bytes"
	"fmt"

	"github.com/shopspring/decimal"
)

// Decimal — точное десятичное число для полей schema type number.
// В JSON это число, не строка и не float64.
// Арифметика доступна через вложенный decimal.Decimal.
type Decimal struct {
	decimal.Decimal
}

func (d Decimal) MarshalJSON() ([]byte, error) {
	return []byte(d.String()), nil
}

func (d *Decimal) UnmarshalJSON(raw []byte) error {
	if d == nil {
		return fmt.Errorf("ggsel: unmarshal decimal: nil receiver")
	}
	if bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		*d = Decimal{}
		return nil
	}
	var inner decimal.Decimal
	if err := inner.UnmarshalJSON(raw); err != nil {
		return fmt.Errorf("ggsel: unmarshal decimal: %w", err)
	}
	d.Decimal = inner
	return nil
}

func (d Decimal) Equal(other Decimal) bool {
	return d.Decimal.Equal(other.Decimal)
}

func (d Decimal) String() string {
	return d.Decimal.String()
}
