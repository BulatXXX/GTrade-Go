# catalog-service docs

Документация по проектированию, локальной разработке и будущей ручной проверке `catalog-service`.

## Назначение

`catalog-service` отвечает за каноническую модель предметов внутри системы.

На первом этапе сервис должен уметь:

- сохранять предмет
- обновлять предмет
- удалять предмет
- получать предмет по id
- выдавать список предметов
- искать предметы по названию
- поддерживать локализации имени и описания
- хранить ежедневную историю цен по предметам

`catalog-service` должен быть источником истины для item metadata и не должен сводиться к простому проксированию внешних API.

## Базовая модель

```go
type Item struct {
	ID           string
	Game         string
	Source       string
	ExternalID   string
	Slug         string
	Name         string
	Description  string
	ImageURL     string
	IsActive     bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
	Translations []ItemTranslation
}

type ItemTranslation struct {
	ItemID       string
	LanguageCode string
	Name         string
	Description  string
}

type PriceHistoryEntry struct {
	ItemID      string
	Source      string
	GameMode    string
	Value       float64
	Currency    string
	CollectedOn string
	CollectedAt time.Time
}
```

## Что означает каждое поле

### Item

- `ID` — внутренний уникальный идентификатор предмета в системе.
- `Game` — игра, к которой относится предмет, например `warframe`, `eve`, `tarkov`.
- `Source` — источник данных внутри конкретной игры, например marketplace, API или import channel.
- `ExternalID` — идентификатор предмета во внешнем источнике.
- `Slug` — стабильный человекочитаемый идентификатор для поиска и внутренних ссылок.
- `Name` — базовое имя предмета на основном языке.
- `Description` — базовое описание предмета на основном языке.
- `ImageURL` — ссылка на изображение предмета.
- `IsActive` — признак, что предмет активен в каталоге и не архивирован.
- `CreatedAt` — время создания записи.
- `UpdatedAt` — время последнего изменения записи.
- `Translations` — список локализованных значений имени и описания.

### ItemTranslation

- `ItemID` — ссылка на предмет.
- `LanguageCode` — код языка, например `en`, `ru`, `de`.
- `Name` — локализованное имя предмета.
- `Description` — локализованное описание предмета.

## Правила модели

- каноническая сущность хранится в `items`
- локализации хранятся отдельно в `item_translations`
- уникальность предмета должна задаваться по `game + source + external_id`
- обычное удаление на текущем этапе трактуется как деактивация через `IsActive=false`
- hard delete допустим только как внутренний административный сценарий
- история цен хранится отдельно в `prices` и пишет не сырые market snapshots, а единое daily top-price значение
- для одного `item_id + source + game_mode + day` хранится одна запись, повторный сбор за день обновляет ее

## План первого MVP

Минимальные операции сервиса:

- `CreateItem`
- `UpdateItem`
- `DeleteItem`
- `GetItemByID`
- `ListItems`
- `SearchItems`
- `UpsertTranslation`
- `DeleteTranslation`

Минимальные HTTP-сценарии:

- `POST /items`
- `POST /items/upsert`
- `PUT /items/:id`
- `DELETE /items/:id`
- `GET /items/:id`
- `GET /items`
- `GET /items/search?q=...&game=...&language=...`
- `GET /items/:id/prices/history?game_mode=...&limit=...`

Поддержка локализаций может быть оформлена либо отдельными endpoint'ами, либо как часть `create/update` payload, но хранение должно оставаться раздельным на уровне модели.

Получение локализованного ответа работает через query param `language`.

Примеры:

```bash
curl -sS 'http://localhost:8084/items/<ITEM_ID>?language=ru'
```

```bash
curl -sS 'http://localhost:8084/items?game=warframe&source=market&language=ru&limit=20&offset=0'
```

```bash
curl -sS 'http://localhost:8084/items/search?q=prime&game=warframe&language=ru&limit=20&offset=0'
```

В ответе при этом будут дополнительные поля:

- `localized_name`
- `localized_description`
- `localized_language`

Если перевод для указанного языка не найден, сервис делает fallback на базовые `name` и `description`.

Поиск работает так:

- `q` — обязательная строка поиска
- `game` — опциональный фильтр по игре
- `language` — опциональный код языка, например `ru` или `en`
- если `language` передан, сервис ищет по `item_translations.name` для этого языка и также оставляет fallback на базовое `items.name`
- по умолчанию поиск возвращает только активные предметы

Пример:

```bash
curl -sS 'http://localhost:8084/items/search?q=continuity&game=test&language=ru&limit=20&offset=0'
```

Для внутренних инструментов наполнения каталога доступен idempotent upsert:

```bash
curl -sS -X POST http://localhost:8084/items/upsert \
  -H 'Content-Type: application/json' \
  -d '{
    "game":"warframe",
    "source":"market",
    "external_id":"frost_prime_set",
    "slug":"frost_prime_set",
    "name":"Frost Prime Set"
  }'
```

`POST /items/upsert` используется `tools/catalog-importer` как ingestion endpoint.

Для чтения истории цен:

```bash
curl -sS 'http://localhost:8084/items/<ITEM_ID>/prices/history?limit=30'
```

Для Tarkov можно фильтровать по режиму игры:

```bash
curl -sS 'http://localhost:8084/items/<ITEM_ID>/prices/history?game_mode=pve&limit=30'
```

Текущий подтвержденный поток наполнения каталога такой:

- внешний источник отдает список предметов
- importer при необходимости догружает item card
- importer отправляет каждый предмет отдельно в `POST /items/upsert`
- базовые поля (`name`, `description`, `image_url`) пишутся в `items`
- локализации пишутся в `item_translations`

Для `warframe` сейчас подтвержден такой сценарий:

- `en` карточка наполняет `items`
- `ru` карточка наполняет `item_translations`

История цен обновляется самим `catalog-service` по таймеру:

- сервис проходит по активным предметам каталога
- для каждого предмета вызывает `api-integration-service` `GET /items/:external_id/top-price`
- для `tarkov` собирает обе ветки `regular` и `pve`
- сохраняет одну daily запись на предмет и режим

Полезные env для этого потока:

- `INTEGRATION_SERVICE_URL` — базовый URL `api-integration-service`, по умолчанию `http://localhost:8083`
- `PRICE_HISTORY_REFRESH_INTERVAL` — период фонового обновления, по умолчанию `24h`

Документация по реальному импорту лежит в:

- `tools/catalog-importer/README.md`

## Swagger / OpenAPI

Текущая OpenAPI-схема сервиса лежит в:

- `services/catalog-service/docs/openapi.yaml`

Это зафиксированный контракт первого MVP каталога. По мере реализации код должен приводиться к этой схеме или схема должна обновляться вместе с изменением контракта.

## Порядок разработки

Для `catalog-service` фиксируем такой порядок работы:

1. сначала пишем тесты
2. затем пишем код
3. затем прогоняем тесты и подтверждаем поведение

Это относится к service layer, repository layer и HTTP handlers.

## Технологические ограничения

При реализации нужно соблюдать общий стек проекта из `docs/requirements/stack.md`:

- HTTP: Gin
- Logging: zerolog
- Database: pgx
- Migrations: golang-migrate
- Validation: validator
- Testing: testing + testify

## Middleware

Сервис должен использовать общие middleware из `shared/httpmiddleware`.

На текущий момент в роутере уже используются:

- `RequestID`
- `RequestLogger`

JWT-auth middleware для `catalog-service` сейчас не требуется. На текущем этапе сервис рассматривается как внутренний доменный сервис, и авторизация не должна тормозить фиксацию и реализацию основного каталожного контракта.

## Локальный запуск и тестирование

Подготовить env:

```bash
cp deploy/.env.example deploy/.env
```

Поднять только `catalog-service` и его PostgreSQL:

```bash
make catalog-up
make catalog-logs
```

Остановить:

```bash
make catalog-down
```

Поднять только БД каталога:

```bash
make catalog-db-up
```

Остановить только БД каталога:

```bash
make catalog-db-down
```

Все тесты сервиса:

```bash
make catalog-test
```

Только интеграционные тесты repository layer с реальной PostgreSQL:

```bash
make catalog-test-integration
```

Локальная сборка:

```bash
make catalog-build
```

Backup базы каталога в `pg_dump` custom format:

```bash
make catalog-backup
```

При необходимости можно указать свой путь:

```bash
make catalog-backup BACKUP_FILE=backups/catalog-manual.dump
```

Restore из backup:

```bash
make catalog-restore BACKUP_FILE=backups/catalog-manual.dump
```

`catalog-restore` использует `pg_restore --clean --if-exists`, то есть перед восстановлением очищает существующие объекты в целевой БД.

Эти команды используют `pg_dump` и `pg_restore` внутри контейнера `postgres-catalog`, чтобы избежать несовместимости версий локального PostgreSQL-клиента и серверного PostgreSQL.

При необходимости прямой локальный прогон без `make`:

```bash
cd services/catalog-service && TEST_DATABASE_URL='postgres://gtrade:gtrade@localhost:5436/gtrade_catalog?sslmode=disable' GOCACHE=/tmp/gocache-catalog go test ./...
```

## Текущее состояние

Сейчас в сервисе уже есть:

- `GET /health`
- общий HTTP-каркас
- shared middleware для request id и logging
- рабочие route'ы:
  - `POST /items`
  - `POST /items/upsert`
  - `PUT /items/:id`
  - `DELETE /items/:id`
  - `GET /items/:id`
  - `GET /items`
  - `GET /items/search`
- service layer с валидацией входных данных
- PostgreSQL repository для `items` и `item_translations`
- soft delete через `is_active=false`

Следующим шагом остается расширение тестового покрытия HTTP-слоя и при необходимости добавление отдельных endpoint'ов для локализаций.
