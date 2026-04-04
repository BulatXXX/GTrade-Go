# catalog-importer

CLI-утилита наполнения `catalog-service` данными из внешних источников.

## Текущий статус

Сейчас в утилите реально поддержан:

- `warframe`

Для `eve` и `tarkov` каркас источников создан, но fetch-логика еще не реализована.

## Что делает утилита

- получает список предметов из внешнего источника
- преобразует их в канонический формат каталога
- отправляет их в `catalog-service` через `POST /items/upsert`

Утилита рассчитана на idempotent импорт: повторный запуск должен обновлять существующие записи по `game + source + external_id`.

## Использование

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
- `-dry-run` — не записывать данные в `catalog-service`

## Ограничения

- `warframe` импортирует каталог предметов без цен
- `eve` пока не реализован
- `tarkov` пока не реализован
- утилита зависит от доступности поднятого `catalog-service`
