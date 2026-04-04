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
- поддерживать локализации имени и описания

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

## План первого MVP

Минимальные операции сервиса:

- `CreateItem`
- `UpdateItem`
- `DeleteItem`
- `GetItemByID`
- `ListItems`
- `UpsertTranslation`
- `DeleteTranslation`

Минимальные HTTP-сценарии:

- `POST /items`
- `PUT /items/:id`
- `DELETE /items/:id`
- `GET /items/:id`
- `GET /items`

Поддержка локализаций может быть оформлена либо отдельными endpoint'ами, либо как часть `create/update` payload, но хранение должно оставаться раздельным на уровне модели.

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

При реализации нужно соблюдать общий стек проекта из `docs/stack.md`:

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

Специальных `make`-таргетов для `catalog-service` в текущем `Makefile` пока нет.

До их добавления нужно исходить из следующих правил:

- сначала добавить тесты под согласованный контракт
- затем реализовать минимальные миграции и service/repository/handler слои
- после этого добавить отдельные `make`-таргеты по шаблону из `docs/make-targets-template.md`

Пока из корня репозитория доступны только общие команды, например:

```bash
make build
```

Для локальной проверки самого сервиса после появления тестов и реализации можно использовать:

```bash
cd services/catalog-service && go test ./...
```

## Текущее состояние

Сейчас в сервисе уже есть:

- `GET /health`
- общий HTTP-каркас
- shared middleware для request id и logging
- placeholder route'ы:
  - `GET /items`
  - `GET /items/:id`
  - `GET /items/search`
  - `POST /items/upsert`

Бизнес-логика каталога, миграции предметной модели и OpenAPI-совместимая реализация еще не сделаны.
