# api-gateway

Внешний HTTP-фасад GTrade поверх доменных сервисов.

## Что уже готово

- `GET /health`
- проксирование `auth-service` через `/api/auth/*`
- проксирование `user-asset-service` через `/api/users/*`
- проксирование `catalog-service` через `/api/items/*`
- проксирование `api-integration-service` через `/api/market/*`
- проксирование `notification-service` через `/api/notifications/*`
- JWT-защита для `/api/users/*` и `/api/notifications/*`
- request id и request logging middleware
- smoke tests на routing, JWT и upstream proxy flow

## Публичный контракт

- `/api/auth/*` -> `auth-service`
- `/api/users/*` -> `user-asset-service`
- `/api/items/*` -> `catalog-service`
- `/api/market/*` -> `api-integration-service`
- `/api/notifications/*` -> `notification-service`

Это сделано намеренно:

- `catalog-service` остается фасадом для локального каталога и поиска
- `api-integration-service` остается фасадом для runtime item/pricing data
- gateway не смешивает локальный `catalog item id` и внешний `external id` под одним route namespace

## Конфигурация

- `AUTH_SERVICE_URL`
- `USER_ASSET_SERVICE_URL`
- `CATALOG_SERVICE_URL`
- `API_INTEGRATION_SERVICE_URL`
- `NOTIFICATION_SERVICE_URL`
- `JWT_SECRET`

## Проверка

```bash
cd services/api-gateway
GOCACHE=/tmp/gocache-gateway go test ./...
```
