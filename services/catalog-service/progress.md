# catalog-service

Сервис каталога предметов GTrade.

## Что уже готово

- HTTP-сервер на Gin
- загрузка конфигурации из env
- подключение к PostgreSQL через repository layer
- endpoint `GET /health`
- endpoint `POST /items`
- endpoint `POST /items/upsert`
- endpoint `PUT /items/:id`
- endpoint `DELETE /items/:id`
- endpoint `GET /items/:id`
- endpoint `GET /items`
- endpoint `GET /items/search`
- каноническая модель `Item`
- поддержка локализаций через `item_translations`
- soft delete через `is_active=false`
- уникальность предмета по `game + source + external_id`
- поиск по базовому `items.name`
- поиск по `item_translations.name` с учетом `language`
- shared middleware: `RequestID`, `RequestLogger`
- unit tests для service layer
- integration tests для repository layer
- локальная документация и OpenAPI в `docs/`
- `make`-команды для сборки, запуска и тестирования
- `make`-команды для backup и restore каталога
- рабочий внешний importer в `tools/catalog-importer`
- проверенный live import `warframe -> catalog-service`

## Готовые endpoint'ы

- `GET /health`
- `POST /items`
- `POST /items/upsert`
- `PUT /items/:id`
- `DELETE /items/:id`
- `GET /items/:id`
- `GET /items`
- `GET /items/search?q=...&game=...&language=...`

## Текущий MVP flow

- `POST /items` создает предмет в `items` и при наличии сохраняет локализации в `item_translations`
- `POST /items/upsert` создает или обновляет предмет по `game + source + external_id`
- `PUT /items/:id` обновляет базовые поля предмета и при наличии заменяет локализации
- `DELETE /items/:id` не удаляет запись физически, а деактивирует ее через `is_active=false`
- `GET /items/:id` возвращает предмет вместе с локализациями
- `GET /items` возвращает список предметов с фильтрами по `game`, `source`, `active_only`
- `GET /items/search` ищет по базовому имени и по локализованному имени для указанного языка
- `tools/catalog-importer` использует `POST /items/upsert` как ingestion endpoint
- для `warframe` базовые `name` и `description` пишутся в `items`, локализованные значения пишутся в `item_translations`

## Что нужно доделать

- HTTP tests для handler/router слоя
- более строгую DTO-валидацию через `validator`
- уточнение публичной семантики `active_only`
- `total` и, возможно, дополнительные pagination metadata в list/search ответах
- индексы под более эффективный поиск по `name` и `item_translations.name`
- отдельный `progressive` ranking/search behavior при необходимости
- отдельные endpoint'ы для translations, если потребуется независимое управление переводами
- internal auth между сервисами, если `catalog-service` станет частью прямого межсервисного контура

## Ключевые файлы

- `internal/service/service.go` — бизнес-логика каталога
- `internal/repository/catalog_repository.go` — PostgreSQL repository для `items` и `item_translations`
- `internal/handler/items.go` — HTTP handlers для CRUD и search
- `internal/model/model.go` — request/response DTO и доменные модели
- `internal/http/service_routes.go` — маршруты сервиса
- `migrations/0001_init.sql` — схема `items`, `item_translations`, `prices`
- `docs/README.md` — локальный запуск, ручная проверка и тесты
- `docs/openapi.yaml` — актуальный OpenAPI/Swagger контракт
- `../tools/catalog-importer/README.md` — гайд по наполнению каталога внешними данными

## Следующий шаг

- закрыть HTTP-тестами текущий CRUD и search
- затем усилить валидацию DTO
- потом решить, нужна ли расширенная поисковая семантика и дополнительная pagination metadata

## Текущие ограничения

- нет HTTP smoke/integration tests
- нет `validator`-based DTO validation
- `active_only=false` сейчас означает фактически выборку неактивных записей, а не "вернуть все"
- поиск использует простой `ILIKE`, без ranking и без полнотекстового поиска
- endpoint'ы управления translations отдельно не выделены
- JWT-auth для `catalog-service` пока не используется

## Тесты

- unit tests сервисного слоя лежат в `internal/service/service_test.go`
- integration tests repository layer лежат в `internal/repository/catalog_repository_integration_test.go`
- importer guide и live import flow описаны в `tools/catalog-importer/README.md`

Проверка:

```bash
make catalog-test
```

Только интеграционные тесты с реальной PostgreSQL:

```bash
make catalog-test-integration
```

Поднять сервис локально:

```bash
make catalog-up
```
