# GTrade Data System (GTrade)

GTrade Data System — платформа управления данными внутриигровых торговых площадок.

## Структура репозитория

- `services/` — Go-микросервисы
- `tools/` — Go CLI-утилиты
- `docs/` — документация по архитектуре и сервисам
- `deploy/` — docker-compose и локальные deploy-артефакты

## Сервисы

- `api-gateway`
- `auth-service`
- `user-asset-service`
- `api-integration-service`
- `catalog-service`
- `notification-service`
- `tools/catalog-importer`

## Технологический стек

- Go
- Gin
- PostgreSQL
- pgx pool
- resty
- zerolog
- Docker / docker-compose

## Запуск локально

1. Скопировать env-файл:

```bash
cp deploy/.env.example deploy/.env
```

2. Поднять инфраструктуру:

```bash
make up
```

3. Проверить health endpoint'ы:

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

## Отдельные контуры запуска

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

## Порты сервисов

- api-gateway: `8080`
- auth-service: `8081`
- user-asset-service: `8082`
- api-integration-service: `8083`
- catalog-service: `8084`
- notification-service: `8085`

## Порты PostgreSQL

- auth DB: `5433`
- user-asset DB: `5434`
- catalog DB: `5436`
- notification DB: `5437`

`api-gateway` и `api-integration-service` в текущем skeleton работают без собственной БД.

## Что уже реализовано

- production-like skeleton для всех сервисов и утилиты
- единый каркас: конфиг из env, логирование, HTTP, repository layer
- health endpoint во всех сервисах
- единый shared middleware: request id, logging, JWT auth validation
- абстракции адаптеров маркетплейсов
- `auth-service` с рабочими flow: register, login, refresh, password reset, email verification
- интеграция `auth-service -> notification-service`
- `notification-service` с PostgreSQL outbox, `mock` provider и рабочей интеграцией с Resend
- `user-asset-service` с базовым CRUD для user/watchlist/preferences
- `catalog-service` с рабочими CRUD/search endpoint'ами, PostgreSQL persistence и локализациями
- `catalog-service` с ingestion endpoint `POST /items/upsert` для внешнего наполнения каталога
- локальный поиск по каталогу через PostgreSQL по `name` и `translations.name`
- `api-integration-service` с рабочими адаптерами `warframe`, `eve`, `tarkov`
- normalized runtime endpoint'ы `GET /search`, `GET /items/:id`, `GET /items/:id/prices`, `GET /items/:id/top-price`
- поддержка `tarkov game_mode=regular|pve`
- `tools/catalog-importer` с рабочим импортом `warframe`, `eve`, `tarkov`
- полный импорт metadata и `ru` локализаций для `warframe`, `eve`, `tarkov` уже подтвержден live-прогонами
- unit, integration и live e2e тесты для связки `auth-service -> notification-service`
- deploy-папка с docker-compose для локальной разработки на Mac

## Что пока заглушка

- реальный reverse proxy / service client flow в gateway
- полное покрытие защищенных внутренних route'ов auth middleware
- значительная часть `api-gateway`
- frontend вынесен в отдельный репозиторий

## Подход к данным

БД оставлены только в сервисах, где хранится состояние в текущем этапе:
`auth-service`, `user-asset-service`, `catalog-service`, `notification-service`.

## Полезные документы

- `docs/architecture.md` — текущее распределение ролей между сервисами
- `docs/services.md` — краткая сводка по endpoint'ам и состоянию сервисов
- `docs/roadmap.md` — общий roadmap и ближайшие приоритеты
- `services/auth-service/progress.md` — актуальный статус auth flow
- `services/notification-service/progress.md` — актуальный статус notification flow
- `services/catalog-service/progress.md` — актуальный статус catalog flow
- `services/api-integration-service/README.md` — актуальный статус integration flow
- `tools/catalog-importer/README.md` — гайд по импорту `warframe`, `eve`, `tarkov`
