# Сервисы

## api-gateway
- Назначение: внешний вход в API и маршрутизация запросов.
- Порт: `8080`
- Текущее состояние: placeholder, без реального proxy/service-client flow.
- Основные endpoint'ы:
  - `GET /health`
  - группы маршрутов: `/api/auth`, `/api/users`, `/api/items`, `/api/notifications`


## auth-service
- Назначение: аутентификация и account lifecycle.
- Порт: `8081`
- Текущее состояние: рабочий MVP+, интегрирован с `notification-service`.
- Основные endpoint'ы:
  - `GET /health`
  - `POST /register`
  - `POST /login`
  - `POST /refresh`
  - `POST /password/reset/request`
  - `POST /password/reset/confirm`
  - `POST /email/verify`
- Что хранит: `users`, `refresh_tokens`, `password_reset_tokens`, `email_verification_tokens`.
- Особенность: reset/verification токены больше не возвращаются в публичном API, а отправляются через `notification-service`.

## user-asset-service
- Назначение: watchlist и пользовательские настройки.
- Порт: `8082`
- Текущее состояние: базовый CRUD поверх PostgreSQL.
- Основные endpoint'ы:
  - `GET /health`
  - `POST /users`
  - `GET /users/:id`
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
- Текущее состояние: рабочий runtime integration service без собственной БД.
- Основные endpoint'ы:
  - `GET /health`
  - `GET /search`
  - `GET /items/:id`
  - `GET /items/:id/prices`
  - `GET /items/:id/top-price`
- Что хранит: не хранит состояние в текущем этапе.
- Особенность:
  - нормализует внешние item/pricing ответы в единый DTO
  - для `tarkov` поддерживает `game_mode=regular|pve`, по умолчанию `regular`
  - для `warframe` умеет искать предметы, получать item card и цены через `warframe.market`
  - для `eve` получает item card и цены через `ESI`, а поиск должен идти через локальный `catalog-service`
  - `GET /items/:id/prices` возвращает полный normalized pricing snapshot
  - `GET /items/:id/top-price` возвращает сокращенный ответ с одним главным значением цены

## catalog-service
- Назначение: канонический каталог предметов, локализации и локальный поиск.
- Порт: `8084`
- Текущее состояние: рабочий metadata-service с PostgreSQL.
- Основные endpoint'ы:
  - `GET /health`
  - `POST /items`
  - `GET /items`
  - `GET /items/:id`
  - `GET /items/search`
  - `POST /items/upsert`
  - `PUT /items/:id`
  - `DELETE /items/:id`
- Что хранит: `items`, `item_translations`, `prices`.
- Особенность:
  - поиск идет по локальной PostgreSQL, а не по внешним API
  - при `language=...` ищет по `item_translations.name` и возвращает `localized_name` / `localized_description`
  - `POST /items/upsert` используется `catalog-importer` для идемпотентного наполнения каталога

## notification-service
- Назначение: отправка уведомлений и интеграция с email-провайдерами.
- Порт: `8085`
- Текущее состояние: рабочий сервис с PostgreSQL outbox.
- Основные endpoint'ы:
  - `GET /health`
  - `POST /send-email`
- Что хранит: `notification_outbox`.
- Особенность: поддерживает `mock` provider для стабильных системных тестов и `Resend` для живой отправки.

## tools/catalog-importer
- Назначение: CLI-импорт каталога из внешних источников.
- Порт: не применяется.
- Команда:
  - `catalog-importer -source warframe|eve|tarkov`
- Что хранит: напрямую не хранит, пишет через repository abstraction.
- Текущее состояние: рабочий importer для `warframe`, `eve`, `tarkov`.
- Особенность:
  - импорт идет потоково по одному предмету
  - базовые `en` поля пишутся в `items`
  - `ru` локализации пишутся в `item_translations`
  - для `tarkov` используется GraphQL API `api.tarkov.dev`

## Реальный системный тест сейчас

На текущем этапе есть live e2e-контур между `auth-service` и `notification-service`:

```bash
make auth-notification-e2e-test
```

Он поднимает реальные контейнеры сервисов и их PostgreSQL и проверяет, что `auth-service` создает записи в `notification_outbox`.
