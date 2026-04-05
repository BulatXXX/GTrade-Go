# api-integration-service

Сервис интеграции GTrade с внешними API маркетов и каталогов игр.

## Что уже готово

- HTTP-сервер на Gin
- загрузка конфигурации из env
- shared middleware: `RequestID`, `RequestLogger`
- endpoint `GET /health`
- endpoint `GET /search`
- endpoint `GET /items/:id`
- endpoint `GET /items/:id/prices`
- endpoint `GET /items/:id/top-price`
- endpoint `POST /internal/sync/item`
- endpoint `POST /internal/sync/search`
- общий service layer с registry внешних provider'ов
- адаптер `warframe.market`
- адаптер `EVE ESI`
- адаптер `tarkov.dev`
- catalog client для `POST /items/upsert`
- smoke tests HTTP-слоя
- unit tests service layer
- provider-level tests для `warframe`, `eve`, `tarkov`

## Готовые endpoint'ы

- `GET /health`
- `GET /search?game=...&q=...&limit=...&offset=...&game_mode=...`
- `GET /items/:id?game=...&game_mode=...`
- `GET /items/:id/prices?game=...&game_mode=...`
- `GET /items/:id/top-price?game=...&game_mode=...`
- `POST /internal/sync/item`
- `POST /internal/sync/search`

## Текущий flow

- внешний клиент вызывает `api-integration-service` для runtime-доступа к внешним данным
- сервис выбирает provider по `game`
- provider ходит во внешний API и нормализует ответ в общий DTO
- для `tarkov` дополнительно учитывается `game_mode` (`regular` по умолчанию, либо `pve`)
- `GET /items/:id/prices` возвращает полный normalized pricing snapshot
- `GET /items/:id/top-price` возвращает только главное значение цены для совместимости и простых UI-сценариев
- `POST /internal/sync/item` забирает внешний item и пишет его в `catalog-service`
- `POST /internal/sync/search` берет страницу provider search results и пишет их в `catalog-service`

## Как распределена ответственность

- `catalog-service` остается локальным source of truth для item metadata и поиска по `eve`
- `api-integration-service` отвечает за runtime fetch, нормализацию внешних данных и sync в каталог
- для `warframe` и `tarkov` сервис умеет и искать, и получать карточки, и получать цены
- для `eve` сервис получает item card и цены, а поиск должен идти из локального каталога

## Поддержанные источники

### Warframe

- поиск через `GET /items`
- карточка через `GET /items/{slug}`
- цены через `GET /orders/item/{slug}/top`
- валюта: `PLAT`
- `market_kind`: `live_orders`

### EVE

- карточка через `GET /universe/types/{type_id}`
- цены через `GET /markets/prices`
- валюта: `ISK`
- `market_kind`: `reference_market`
- runtime search не используется, поиск должен идти через `catalog-service`

### Tarkov

- поиск через GraphQL `items(...)`
- карточка через GraphQL `item(id: ..., gameMode: ...)`
- цены через GraphQL `item(id: ..., gameMode: ...)`
- валюта: `RUB`
- `market_kind`: `aggregated_market`
- поддержаны `game_mode=regular|pve`

## Что нужно доделать

- вынести OpenAPI/Swagger UI в удобный локальный просмотр
- добавить более богатые analytics endpoint'ы при необходимости
- решить, нужен ли storage для historical pricing snapshots
- добавить internal auth для внутренних sync endpoint'ов
- решить, нужен ли отдельный scheduler/worker для регулярного catalog sync

## Ключевые файлы

- `internal/service/service.go` — общий orchestrator и provider registry
- `internal/service/marketplace/warframe.go` — адаптер `warframe.market`
- `internal/service/marketplace/eve.go` — адаптер `EVE ESI`
- `internal/service/marketplace/tarkov.go` — адаптер `tarkov.dev`
- `internal/client/catalog/client.go` — клиент `catalog-service`
- `internal/handler/integration.go` — HTTP handlers runtime fetch и internal sync endpoint'ов
- `internal/model/model.go` — unified DTO для item/pricing/top-price/sync
- `internal/http/router.go` — роутер и middleware
- `internal/http/router_smoke_test.go` — HTTP smoke tests
- `docs/openapi.yaml` — актуальный OpenAPI/Swagger контракт
- `docs/README.md` — локальный запуск, ручная проверка и тесты

## Тесты

- unit tests service layer лежат в `internal/service/service_test.go`
- provider tests лежат в:
  - `internal/service/marketplace/warframe_test.go`
  - `internal/service/marketplace/eve_test.go`
  - `internal/service/marketplace/tarkov_test.go`
- HTTP smoke tests лежат в `internal/http/router_smoke_test.go`

Проверка:

```bash
cd services/api-integration-service
GOCACHE=/tmp/gocache-api-integration go test ./...
```

## Полезные документы

- `docs/README.md`
- `docs/openapi.yaml`
- `../../docs/architecture.md`
- `../../docs/roadmap.md`
