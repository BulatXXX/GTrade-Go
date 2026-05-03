# catalog-service

Канонический локальный каталог предметов GTrade.

## Что уже готово

- `GET /health`
- `GET /items`
- `GET /items/search`
- `GET /items/:id`
- `GET /items/:id/prices/history`
- `POST /items`
- `POST /items/upsert`
- `PUT /items/:id`
- `DELETE /items/:id`
- PostgreSQL persistence для `items` и `item_translations`
- PostgreSQL persistence для daily `prices` history
- локальный поиск по `name` и `translations.name`
- локализации через `item_translations`
- ingestion flow для `catalog-importer`
- integration tests repository layer
- service tests

## Роль сервиса

- локальный source of truth для item metadata
- локальный поиск предметов
- хранение переводов и локализованных полей
- точка записи для importer и sync flow
- владелец historical pricing snapshots

## Важные особенности

- `POST /items/upsert` используется для идемпотентного наполнения и обновления каталога
- уникальность предмета задается через `game + source + external_id`
- `DELETE /items/:id` по умолчанию работает как soft delete через `is_active=false`
- sync через `api-integration-service` обновляет базовую metadata и не должен сносить существующие переводы, если новые `translations` не переданы
- history цен обновляется фоновым collector'ом внутри сервиса через `api-integration-service`

## Следующий логичный шаг

- hardening backup flow перед full sync
- расширение scheduler/collector, если daily sync станет тяжелым и его придется выносить в отдельный worker
- дополнительные handler/router HTTP tests, если понадобится расширить покрытие

## Тесты

```bash
cd services/catalog-service
GOCACHE=/tmp/gocache-catalog go test ./...
```

Интеграционные тесты repository layer с PostgreSQL:

```bash
cd services/catalog-service
TEST_DATABASE_URL='postgres://gtrade:gtrade@localhost:5436/gtrade_catalog?sslmode=disable' \
GOCACHE=/tmp/gocache-catalog go test ./internal/repository -v
```

## Ключевые файлы

- `internal/service/service.go`
- `internal/repository/catalog_repository.go`
- `internal/handler/items.go`
- `internal/repository/catalog_repository_integration_test.go`
- `internal/service/service_test.go`
- `docs/openapi.yaml`
- `docs/README.md`
