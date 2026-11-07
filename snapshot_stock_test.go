package ggsel

import (
	"slices"
	"testing"
)

func TestSnapshotStockEnums(t *testing.T) {
	t.Parallel()
	v2 := openAPI(t, "testdata/seller-api-v2.openapi.json")
	v1 := openAPI(t, "testdata/seller-api-v1.openapi.json")
	same := func(name string, got, want []string) {
		t.Helper()
		slices.Sort(got)
		slices.Sort(want)
		if !slices.Equal(got, want) {
			t.Fatalf("%s enum = %v, constants = %v", name, got, want)
		}
	}

	optionBody := v2.object("paths").object("/api_sellers/v2/offers/{offer_id}/options").object("post").object("requestBody").object("content").object("application/json").object("schema").object("properties").object("options").object("items").object("properties")
	same("option type", enumAt(t, optionBody.object("type")), []string{
		string(OfferOptionText), string(OfferOptionMultilineText), string(OfferOptionCheckBox), string(OfferOptionRadioButton),
	})
	same("option status", enumAt(t, optionBody.object("status")), []string{
		string(OfferOptionActive), string(OfferOptionHidden), string(OfferOptionArchived),
	})

	archive := v2.object("paths").object("/api_sellers/v2/offers/{offer_id}/options").object("delete").object("requestBody").object("content").object("application/json").object("schema").object("properties")
	same("delete_all", enumAt(t, archive.object("delete_all")), []string{string(DeleteAllTrue), string(DeleteAllFalse)})

	variantBody := v2.object("paths").object("/api_sellers/v2/offers/{offer_id}/options/{option_id}/variants").object("post").object("requestBody").object("content").object("application/json").object("schema").object("properties").object("variants").object("items").object("properties")
	same("discount", enumAt(t, variantBody.object("discount_type")), []string{string(DiscountFixed), string(DiscountPercent)})
	same("impact", enumAt(t, variantBody.object("impact_type")), []string{string(ImpactIncrease), string(ImpactDecrease)})

	products := v2.object("paths").object("/api_sellers/v2/offers/{offer_id}/products").object("get")
	same("product filter", paramEnum(t, products, "status"), []string{string(OfferProductInStock), string(OfferProductSold)})

	job := v2.object("paths").object("/api_sellers/v2/async_job_results/{job_id}").object("get").object("responses").object("200").object("content").object("application/json").object("schema").object("properties").object("data").object("properties")
	same("async job", enumAt(t, job.object("status")), []string{string(AsyncJobPending), string(AsyncJobCompleted), string(AsyncJobFailed)})

	priceItem := v1.object("paths").object("/api_sellers/api/product/edit/prices").object("post").object("requestBody").object("content").object("application/json").object("schema").object("items").object("properties").object("variants").object("items").object("properties")
	same("price change", enumAt(t, priceItem.object("type")), []string{
		string(PricePercentPlus), string(PricePercentMinus), string(PriceMinus), string(PricePlus),
	})

	reviews := v1.object("paths").object("/api_sellers/api/reviews").object("get")
	same("review filter", paramEnum(t, reviews, "type"), []string{string(ReviewGood), string(ReviewBad), string(ReviewAll)})
	same("review locale", paramEnum(t, reviews, "locale"), []string{string(LangRU), string(LangEN)})

	receipts := v1.object("paths").object("/api_sellers/api/sellers/account/receipts").object("get")
	same("receipt currency", paramEnum(t, receipts, "currency"), []string{"WMT"})
	same("receipt code", paramEnum(t, receipts, "codeFilter"), []string{string(ReceiptOnlyWaitingCode), string(ReceiptHideWaitingCode)})
	same("receipt type", paramEnum(t, receipts, "type"), []string{
		string(ReceiptAgentAccruals), string(ReceiptProductSales), string(ReceiptAddFunds), string(ReceiptExchangeResponse),
		string(ReceiptExchangeRequest), string(ReceiptRefund), string(ReceiptAdvGoods), string(ReceiptExternalCommission),
		string(ReceiptDiskRent), string(ReceiptPartnerSpace), string(ReceiptGiftCertificates), string(ReceiptWalletTransfer),
	})

	optionsGET := v2.object("paths").object("/api_sellers/v2/offers/{offer_id}/options").object("get")
	if queries := queryParams(t, optionsGET); len(queries) != 0 {
		t.Fatalf("list offer options query = %v", queries)
	}
	productQuery := []string{"sort_column", "sort_direction", "status"}
	for _, path := range []string{
		"/api_sellers/v2/offers/{offer_id}/products",
		"/api_sellers/v2/offers/{offer_id}/variants/{variant_id}/splitted_products",
	} {
		got := queryParams(t, v2.object("paths").object(path).object("get"))
		slices.Sort(got)
		if !slices.Equal(got, productQuery) {
			t.Fatalf("%s query = %v", path, got)
		}
	}
	for _, path := range []string{
		"/api_sellers/v2/offers/batch_activate",
		"/api_sellers/v2/offers/batch_pause",
		"/api_sellers/v2/offers/batch_delete",
	} {
		raw := v2.object("paths").object(path).object("post").object("requestBody").object("content").object("application/json").object("schema").object("properties").object("offer_ids")["maxItems"]
		limit, ok := raw.(float64)
		if !ok || int(limit) != maxOfferBatch {
			t.Fatalf("%s maxItems = %v", path, raw)
		}
	}

	feedback := v1.object("paths").object("/api_sellers/api/purchase/info/{invoice_id}").object("get").object("responses").object("200").object("content").object("application/json").object("schema").object("properties").object("content").object("properties").object("feedback").object("properties")
	same("feedback type", enumAt(t, feedback.object("feedback_type")), []string{string(FeedbackPositive), string(FeedbackNegative)})
	goods := v1.object("paths").object("/api_sellers/api/seller-goods").object("post").object("requestBody").object("content").object("application/json").object("schema").object("properties")
	same("goods order", enumAt(t, goods.object("order_col")), []string{string(GoodsOrderSales)})
	same("goods direction", enumAt(t, goods.object("order_dir")), []string{string(SortAsc), string(SortDesc)})
	same("goods currency", enumAt(t, goods.object("currency")), []string{string(GoodsCurrencyUSD), string(GoodsCurrencyEUR), string(GoodsCurrencyRUB)})
	same("goods lang", enumAt(t, goods.object("lang")), []string{string(LangRU), string(LangEN)})

	task := v1.object("paths").object("/api_sellers/api/product/edit/UpdateProductsTaskStatus").object("get").object("responses").object("200").object("content").object("application/json").object("schema").object("properties").object("Status")
	got := enumInts(t, task["enum"])
	want := []int{int(ProductTaskStatus1), int(ProductTaskStatus2), int(ProductTaskStatus3)}
	slices.Sort(got)
	slices.Sort(want)
	if !slices.Equal(got, want) {
		t.Fatalf("task status enum = %v, constants = %v", got, want)
	}
}

func queryParams(t *testing.T, op openDoc) []string {
	t.Helper()
	params, _ := op["parameters"].([]any)
	var names []string
	for _, item := range params {
		param, _ := item.(map[string]any)
		if param["in"] != "query" {
			continue
		}
		name, _ := param["name"].(string)
		names = append(names, name)
	}
	return names
}

func enumInts(t *testing.T, raw any) []int {
	t.Helper()
	items, _ := raw.([]any)
	if len(items) == 0 {
		t.Fatal("empty enum")
	}
	out := make([]int, 0, len(items))
	for _, item := range items {
		number, ok := item.(float64)
		if !ok {
			t.Fatalf("enum value %v", item)
		}
		out = append(out, int(number))
	}
	return out
}
