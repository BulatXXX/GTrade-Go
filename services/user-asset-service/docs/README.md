# user-asset-service docs

Документация по локальному запуску и ручной проверке `user-asset-service`.

## Назначение

`user-asset-service` хранит пользовательское состояние в облаке:

- профиль пользователя
- watchlist предметов
- пользовательские preferences

Watchlist связан с `catalog-service`:

- при добавлении item валидируется через каталог
- в ответах watchlist и user item обогащается summary из каталога

## Локальный запуск

```bash
cp deploy/.env.example deploy/.env
make up
```

Базовый адрес:

```text
http://localhost:8082
```

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

## Ручная проверка

### Create user

```bash
curl -X POST http://localhost:8082/users \
  -H 'Content-Type: application/json' \
  -d '{"user_id":1,"display_name":"Alice","avatar_url":"https://cdn.example.com/a.png","bio":"Trader"}'
```

### Get user

```bash
curl -sS http://localhost:8082/users/1
```

### Update user

```bash
curl -X PUT http://localhost:8082/users/1 \
  -H 'Content-Type: application/json' \
  -d '{"display_name":"Alice Updated","avatar_url":"https://cdn.example.com/a2.png","bio":"Collector"}'
```

### Add watchlist item

```bash
curl -X POST http://localhost:8082/watchlist \
  -H 'Content-Type: application/json' \
  -d '{"user_id":1,"item_id":"item-uuid-1"}'
```

Если item отсутствует в `catalog-service`, сервис вернет ошибку.

### Get watchlist

```bash
curl -sS 'http://localhost:8082/watchlist?user_id=1'
```

В ответе каждый элемент watchlist теперь может содержать вложенный объект `item` с базовыми данными из каталога.

### Delete watchlist item

```bash
curl -X DELETE 'http://localhost:8082/watchlist/1?user_id=1'
```

### Get preferences

```bash
curl -sS 'http://localhost:8082/preferences?user_id=1'
```

### Update preferences

```bash
curl -X PUT http://localhost:8082/preferences \
  -H 'Content-Type: application/json' \
  -d '{"user_id":1,"currency":"usd","notifications_enabled":true}'
```

## Swagger / OpenAPI

OpenAPI-схема сервиса лежит в:

- `services/user-asset-service/docs/openapi.yaml`
