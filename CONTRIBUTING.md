# Участие

Нужны Go 1.26. Локальный съёмщик документации в репозиторий не входит.

Перед pull request:

```bash
gofmt -w .
make test
```

`make test` запускает `go test ./...`. Так же годится `go vet ./...`.

## Снимок API

В git лежат оба контракта: [testdata/seller-api-v1.openapi.json](testdata/seller-api-v1.openapi.json) и [testdata/seller-api-v2.openapi.json](testdata/seller-api-v2.openapi.json). `go test` сверяет enum обоих снимков с константами, `maxItems` пачки офферов, `feedback_type`, enum `seller-goods`, query списков ключей и пустой query у списка опций. Наличие `apilogin` тоже проверяется. Имена всех полей снимок с типами не сверяет.

Раз в месяц workflow запускает `go run ./cmd/docscheck`. Команда читает живую документацию и сравнивает пути, схемы и авторизацию с этими двумя файлами. Дату в `info` она не учитывает. Python-папка `docs/` в репозиторий не входит.

Если сверка разошлась, локально `go run ./cmd/docscheck -write`, затем типы, тесты и [CHANGELOG.md](CHANGELOG.md). Короткий diff: `go run ./cmd/docscheck -diff`.

## Чего не должно быть в PR

API-ключа продавца, token, sign и файла `.env`.

## Версии

Тег модуля — `v0.x.y` или `v1.x.y`. Тег `v2` и выше не ставьте: Go воспримет его как новую мажорную версию модуля и сломает путь импорта. Seller API v2 — это контур внутри того же модуля.

Релиз дописывает в [CHANGELOG.md](CHANGELOG.md) строку версии и дату. Текст раздела пишется вручную.
