# catalog-importer

CLI-утилита наполнения `catalog-service` данными из внешних источников.

## Текущий статус

Сейчас реально поддержаны источники:

- `warframe`
- `eve`

Для `tarkov` каркас источника создан, но fetch-логика еще не реализована.

## Что делает утилита

`catalog-importer` работает как внешний ingestion client для `catalog-service`.

Для каждого предмета поток такой:

1. получить предмет из внешнего источника
2. при необходимости догрузить item card
3. преобразовать внешний payload в каноническую модель каталога
4. отправить один предмет в `catalog-service` через `POST /items/upsert`

Импорт идет потоково, по одному предмету за раз. Это важно: предметы начинают появляться в БД сразу по ходу импорта, а не только в конце всего прогона.

Утилита рассчитана на idempotent импорт: повторный запуск должен обновлять существующие записи по `game + source + external_id`.

## Как данные ложатся в каталог

Для `warframe` используется такая схема:

- базовая карточка на `en` заполняет таблицу `items`
- локализованная карточка, например `ru`, заполняет `item_translations`

То есть:

- `items.name` и `items.description` берутся из `en`
- `item_translations.name` и `item_translations.description` берутся из выбранного `-language`

Если импортер запущен без `-language`, будут заполнены только базовые поля в `items`.

## Warframe

Источник `warframe` использует:

- список предметов: `GET https://api.warframe.market/v2/items`
- карточку предмета: `GET https://api.warframe.market/v2/items/{slug}`

Что импортируется:

- `slug`
- `name`
- `description`
- `image_url`
- локализованные `name`
- локализованные `description`

Что пока не импортируется:

- цены
- ордера
- рыночная статистика

## EVE

Источник `eve` использует:

- список type id: `GET https://esi.evetech.net/latest/markets/prices/?datasource=tranquility`
- карточку типа: `GET https://esi.evetech.net/latest/universe/types/{type_id}/?datasource=tranquility&language=<lang>`
- изображение: `https://images.evetech.net/types/{type_id}/icon?size=128`

Что импортируется:

- `type_id` как `external_id`
- `name`
- `description`
- `image_url`
- локализованные `name`
- локализованные `description`

Что пока не импортируется:

- сами рыночные цены в отдельную таблицу
- история цен
- объемы рынка

Для `eve` используется:

- `game=eve`
- `source=esi`

Важно: `markets/prices` сейчас нужен только как источник списка `type_id`. Цены из него пока не сохраняются, потому что текущий `catalog-service` хранит item metadata, а не market snapshots.

## Пошаговый гайд

1. Подготовить окружение и поднять `catalog-service`:

```bash
cp deploy/.env.example deploy/.env
make catalog-up
```

2. Проверить, что сервис жив:

```bash
curl -sS http://localhost:8084/health
```

3. Перейти в утилиту:

```bash
cd tools/catalog-importer
```

4. Запустить импорт.

Короткая проверка:

```bash
GOCACHE=/tmp/gocache-importer go run ./cmd/catalog-importer -source warframe -limit 10 -catalog-url http://localhost:8084
```

Короткая проверка с локализацией:

```bash
GOCACHE=/tmp/gocache-importer go run ./cmd/catalog-importer -source warframe -language ru -limit 10 -catalog-url http://localhost:8084
```

Полный импорт базового каталога:

```bash
GOCACHE=/tmp/gocache-importer go run ./cmd/catalog-importer -source warframe -catalog-url http://localhost:8084
```

Полный импорт с русской локализацией:

```bash
GOCACHE=/tmp/gocache-importer go run ./cmd/catalog-importer -source warframe -language ru -catalog-url http://localhost:8084
```

Короткая проверка EVE:

```bash
GOCACHE=/tmp/gocache-importer go run ./cmd/catalog-importer -source eve -limit 10 -catalog-url http://localhost:8084
```

Короткая проверка EVE с локализацией:

```bash
GOCACHE=/tmp/gocache-importer go run ./cmd/catalog-importer -source eve -language ru -limit 10 -catalog-url http://localhost:8084
```

5. Проверить результат через API.

Список:

```bash
curl -sS 'http://localhost:8084/items?game=warframe&source=market&limit=20&offset=0'
```

Поиск:

```bash
curl -sS 'http://localhost:8084/items/search?q=prime&game=warframe&limit=20&offset=0'
```

Локализованный поиск:

```bash
curl -sS 'http://localhost:8084/items/search?q=%D0%BF%D1%80%D0%B0%D0%B9%D0%BC&game=warframe&language=ru&limit=20&offset=0'
```

Поиск по EVE:

```bash
curl -sS 'http://localhost:8084/items/search?q=plagioclase&game=eve&language=ru&limit=20&offset=0'
```

Получение локализованной карточки:

```bash
curl -sS 'http://localhost:8084/items/<ITEM_ID>?language=ru'
```

## Прогресс импорта

Во время импорта утилита пишет progress в stdout.

Пример:

```text
catalog import progress: processed=25 game=warframe source=market slug=decurion_barrel
```

Это означает, что предмет уже не просто fetched, а уже прошел `upsert` в `catalog-service`.

## Проверка через БД

Проверить количество импортированных Warframe-предметов:

```bash
docker compose -f deploy/docker-compose.yml --env-file deploy/.env exec -T postgres-catalog \
  psql -U gtrade -d gtrade_catalog \
  -c "SELECT COUNT(*) FROM items WHERE game='warframe' AND source='market';"
```

Проверить, сколько предметов уже имеют базовое описание:

```bash
docker compose -f deploy/docker-compose.yml --env-file deploy/.env exec -T postgres-catalog \
  psql -U gtrade -d gtrade_catalog \
  -c "SELECT COUNT(*) FROM items WHERE game='warframe' AND source='market' AND description IS NOT NULL;"
```

Проверить, что переводы реально легли в `item_translations`:

```bash
docker compose -f deploy/docker-compose.yml --env-file deploy/.env exec -T postgres-catalog \
  psql -U gtrade -d gtrade_catalog \
  -c "SELECT COUNT(*) FROM item_translations WHERE language_code='ru';"
```

Проверить один конкретный предмет:

```bash
docker compose -f deploy/docker-compose.yml --env-file deploy/.env exec -T postgres-catalog \
  psql -U gtrade -d gtrade_catalog \
  -c "SELECT i.slug, i.description, t.language_code, t.description FROM items i LEFT JOIN item_translations t ON t.item_id = i.id AND t.language_code = 'ru' WHERE i.slug = 'secura_dual_cestra';"
```

## Примеры запуска

Базовый импорт Warframe в локальный `catalog-service`:

```bash
go run ./cmd/catalog-importer -source warframe
```

Импорт только первых 50 предметов:

```bash
go run ./cmd/catalog-importer -source warframe -limit 50
```

Импорт с локализацией:

```bash
go run ./cmd/catalog-importer -source warframe -language ru -limit 50
```

Импорт EVE:

```bash
go run ./cmd/catalog-importer -source eve -limit 50
```

Импорт EVE с локализацией:

```bash
go run ./cmd/catalog-importer -source eve -language ru -limit 50
```

Сухой прогон без записи в каталог:

```bash
go run ./cmd/catalog-importer -source warframe -dry-run -limit 20
```

Явный адрес каталога:

```bash
go run ./cmd/catalog-importer -source warframe -catalog-url http://localhost:8084
```

## Поддерживаемые флаги

- `-source` — `warframe|eve|tarkov`
- `-catalog-url` — базовый URL `catalog-service`, по умолчанию `http://localhost:8084`
- `-language` — язык импорта, по умолчанию `en`
- `-limit` — ограничение на количество импортируемых предметов, `0` значит без лимита
- `-dry-run` — fetch и transform без записи в `catalog-service`

## Ограничения

- `warframe` импортирует каталог без цен
- `warframe` импортирует item metadata и локализации, но не market orders
- `eve` импортирует item metadata и локализации, но не сохраняет market prices
- `eve` использует `markets/prices` только как источник списка `type_id`
- локализация появляется в API только после прогона с соответствующим `-language`
- `tarkov` пока не реализован
- утилита зависит от доступности поднятого `catalog-service`
