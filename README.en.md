# go-ggsel-api

[![Go Reference](https://pkg.go.dev/badge/github.com/Zoomish/go-ggsel-api.svg)](https://pkg.go.dev/github.com/Zoomish/go-ggsel-api)
[![Go 1.26](https://img.shields.io/badge/Go-1.26-00ADD8?logo=go)](https://go.dev/dl/)
[![CI](https://github.com/Zoomish/go-ggsel-api/actions/workflows/ci.yml/badge.svg)](https://github.com/Zoomish/go-ggsel-api/actions/workflows/ci.yml)

Unofficial client for the official GGSel Seller API. One module, `github.com/Zoomish/go-ggsel-api`, package `ggsel`, both contours, base `https://seller.ggsel.com`. This is not a GGSel product and it does not deliver goods.

[Русский](README.md) · [package docs](https://pkg.go.dev/github.com/Zoomish/go-ggsel-api)

## Install

```bash
go get github.com/Zoomish/go-ggsel-api
```

## v2

`New` does not dial. The v2 key is sent in `Authorization` with no `Bearer` prefix.

```go
client, err := ggsel.New(ggsel.Options{
    APIKey: os.Getenv("GGSEL_API_KEY"),
})
page, err := client.ListOffers(ctx, ggsel.ListOffersParams{})
```

Without a key, `go run ./examples/list_offers` prints a notice and exits 0.

## v1

Orders, unique codes, and chats need `SellerID`. The caller never passes or reads the token. The client signs in and caches it until `valid_thru`.

```go
client, err := ggsel.New(ggsel.Options{
    APIKey:   os.Getenv("GGSEL_API_KEY"),
    SellerID: sellerID,
})
order, err := client.GetOrder(ctx, invoiceID)
```

## Money and partial updates

Prices and amounts are `ggsel.Decimal`, not `float64`. A nil field on `OfferWrite` is omitted. `Ptr(false)`, zero, and an empty string are sent.

## Errors and retries

Use `errors.As` with `*ggsel.APIError`. `IsUnauthorized` reports HTTP 401. `Error()` does not contain the API key, sign, or token.

Only safe GET calls are retried. A 401, a failed `apilogin`, and any POST or PATCH are not.

## Scope

The module currently covers v2 offers and categories, plus v1 orders, unique codes, and chats. Batch activate/pause/delete, options, products, the rest of v1, and the XML `shop_search.asp` method are not included. Method list: [pkg.go.dev](https://pkg.go.dev/github.com/Zoomish/go-ggsel-api).

Paths and fields follow the public Seller API documentation. The module does not fetch the spec. Both snapshots are [testdata/seller-api-v1.openapi.json](testdata/seller-api-v1.openapi.json) and [testdata/seller-api-v2.openapi.json](testdata/seller-api-v2.openapi.json). `go run ./cmd/docscheck` compares them with the live docs once a month. Report vulnerabilities as in [SECURITY.md](SECURITY.md).

## License

[MIT](LICENSE).
