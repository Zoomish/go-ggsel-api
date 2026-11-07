package ggsel

import (
	"encoding/json"
	"testing"

	"github.com/shopspring/decimal"
)

func TestDecimalJSON(t *testing.T) {
	t.Parallel()
	price, err := decimal.NewFromString("19.99")
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(Decimal{Decimal: price})
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != "19.99" {
		t.Fatalf("json = %s", raw)
	}
	var back Decimal
	if err := json.Unmarshal(raw, &back); err != nil {
		t.Fatal(err)
	}
	if !back.Equal(Decimal{Decimal: price}) {
		t.Fatalf("back = %s", back)
	}

	var withExtra struct {
		Price Decimal `json:"price"`
	}
	if err := json.Unmarshal([]byte(`{"price":19.99,"unknown":{"n":1}}`), &withExtra); err != nil {
		t.Fatal(err)
	}
	if !withExtra.Price.Equal(Decimal{Decimal: price}) {
		t.Fatalf("price = %s", withExtra.Price)
	}
}

func TestDecimalPointerOmitempty(t *testing.T) {
	t.Parallel()
	type row struct {
		Price *Decimal `json:"price,omitempty"`
		On    *bool    `json:"on,omitempty"`
	}
	raw, err := json.Marshal(row{})
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != "{}" {
		t.Fatalf("empty = %s", raw)
	}
	zero := Decimal{Decimal: decimal.NewFromInt(0)}
	off := false
	raw, err = json.Marshal(row{Price: &zero, On: &off})
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != `{"price":0,"on":false}` {
		t.Fatalf("zero = %s", raw)
	}
}
