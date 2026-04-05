# GTrade Data System

GTrade Data System — платформа управления данными внутриигровых торговых площадок. На текущем этапе это локально поднимаемый backend-контур из нескольких Go-сервисов с единым gateway, локальным каталогом, runtime-интеграциями с внешними API и пользовательским cloud-state.

## Структура репозитория

- `services/` — Go-микросервисы
- `tools/` — CLI-утилиты
- `docs/` — требования, backlog, сценарии и шаблоны
- `deploy/` — `docker-compose`, env и Dockerfile для локального запуска

## Технологический стек

- Go
- Gin
- PostgreSQL
- pgx pool
- resty
- zerolog
- Docker / docker-compose

## Локальный запуск

1. Подготовить env:

```bash
cp deploy/.env.example deploy/.env
```

2. Поднять весь стек:

```bash
make up
```

3. Проверить health:

- `http://localhost:8080/health`
- `http://localhost:8081/health`
- `http://localhost:8082/health`
- `http://localhost:8083/health`
- `http://localhost:8084/health`
- `http://localhost:8085/health`

4. Остановить:

```bash
make down
```

## Отдельные контуры

Только `notification-service` и его PostgreSQL:

```bash
make notification-up
make notification-logs
make notification-down
```

Только `auth-service` и его PostgreSQL:

```bash
make auth-up
make auth-logs
make auth-down
```

Живой e2e-контур `auth-service -> notification-service`:

```bash
make auth-notification-e2e-test
```

## Сервисы

### `api-gateway`

- Назначение: внешний вход в систему и маршрутизация запросов.
- Порт: `8080`
- Состояние: рабочий gateway-фасад без собственной БД.
- Публичные группы маршрутов:
  - `/api/auth/*`
  - `/api/users/*`
  - `/api/items/*`
  - `/api/market/*`
  - `/api/notifications/*`
- Особенности:
  - проксирует доменные сервисы
  - защищает JWT-мидлварью `/api/users/*` и `/api/notifications/*`

### `auth-service`

- Назначение: аутентификация и account lifecycle.
- Порт: `8081`
- Состояние: рабочий MVP+.
- Основные endpoint'ы:
  - `GET /health`
  - `POST /register`
  - `POST /login`
  - `POST /refresh`
  - `POST /password/reset/request`
  - `POST /password/reset/confirm`
  - `POST /email/verify`
- Хранит:
  - `users`
  - `refresh_tokens`
  - `password_reset_tokens`
  - `email_verification_tokens`
- Особенность:
  - токены reset/verification не отдаются наружу, а уходят через `notification-service`

### `user-asset-service`

- Назначение: пользовательский cloud-state.
- Порт: `8082`
- Состояние: рабочий service для profile/watchlist/preferences.
- Основные endpoint'ы:
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
- Хранит:
  - `user_profiles`
  - `watchlist_items`
  - `user_preferences`
- Особенности:
  - хранит `display_name`, `avatar_url`, `bio`
  - валидирует и enrich'ит item refs через `catalog-service`

### `api-integration-service`

- Назначение: runtime-доступ к внешним API и sync metadata в каталог.
- Порт: `8083`
- Состояние: рабочий integration-layer без собственной БД.
- Основные endpoint'ы:
  - `GET /health`
  - `GET /search`
  - `GET /items/:id`
  - `GET /items/:id/prices`
  - `GET /items/:id/top-price`
  - `POST /internal/sync/item`
  - `POST /internal/sync/search`
- Особенности:
  - нормализует внешние item/pricing ответы в единый DTO
  - поддерживает `warframe`, `eve`, `tarkov`
  - для `tarkov` поддерживает `game_mode=regular|pve`
  - internal sync routes закрыты через `X-Internal-Token`

### `catalog-service`

- Назначение: канонический локальный каталог предметов.
- Порт: `8084`
- Состояние: рабочий metadata-service на PostgreSQL.
- Основные endpoint'ы:
  - `GET /health`
  - `POST /items`
  - `GET /items`
  - `GET /items/:id`
  - `GET /items/search`
  - `POST /items/upsert`
  - `PUT /items/:id`
  - `DELETE /items/:id`
- Хранит:
  - `items`
  - `item_translations`
  - `prices`
- Особенности:
  - локальный поиск идет по PostgreSQL
  - `POST /items/upsert` используется importer'ом и integration sync flow

### `notification-service`

- Назначение: отправка email и outbox delivery.
- Порт: `8085`
- Состояние: рабочий notification-service.
- Основные endpoint'ы:
  - `GET /health`
  - `POST /send-email`
- Хранит:
  - `notification_outbox`
- Особенности:
  - поддерживает `mock` provider
  - поддерживает `Resend`

### `tools/catalog-importer`

- Назначение: пакетный импорт каталога из внешних источников.
- Формат запуска:
  - `catalog-importer -source warframe|eve|tarkov`
- Состояние: рабочий importer для `warframe`, `eve`, `tarkov`.
- Особенности:
  - потоковый импорт предметов
  - пишет базовые `en` поля и `ru` локализации

## Что уже реализовано

- рабочий многосервисный backend-контур
- единый shared middleware: request id, logging, JWT validation, internal token validation
- связка `auth-service -> notification-service`
- связка `user-asset-service <-> catalog-service`
- связка `api-integration-service <-> catalog-service`
- `api-gateway -> domain services`
- runtime pricing и metadata fetch для `warframe`, `eve`, `tarkov`
- локальный каталог с search и локализациями
- smoke, unit, integration и частично live e2e тесты по основным контурам

## Что еще не закрыто

- полный rollout internal auth для всех внутренних чувствительных route'ов
- scheduler/runner для регулярного sync
- backup flow перед full catalog sync
- historical pricing storage для аналитики и дашбордов
- frontend живет отдельно и не входит в этот репозиторий

## Порты

Сервисы:

- `api-gateway`: `8080`
- `auth-service`: `8081`
- `user-asset-service`: `8082`
- `api-integration-service`: `8083`
- `catalog-service`: `8084`
- `notification-service`: `8085`

PostgreSQL:

- `auth`: `5433`
- `user-asset`: `5434`
- `catalog`: `5436`
- `notification`: `5437`

`api-gateway` и `api-integration-service` работают без собственной БД.

## Документация

Требования и архитектура:

- `docs/requirements/TZ.md`
- `docs/requirements/architecture.md`
- `docs/requirements/stack.md`

Текущее управление работой:

- `docs/backlog/roadmap.md`
- `docs/backlog/bug-log.md`
- `docs/scenarios/smoke-scenarios.md`

OpenAPI:

- `services/auth-service/docs/openapi.yaml`
- `services/user-asset-service/docs/openapi.yaml`
- `services/catalog-service/docs/openapi.yaml`
- `services/api-integration-service/docs/openapi.yaml`
- `services/notification-service/docs/openapi.yaml`

Сервисные README:

- `services/api-gateway/README.md`
- `services/api-integration-service/README.md`
- `services/catalog-service/README.md`
- `services/user-asset-service/README.md`
- `services/notification-service/README.md`

Инструменты:

- `tools/catalog-importer/README.md`
