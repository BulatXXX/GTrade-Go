# Сервисы

## api-gateway
- Назначение: внешний вход в API и маршрутизация запросов.
- Порт: `8080`
- Основные endpoint'ы:
  - `GET /health`
  - группы маршрутов: `/api/auth`, `/api/users`, `/api/items`, `/api/notifications`


## auth-service
- Назначение: аутентификация и account lifecycle.
- Порт: `8081`
- Основные endpoint'ы:
  - `GET /health`
  - `POST /register`
  - `POST /login`
  - `POST /refresh`
  - `POST /password/reset/request`
  - `POST /password/reset/confirm`
  - `POST /email/verify`
- Что хранит: `users`.

## user-asset-service
- Назначение: watchlist и пользовательские настройки.
- Порт: `8082`
- Основные endpoint'ы:
  - `GET /health`
  - `GET /watchlist`
  - `POST /watchlist`
  - `DELETE /watchlist/:id`
  - `GET /recent`
  - `GET /preferences`
  - `PUT /preferences`
- Что хранит: `watchlist_items`.

## api-integration-service
- Назначение: интеграция с внешними маркетплейсами и адаптеры.
- Порт: `8083`
- Основные endpoint'ы:
  - `GET /health`
  - `GET /search`
  - `GET /items/:id`
  - `GET /items/:id/top-price`
- Что хранит: не хранит состояние в текущем skeleton.

## catalog-service
- Назначение: каталог предметов и цены.
- Порт: `8084`
- Основные endpoint'ы:
  - `GET /health`
  - `GET /items`
  - `GET /items/:id`
  - `GET /items/search`
  - `POST /items/upsert`
- Что хранит: `items`, `prices`.

## notification-service
- Назначение: отправка уведомлений и интеграция с email-провайдерами.
- Порт: `8085`
- Основные endpoint'ы:
  - `GET /health`
  - `POST /send-email`
- Что хранит: `notification_outbox`.

## tools/catalog-importer
- Назначение: CLI-импорт каталога из внешних источников.
- Порт: не применяется.
- Команда:
  - `catalog-importer -source warframe|eve|tarkov`
- Что хранит: напрямую не хранит, пишет через repository abstraction.
