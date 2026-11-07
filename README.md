# go-ggsel-api

[![Go Reference](https://pkg.go.dev/badge/github.com/Zoomish/go-ggsel-api.svg)](https://pkg.go.dev/github.com/Zoomish/go-ggsel-api)
[![Go 1.26](https://img.shields.io/badge/Go-1.26-00ADD8?logo=go)](https://go.dev/dl/)
[![CI](https://github.com/Zoomish/go-ggsel-api/actions/workflows/ci.yml/badge.svg)](https://github.com/Zoomish/go-ggsel-api/actions/workflows/ci.yml)

Неофициальный клиент официального Seller API GGSel. Один модуль `github.com/Zoomish/go-ggsel-api`, пакет `ggsel`, оба контура, база `https://seller.ggsel.com`. Это не клиент компании GGSel и не автовыдача.

[English](README.en.md) · [документация пакета](https://pkg.go.dev/github.com/Zoomish/go-ggsel-api)

## Содержание

- [Установка](#установка)
- [Быстрый старт](#быстрый-старт)
- [Контур v1](#контур-v1)
- [Деньги и частичное обновление](#деньги-и-частичное-обновление)
- [Ошибки](#ошибки)
- [Повторы](#повторы)
- [Что входит в модуль](#что-входит-в-модуль)
- [Контракт](#контракт)
- [Безопасность](#безопасность)
- [Лицензия](#лицензия)

## Установка

```bash
go get github.com/Zoomish/go-ggsel-api
```

## Быстрый старт

`New` только собирает клиент и не открывает соединение. Дедлайн задаётся через `context`.

```go
client, err := ggsel.New(ggsel.Options{
    APIKey: os.Getenv("GGSEL_API_KEY"),
})
if err != nil {
    return err
}
page, err := client.ListOffers(ctx, ggsel.ListOffersParams{})
```

Ключ v2 уходит в заголовок `Authorization` как есть, без `Bearer`.

Без ключа пример не открывает сеть и завершается с кодом 0:

```bash
go run ./examples/list_offers
```

## Контур v1

Для заказа, уникального кода и чатов нужен `SellerID`. Token вызывающий не передаёт и не видит: клиент сам делает `apilogin` и подставляет query-параметр.

```go
client, err := ggsel.New(ggsel.Options{
    APIKey:   os.Getenv("GGSEL_API_KEY"),
    SellerID: sellerID,
})
order, err := client.GetOrder(ctx, invoiceID)
```

## Деньги и частичное обновление

Цены и суммы — `ggsel.Decimal` поверх `shopspring/decimal`, не `float64`. В JSON уходит число.

В `OfferWrite` nil-поле не сериализуется. `Ptr(false)`, ноль и пустая строка уходят.

```go
_, err = client.UpdateOffer(ctx, offerID, ggsel.OfferWrite{
    IsAutoselling: ggsel.Ptr(false),
})
```

## Ошибки

```go
var apiErr *ggsel.APIError
if errors.As(err, &apiErr) {
    // apiErr.StatusCode, apiErr.RetVal, apiErr.Errors
}
if ggsel.IsUnauthorized(err) {
    // HTTP 401
}
```

`Error()` не содержит ключ, sign и token.

## Повторы

Повторяются только безопасные GET. `401` и отказ `apilogin` не повторяются. POST и PATCH, включая создание оффера и отправку сообщения, тоже не повторяются.

## Что входит в модуль

Сейчас это офферы и категории v2, а в v1 — заказ, уникальный код и чаты. Полный список методов — на [pkg.go.dev](https://pkg.go.dev/github.com/Zoomish/go-ggsel-api).

В модуле нет пакетных activate/pause/delete, опций, товаров, остальных методов v1 и XML `shop_search.asp`.

## Контракт

Пути и поля взяты из публичной документации Seller API. Клиент на seller.ggsel.com за спецификацией не ходит. Оба снимка лежат в [testdata/seller-api-v1.openapi.json](testdata/seller-api-v1.openapi.json) и [testdata/seller-api-v2.openapi.json](testdata/seller-api-v2.openapi.json). Раз в месяц `go run ./cmd/docscheck` сравнивает их с сайтом.

## Безопасность

Не кладите API-ключ, sign и token в issue и pull request. Как сообщать об уязвимости: [SECURITY.md](SECURITY.md).

## Лицензия

[MIT](LICENSE).
