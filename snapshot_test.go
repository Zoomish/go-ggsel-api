package ggsel

import (
	"encoding/json"
	"os"
	"slices"
	"testing"
)

func TestSnapshotEnumsMatchConstants(t *testing.T) {
	t.Parallel()
	doc := openAPI(t, "docs/seller-api-v2.openapi.json")
	offers := doc.object("paths").object("/api_sellers/v2/offers")
	list := offers.object("get")
	create := offers.object("post").object("requestBody").object("content").object("application/json").object("schema").object("properties")
	offerID := doc.object("paths").object("/api_sellers/v2/offers/{id}").object("get")
	statusEnum := enumAt(t, offerID.object("responses").object("200").object("content").object("application/json").object("schema").object("properties").object("data").object("properties").object("status"))

	assertEnum := func(name string, got, want []string) {
		t.Helper()
		slices.Sort(got)
		slices.Sort(want)
		if !slices.Equal(got, want) {
			t.Fatalf("%s enum = %v, constants = %v", name, got, want)
		}
	}

	assertEnum("offer status", statusEnum, []string{
		string(OfferStatusDraft), string(OfferStatusActive), string(OfferStatusPaused), string(OfferStatusArchived),
	})
	listStatus := paramEnum(t, list, "status")
	assertEnum("list status", listStatus, []string{
		string(OfferListStatusActive), string(OfferListStatusPaused), string(OfferListStatusDraft),
	})
	for _, status := range []OfferListStatus{OfferListStatusActive, OfferListStatusPaused, OfferListStatusDraft} {
		if !validOfferListStatus(status) {
			t.Fatalf("validator rejected %s", status)
		}
	}
	if validOfferListStatus(OfferListStatus(OfferStatusArchived)) {
		t.Fatal("archived passed the list filter")
	}

	assertEnum("sort", paramEnum(t, list, "sort"), []string{
		string(OfferSortPriceAsc), string(OfferSortPriceDesc),
		string(OfferSortSalesCountAsc), string(OfferSortSalesCountDesc),
		string(OfferSortProductsCountAsc), string(OfferSortProductsCountDesc),
	})
	for _, sort := range []OfferSort{OfferSortPriceAsc, OfferSortPriceDesc, OfferSortSalesCountAsc, OfferSortSalesCountDesc, OfferSortProductsCountAsc, OfferSortProductsCountDesc} {
		if !validOfferSort(sort) {
			t.Fatalf("validator rejected %s", sort)
		}
	}

	delivery := []string{string(DeliveryAuto), string(DeliveryManual)}
	assertEnum("delivery query", paramEnum(t, list, "delivery"), delivery)
	assertEnum("delivery body", enumAt(t, create.object("delivery")), delivery)
	for _, value := range []Delivery{DeliveryAuto, DeliveryManual} {
		if !validDelivery(value) {
			t.Fatalf("validator rejected %s", value)
		}
		if err := (OfferWrite{Delivery: &value}).validate("test"); err != nil {
			t.Fatalf("write validator rejected %s: %v", value, err)
		}
	}

	assertEnum("currency", enumAt(t, create.object("currency")), []string{string(CurrencyRUB)})
	currency := CurrencyRUB
	if err := (OfferWrite{Currency: &currency}).validate("test"); err != nil {
		t.Fatal(err)
	}

	notes := create.object("notification_settings").object("properties")
	assertEnum("notification type", enumAt(t, notes.object("type")), []string{string(NotificationEmail), string(NotificationURL)})
	assertEnum("notification method", enumAt(t, notes.object("http_method")), []string{string(NotificationMethodGET), string(NotificationMethodPOST)})
	for _, kind := range []NotificationType{NotificationEmail, NotificationURL} {
		if err := (OfferWrite{NotificationSettings: &NotificationSettingsWrite{Type: &kind}}).validate("test"); err != nil {
			t.Fatalf("type %s: %v", kind, err)
		}
	}
	for _, method := range []NotificationMethod{NotificationMethodGET, NotificationMethodPOST} {
		if err := (OfferWrite{NotificationSettings: &NotificationSettingsWrite{HTTPMethod: &method}}).validate("test"); err != nil {
			t.Fatalf("method %s: %v", method, err)
		}
	}
}

type openDoc map[string]any

func openAPI(t *testing.T, path string) openDoc {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	return doc
}

func (d openDoc) object(key string) openDoc {
	child, _ := d[key].(map[string]any)
	if child == nil {
		return openDoc{}
	}
	return child
}

func paramEnum(t *testing.T, op openDoc, name string) []string {
	t.Helper()
	params, _ := op["parameters"].([]any)
	for _, item := range params {
		param, _ := item.(map[string]any)
		if param["name"] != name {
			continue
		}
		schema, _ := param["schema"].(map[string]any)
		return enumStrings(t, schema["enum"])
	}
	t.Fatalf("parameter %s not found", name)
	return nil
}

func enumAt(t *testing.T, schema openDoc) []string {
	t.Helper()
	return enumStrings(t, schema["enum"])
}

func enumStrings(t *testing.T, raw any) []string {
	t.Helper()
	items, _ := raw.([]any)
	if len(items) == 0 {
		t.Fatal("empty enum")
	}
	out := make([]string, 0, len(items))
	for _, item := range items {
		text, ok := item.(string)
		if !ok {
			t.Fatalf("enum value %v", item)
		}
		out = append(out, text)
	}
	return out
}
