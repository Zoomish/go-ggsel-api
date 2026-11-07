# Участие

Нужны Go 1.26. Локальный съёмщик документации в репозиторий не входит.

Перед pull request:

```bash
gofmt -w .
make test
```

`make test` запускает `go test ./...`. Так же годится `go vet ./...`.

## Снимок API

В git лежат оба контракта: [testdata/seller-api-v1.openapi.json](testdata/seller-api-v1.openapi.json) и [testdata/seller-api-v2.openapi.json](testdata/seller-api-v2.openapi.json). `go test` сверяет enum v2 со вторым файлом и проверяет, что в первом есть `apilogin`.

Раз в месяц workflow запускает `go run ./cmd/docscheck`. Команда читает живую документацию и сравнивает пути, схемы и авторизацию с этими двумя файлами. Дату в `info` она не учитывает. Python-папка `docs/` в репозиторий не входит.

Если сверка разошлась, локально `go run ./cmd/docscheck -write`, затем типы, тесты и [CHANGELOG.md](CHANGELOG.md). Короткий diff: `go run ./cmd/docscheck -diff`.

## Чего не должно быть в PR

API-ключа продавца, token, sign и файла `.env`.

## Версии

Тег модуля — `v0.x.y`. Тег `v2` из-за Seller API v2 не ставьте: Go воспримет его как новую мажорную версию модуля и сломает путь импорта.

## Первая публикация

Тег ставит человек. Перед пушем тега проверьте, что в [go.mod](go.mod) нет директивы `replace`, [go.sum](go.sum) закоммичен, а [LICENSE](LICENSE) лежит в корне: pkg.go.dev по нему показывает MIT.

Тег имеет вид `v0.1.0`, не `v2.0.0`. Seller API v2 — это контур внутри того же модуля. После `git push` тега откройте https://pkg.go.dev/github.com/Zoomish/go-ggsel-api, чтобы индекс забрал версию.

Ежемесячный workflow сверяет оба снимка с сайтом и отдельно гоняет `go test`. Он живёт только на ветке по умолчанию и засыпает, если в репозитории долго нет активности.
