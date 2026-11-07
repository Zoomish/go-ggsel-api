# Участие

Нужны Go 1.26 и, если обновляете снимок API, Python 3. Парсер снимка использует только стандартную библиотеку.

Перед pull request:

```bash
gofmt -w .
make test
```

`make test` запускает `go test ./...`. Так же годится `go vet ./...`.

## Снимок API

[docs/src/extract.py](docs/src/extract.py) не правьте, чтобы «починить» расхождение с сайтом. Он снимает публичную документацию как есть.

Обновление контракта:

```bash
python docs/src/extract.py
```

Просмотрите diff `docs/seller-api-v1.openapi.json` и `docs/seller-api-v2.openapi.json`. Потом обновите типы и тесты клиента и допишите [CHANGELOG.md](CHANGELOG.md). Проверка без записи: `python docs/src/extract.py --check`.

## Чего не должно быть в PR

API-ключа продавца, token, sign и файла `.env`.

## Версии

Тег модуля — `v0.x.y`. Тег `v2` из-за Seller API v2 не ставьте: Go воспримет его как новую мажорную версию модуля и сломает путь импорта.

Ежемесячная сверка снимка с сайтом запускается только с ветки по умолчанию. GitHub отключает scheduled workflow, если в репозитории долго нет активности.
