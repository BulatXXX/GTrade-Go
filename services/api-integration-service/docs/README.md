# api-integration-service docs

Документация по локальному запуску, тестированию и ручной проверке `api-integration-service`.

## Назначение

`api-integration-service` отвечает за runtime-интеграцию с внешними источниками данных по играм.

Сервис:

- получает item data из внешних API
- получает pricing data из внешних API
- нормализует ответы в единый DTO для фронта и соседних сервисов
- не заменяет локальный `catalog-service`

## Поддержанные сценарии

### Warframe

- поиск предметов
- карточка предмета
- top orders и normalized pricing

### EVE

- карточка предмета
- normalized pricing из `markets/prices`
- поиск должен идти через локальный каталог

### Tarkov

- поиск предметов
- карточка предмета
- pricing с учетом `game_mode=regular|pve`

## Swagger / OpenAPI

Текущая OpenAPI-схема лежит в:

- `services/api-integration-service/docs/openapi.yaml`

## Локальный запуск

Подготовить env:

```bash
cp deploy/.env.example deploy/.env
```

Поднять весь локальный контур:

```bash
make up
```

Или поднять сервис вручную:

```bash
cd services/api-integration-service
go run ./cmd/server
```

Базовый адрес:

```text
http://localhost:8083
```

## Тесты

Все тесты сервиса:

```bash
cd services/api-integration-service
GOCACHE=/tmp/gocache-api-integration go test ./...
```

## Ручная проверка API

### 1. Health

```bash
curl http://localhost:8083/health
```

### 2. Warframe search

```bash
curl -sS 'http://localhost:8083/search?game=warframe&q=frost&limit=5&offset=0'
```

### 3. Warframe item

```bash
curl -sS 'http://localhost:8083/items/frost_prime_set?game=warframe'
```

### 4. Warframe prices

```bash
curl -sS 'http://localhost:8083/items/frost_prime_set/prices?game=warframe'
```

### 5. Warframe top price

```bash
curl -sS 'http://localhost:8083/items/frost_prime_set/top-price?game=warframe'
```

### 6. EVE item

```bash
curl -sS 'http://localhost:8083/items/34?game=eve'
```

### 7. EVE prices

```bash
curl -sS 'http://localhost:8083/items/34/prices?game=eve'
```

### 8. Tarkov search regular

```bash
curl -sS 'http://localhost:8083/search?game=tarkov&q=makarov&game_mode=regular&limit=5&offset=0'
```

### 9. Tarkov search pve

```bash
curl -sS 'http://localhost:8083/search?game=tarkov&q=makarov&game_mode=pve&limit=5&offset=0'
```

### 10. Tarkov item

```bash
curl -sS 'http://localhost:8083/items/5448bd6b4bdc2dfc2f8b4569?game=tarkov&game_mode=regular'
```

### 11. Tarkov prices regular

```bash
curl -sS 'http://localhost:8083/items/5448bd6b4bdc2dfc2f8b4569/prices?game=tarkov&game_mode=regular'
```

### 12. Tarkov prices pve

```bash
curl -sS 'http://localhost:8083/items/5448bd6b4bdc2dfc2f8b4569/prices?game=tarkov&game_mode=pve'
```

Если `game_mode` для `tarkov` не передать, сервис использует `regular`.

## Ожидаемые контракты

`GET /items/:id/prices` возвращает полный normalized pricing snapshot.

`GET /items/:id/top-price` возвращает только главное значение цены:

- `item_id`
- `game`
- `game_mode`
- `source`
- `currency`
- `value`
- `fetched_at`

## Ограничения

- `eve` search сейчас не реализован как runtime endpoint поверх ESI
- `top-price` для игр с агрегированной аналитикой является сокращением от полного pricing snapshot
- нет persistence layer для historical price snapshots
- нет internal auth для будущих sync endpoint'ов
