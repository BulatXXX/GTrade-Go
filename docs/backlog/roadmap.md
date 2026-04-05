# Roadmap

Текущий `roadmap` используется как backlog улучшений для уже работающего локального контура.

Он не дублирует описание текущей архитектуры и не используется как bug tracker.

Для этого есть отдельные документы:

- ошибки и дефекты: `docs/backlog/bug-log.md`
- текущее устройство системы: `docs/requirements/architecture.md`
- smoke-проверка: `docs/scenarios/smoke-scenarios.md`

## Что уже есть

На текущем этапе уже работают:

- `api-gateway` как внешний фасад
- `auth-service`
- `user-asset-service`
- `catalog-service`
- `api-integration-service`
- `notification-service`
- `catalog-importer`

Уже собраны ключевые связи:

- `auth-service -> notification-service`
- `user-asset-service -> catalog-service`
- `api-integration-service -> catalog-service`
- `api-gateway -> domain services`

## Ближайший backlog

### 1. Internal auth rollout

Что нужно:

- довести internal auth до всех внутренних service-to-service сценариев
- определить единый подход к internal token rotation и rollout
- закрыть не только sync flow, но и остальные чувствительные внутренние ручки

Почему это важно:

- сейчас система уже работает как многосервисный контур
- безопасность внутренних маршрутов больше нельзя откладывать

### 2. Scheduler и sync automation

Что нужно:

- определить формат scheduler/runner для регулярного sync
- решить, какие сценарии нужны:
  - full sync
  - incremental sync
  - per-game sync
  - per-item sync
- не смешивать scheduler с runtime API

Почему это важно:

- sync flow уже существует
- не хватает только автоматизации поверх него

### 3. Backup flow перед full sync

Что нужно:

- определить, как делать backup каталога перед массовыми обновлениями
- зафиксировать, когда backup обязателен, а когда нет
- не делать тяжелый backup на каждый одиночный item sync

Почему это важно:

- full sync без rollback story опасен для локального source of truth

### 4. Pricing history и analytics storage

Что нужно:

- решить, где хранить historical pricing snapshots
- определить минимальный набор аналитических метрик
- не смешивать history storage с metadata catalog без явной причины

Почему это важно:

- runtime pricing уже есть
- для дашбордов не хватает истории

### 5. Gateway hardening

Что нужно:

- зафиксировать публичный API-contract через gateway
- убрать лишние временные допущения
- при необходимости добавить более явную auth-aware интеграцию с `user-asset-service`

Почему это важно:

- gateway уже работает
- теперь его нужно полировать, а не заново строить

### 6. User-state cleanup

Что нужно:

- убрать лишнюю зависимость от ручной передачи `user_id` после полного auth-aware flow
- решить, нужен ли кроме базового watchlist еще один тип пользовательских списков
- определить, нужен ли enrichment pricing summary прямо в user responses

Почему это важно:

- базовый user-state уже есть
- дальше нужны именно продуктовые улучшения

## Не в приоритете сейчас

Пока не первоочередные задачи:

- swagger-ui как отдельный слой поверх OpenAPI
- новый сервис ради scheduler, если можно обойтись CLI/job runner
- расширение каталога ради хранения volatile pricing data
- frontend-специфические улучшения внутри backend backlog

## Как обновлять этот файл

Сюда попадают:

- задачи улучшения
- архитектурные следующие шаги
- hardening и product backlog

Сюда не попадают:

- баги и дефекты
- подробные smoke-результаты
- дублирование OpenAPI контрактов
