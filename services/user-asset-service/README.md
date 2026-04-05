# user-asset-service

Сервис пользовательских данных GTrade.

## Что уже готово

- HTTP-сервер на Gin
- загрузка конфигурации из env
- подключение к PostgreSQL через repository layer
- shared middleware: `RequestID`, `RequestLogger`
- `GET /health`
- `POST /users`
- `GET /users/:id`
- `PUT /users/:id`
- `GET /watchlist`
- `POST /watchlist`
- `DELETE /watchlist/:id`
- `GET /recent`
- `GET /preferences`
- `PUT /preferences`
- PostgreSQL persistence для user profile, watchlist и preferences
- валидация `watchlist` через `catalog-service`
- enrichment watchlist item summary из `catalog-service`
- базовые unit tests сервисного слоя
- HTTP smoke tests
- HTTP integration tests с реальной PostgreSQL и fake catalog client

## Что хранит сервис

- профиль пользователя:
  - `display_name`
  - `avatar_url`
  - `bio`
- watchlist пользователя в облаке
- пользовательские preferences

## Важные особенности

- `watchlist.item_id` теперь строковый и совместим с `catalog-service`
- watchlist должен хранить локальные идентификаторы предметов из каталога, а не числовые external ids
- при добавлении в watchlist сервис проверяет, что item существует и активен в `catalog-service`
- ответы watchlist/user/recent обогащаются item summary из `catalog-service`, если каталог доступен
- `GET /users/:id` возвращает профиль и текущий watchlist одним ответом
- preferences пока минимальные: `currency`, `notifications_enabled`

## Следующий логичный шаг

- решить, нужны ли отдельные list entity кроме базового watchlist
- добавить auth-aware flow, чтобы не передавать `user_id` в query/body после подключения gateway/auth
- добавить OpenAPI-driven client contract в gateway после подключения auth

## Тесты

```bash
cd services/user-asset-service
GOCACHE=/tmp/gocache-user-asset go test ./...
```

Интеграционные тесты с реальной PostgreSQL:

```bash
cd services/user-asset-service
TEST_DATABASE_URL='postgres://gtrade:gtrade@localhost:5434/gtrade_user_asset?sslmode=disable' \
GOCACHE=/tmp/gocache-user-asset go test ./internal/http -run TestRouterIntegration -v
```

## Ключевые файлы

- `internal/service/user_asset.go`
- `internal/repository/user_asset_repository.go`
- `internal/handler/user_asset.go`
- `internal/http/router_smoke_test.go`
- `internal/http/router_integration_test.go`
- `internal/service/user_asset_test.go`
- `docs/openapi.yaml`
- `migrations/0001_init.sql`
- `migrations/0002_profile_fields_and_watchlist_refs.sql`
