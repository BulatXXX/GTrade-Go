# Roadmap

Текущий roadmap проекта GTrade с `service-first` стратегией:

1. сначала довести самостоятельные сервисы
2. потом собрать прямые межсервисные интеграции
3. потом подключить `api-gateway` как внешний фасад
4. и только после этого заниматься глубокой полировкой

## Главный принцип

Сейчас не нужно пытаться строить продукт через `api-gateway`, если сами доменные сервисы еще не доведены до минимально рабочего состояния.

Сейчас правильнее:

- сначала сделать сервисы самостоятельными и полезными
- затем проверить их прямые интеграции
- и только потом надевать поверх них `api-gateway`

Такой подход уменьшает хаос и не превращает gateway в слой, который компенсирует сырые сервисы.

## Что уже достаточно готово

### auth-service

Уже доведен до хорошего MVP-состояния:

- register/login/refresh
- password reset
- email verification
- PostgreSQL persistence
- интеграция с `notification-service`
- токены reset/verify больше не торчат в публичном API

### notification-service

Уже доведен до хорошего supporting-service состояния:

- `POST /send-email`
- PostgreSQL outbox
- `mock` provider
- `Resend` provider
- repeatable тестовый контур

### user-asset-service

Уже доведен до хорошего user-state MVP-состояния:

- profile / watchlist / preferences CRUD
- строковый `item_id`, совместимый с `catalog-service`
- валидация и enrichment watchlist через `catalog-service`
- unit / smoke / integration tests

### catalog-service

Уже доведен до самостоятельного metadata-MVP-состояния:

- CRUD и upsert предметов
- локальный поиск по каталогу
- локализации через `item_translations`
- рабочий importer flow для `warframe`, `eve`, `tarkov`

## Этап 1. Довести оставшиеся сервисы до самостоятельного MVP

Главная цель этапа:

- каждый ключевой сервис должен уметь выполнять свою доменную задачу без участия `api-gateway`

### Приоритет 1. catalog-service

Что уже сделано:

- рабочие `POST /items`, `PUT /items/:id`, `DELETE /items/:id`
- рабочие `GET /items`, `GET /items/:id`, `GET /items/search`
- ingestion endpoint `POST /items/upsert`
- PostgreSQL persistence для `items` и `item_translations`
- локализованный поиск по `name` и `translations.name`
- backup/restore команды
- рабочий importer flow `warframe|eve|tarkov -> catalog-service`

Что осталось:

- HTTP tests для handler/router слоя
- более строгая DTO-валидация
- уточнение модели `prices`, если цены будут храниться внутри каталога

Статус:

- `catalog-service` уже доведен до самостоятельного MVP

### Приоритет 2. api-integration-service

Что уже сделано:

- рабочий provider registry и service layer
- `warframe` search / item / pricing
- `eve` item / pricing
- `tarkov` search / item / pricing
- normalized endpoint'ы `GET /search`, `GET /items/:id`, `GET /items/:id/prices`, `GET /items/:id/top-price`
- internal sync endpoint'ы `POST /internal/sync/item`, `POST /internal/sync/search`
- поддержка `tarkov game_mode=regular|pve`
- service/unit/provider/HTTP tests
- рабочая связка `api-integration-service -> catalog-service` через `POST /items/upsert`

Что осталось:

- решить, нужен ли storage для historical pricing snapshots и analytics
- решить, нужен ли scheduler/runner для регулярного catalog sync
- определить backup flow перед full sync
- при необходимости расширить pricing endpoint'ы под dashboard-specific метрики
- добавить internal auth для будущих sync/internal endpoint'ов

Цель:

- получить самостоятельный сервис, который уже реально ходит во внешние источники и становится слоем нормализации внешних item/pricing данных

### Приоритет 3. user-asset-service

Что уже сделано:

- полный минимальный watchlist flow
- понятный profile/preferences flow
- связка watchlist с item ids из каталога
- profile fields: `display_name`, `avatar_url`, `bio`
- integration tests с реальной PostgreSQL
- OpenAPI/Swagger документация

Что осталось:

- решить, нужны ли отдельные list entity кроме одного watchlist
- подготовить auth-aware flow, чтобы убрать `user_id` из query/body после подключения gateway/auth
- при необходимости добавить более богатый enrichment item data или pricing summary

Цель:

- получить самостоятельный сервис пользовательского состояния, который уже можно использовать в MVP

## Этап 2. Прямые межсервисные интеграции

Главная цель этапа:

- связать уже рабочие сервисы напрямую там, где это действительно нужно по бизнес-логике

### Уже сделано

- `auth-service -> notification-service`
- `user-asset-service -> catalog-service`
- `api-integration-service -> catalog-service`

### Следующие кандидаты

- `user-asset-service <-> catalog-service` при необходимости
- scheduler/runner и backup flow вокруг `api-integration-service -> catalog-service`

Важно:

- на этом этапе интеграции должны быть service-to-service
- без обязательного участия `api-gateway`

Цель:

- собрать реальную внутреннюю систему из работающих компонентов

## Этап 3. Подключить api-gateway

Только после того как доменные сервисы уже рабочие:

- gateway уже переведен из placeholder в рабочий facade/proxy слой
- следующий шаг для него не базовое проксирование, а hardening и public API polishing

Какой должна быть роль gateway:

- внешний фасад
- единая входная точка
- транспортный слой

Чего не стоит делать:

- не превращать `api-gateway` в место, где живет доменная бизнес-логика
- не использовать gateway как компенсацию за недоделанные сервисы

## Этап 4. Полировка и hardening

Только после того как система уже реально работает end-to-end:

### auth-service

- защита от user enumeration
- rate limiting / throttling
- logout / revoke flow
- `GET /me`
- более строгая валидация DTO

### notification-service

- внутренняя защита `POST /send-email`
- retry/worker модель
- шаблоны писем
- улучшение error model

### gateway / platform

- internal auth между сервисами
- лучшая observability
- более зрелые e2e сценарии
- swagger-ui и дополнительные DX-улучшения

## Что делать следующим практически

Если идти в правильном порядке, следующий шаг сейчас не базовый `api-gateway`, а hardening вокруг уже собранных связок.

Следующий шаг:

1. определить, нужен ли scheduler/runner для регулярного sync и backup flow
2. решить storage для historical pricing snapshots
3. добавить internal auth для внутренних endpoint'ов и sync задач
4. после этого полировать публичный API gateway

Рекомендуемый порядок:

1. scheduler/runner и backup flow вокруг `api-integration-service`
2. дополнительные runtime/sync сценарии поверх `user-asset-service`
3. потом hardening `api-gateway`

## Какой MVP-контур хотим получить в итоге

После этапов 1-3 рабочий пользовательский путь должен выглядеть так:

1. пользователь логинится
2. пользователь ищет предмет
3. система получает item data
4. пользователь добавляет предмет в watchlist
5. система хранит его пользовательское состояние
6. внешний вход в этот сценарий уже идет через `api-gateway`

Но важно:

- сначала должны заработать сами сервисы
- и только потом gateway собирает их в единый публичный контур

## Что уже можно использовать прямо сейчас

Поднять `notification-service` отдельно:

```bash
make notification-up
```

Прогнать тесты `notification-service`:

```bash
make notification-test
```

Прогнать тесты `auth-service`:

```bash
make auth-test
```

Прогнать живой e2e контур между `auth-service` и `notification-service`:

```bash
make auth-notification-e2e-test
```

## Итог

Текущая стратегия проекта:

- не gateway-first
- не polishing-first
- а service-first

То есть:

1. расширяем уже собранные сервисы и их прямые интеграции
2. собираем недостающие service-to-service связи
3. подключаем `api-gateway`
4. только потом глубоко полируем систему
