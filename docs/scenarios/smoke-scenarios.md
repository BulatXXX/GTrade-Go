# Smoke Scenarios

Пошаговые smoke-сценарии для ручной проверки локального стека GTrade.

## Подготовка

Поднять весь стек:

```bash
docker compose -f deploy/docker-compose.yml --env-file deploy/.env up -d
```

Проверить, что контейнеры живы:

```bash
docker compose -f deploy/docker-compose.yml --env-file deploy/.env ps
```

Проверить `health` всех HTTP-сервисов:

```bash
for port in 8080 8081 8082 8083 8084 8085; do
  echo "PORT=$port"
  curl -fsS "http://localhost:${port}/health"
  echo
done
```

Подготовить переменные:

```bash
export GATEWAY_URL=http://localhost:8080
export AUTH_URL=http://localhost:8081
export USER_ASSET_URL=http://localhost:8082
export INTEGRATION_URL=http://localhost:8083
export CATALOG_URL=http://localhost:8084
export NOTIFICATION_URL=http://localhost:8085
export INTERNAL_API_TOKEN=change-me-internal-token
```

## Сценарий 1. Auth через gateway

Регистрация:

```bash
export SMOKE_EMAIL="smoke.$(date +%s)@example.com"

curl -sS -X POST "$GATEWAY_URL/api/auth/register" \
  -H 'Content-Type: application/json' \
  -d "{
    \"email\": \"$SMOKE_EMAIL\",
    \"password\": \"secret123\"
  }" | tee /tmp/gtrade-register.json
```

Сохранить токены:

```bash
export JWT_TOKEN="$(jq -r '.access_token' /tmp/gtrade-register.json)"
export REFRESH_TOKEN="$(jq -r '.refresh_token' /tmp/gtrade-register.json)"
```

Refresh:

```bash
curl -sS -X POST "$GATEWAY_URL/api/auth/refresh" \
  -H 'Content-Type: application/json' \
  -d "{
    \"refresh_token\": \"$REFRESH_TOKEN\"
  }"
```

Логин отдельным запросом:

```bash
curl -sS -X POST "$GATEWAY_URL/api/auth/login" \
  -H 'Content-Type: application/json' \
  -d "{
    \"email\": \"$SMOKE_EMAIL\",
    \"password\": \"secret123\"
  }"
```

Ожидаемый результат:

- `register` возвращает `access_token` и `refresh_token`
- `refresh` возвращает новую пару токенов
- `login` возвращает пару токенов

## Сценарий 2. User profile и preferences через gateway

Создать профиль:

```bash
export SMOKE_USER_ID=$(( (RANDOM % 900000) + 100000 ))

curl -sS -X POST "$GATEWAY_URL/api/users/users" \
  -H "Authorization: Bearer $JWT_TOKEN" \
  -H 'Content-Type: application/json' \
  -d "{
    \"user_id\": $SMOKE_USER_ID,
    \"display_name\": \"Smoke User\"
  }"
```

Получить профиль:

```bash
curl -sS "$GATEWAY_URL/api/users/users/$SMOKE_USER_ID" \
  -H "Authorization: Bearer $JWT_TOKEN"
```

Обновить профиль:

```bash
curl -sS -X PUT "$GATEWAY_URL/api/users/users/$SMOKE_USER_ID" \
  -H "Authorization: Bearer $JWT_TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{
    "display_name": "Smoke User Updated",
    "avatar_url": "https://example.com/avatar.png",
    "bio": "Trader and collector"
  }'
```

Обновить preferences:

```bash
curl -sS -X PUT "$GATEWAY_URL/api/users/preferences" \
  -H "Authorization: Bearer $JWT_TOKEN" \
  -H 'Content-Type: application/json' \
  -d "{
    \"user_id\": $SMOKE_USER_ID,
    \"currency\": \"usd\",
    \"notifications_enabled\": true
  }"
```

Получить preferences:

```bash
curl -sS "$GATEWAY_URL/api/users/preferences?user_id=$SMOKE_USER_ID" \
  -H "Authorization: Bearer $JWT_TOKEN"
```

Ожидаемый результат:

- профиль создается и читается
- `display_name` обновляется
- preferences возвращают `currency=usd`

## Сценарий 3. Catalog через gateway

Поиск:

```bash
curl -sS "$GATEWAY_URL/api/items/search?q=frost&game=warframe&language=ru&limit=5&offset=0" \
  | tee /tmp/gtrade-catalog-search.json
```

Сохранить локальный `catalog item id`:

```bash
export CATALOG_ITEM_ID="$(jq -r '.items[0].id' /tmp/gtrade-catalog-search.json)"
echo "$CATALOG_ITEM_ID"
```

Получить предмет по локальному id:

```bash
curl -sS "$GATEWAY_URL/api/items/$CATALOG_ITEM_ID?language=ru"
```

Ожидаемый результат:

- поиск возвращает `items`
- `CATALOG_ITEM_ID` не пустой
- `GET /api/items/$CATALOG_ITEM_ID` возвращает локальную карточку

## Сценарий 4. Runtime market data через gateway

Warframe search:

```bash
curl -sS "$GATEWAY_URL/api/market/search?game=warframe&q=frost&limit=3&offset=0" \
  | tee /tmp/gtrade-market-search.json
```

Сохранить provider id:

```bash
export MARKET_ITEM_ID="$(jq -r '.items[0].id' /tmp/gtrade-market-search.json)"
echo "$MARKET_ITEM_ID"
```

Получить runtime item:

```bash
curl -sS "$GATEWAY_URL/api/market/items/$MARKET_ITEM_ID?game=warframe"
```

Получить полный pricing snapshot:

```bash
curl -sS "$GATEWAY_URL/api/market/items/$MARKET_ITEM_ID/prices?game=warframe"
```

Получить top price:

```bash
curl -sS "$GATEWAY_URL/api/market/items/$MARKET_ITEM_ID/top-price?game=warframe"
```

Tarkov пример:

```bash
curl -sS "$GATEWAY_URL/api/market/search?game=tarkov&q=makarov&game_mode=regular&limit=3&offset=0"
```

Ожидаемый результат:

- market search возвращает `items`
- `prices` возвращает объект `price`
- `top-price` возвращает поле `value`

## Сценарий 5. Watchlist через gateway

Добавить локальный catalog item в watchlist:

```bash
curl -sS -X POST "$GATEWAY_URL/api/users/watchlist" \
  -H "Authorization: Bearer $JWT_TOKEN" \
  -H 'Content-Type: application/json' \
  -d "{
    \"user_id\": $SMOKE_USER_ID,
    \"item_id\": \"$CATALOG_ITEM_ID\"
  }" | tee /tmp/gtrade-watch-create.json
```

Получить watchlist:

```bash
curl -sS "$GATEWAY_URL/api/users/watchlist?user_id=$SMOKE_USER_ID" \
  -H "Authorization: Bearer $JWT_TOKEN"
```

Получить recent:

```bash
curl -sS "$GATEWAY_URL/api/users/recent?user_id=$SMOKE_USER_ID" \
  -H "Authorization: Bearer $JWT_TOKEN"
```

Удалить watchlist item:

```bash
export WATCHLIST_ID="$(jq -r '.id' /tmp/gtrade-watch-create.json)"

curl -i -X DELETE "$GATEWAY_URL/api/users/watchlist/$WATCHLIST_ID?user_id=$SMOKE_USER_ID" \
  -H "Authorization: Bearer $JWT_TOKEN"
```

Ожидаемый результат:

- item добавляется
- `watchlist` и `recent` возвращают хотя бы один элемент
- delete возвращает `204 No Content`

## Сценарий 6. Internal sync напрямую в api-integration-service

Sync одного item:

```bash
curl -sS -X POST "$INTEGRATION_URL/internal/sync/item" \
  -H 'Content-Type: application/json' \
  -H "X-Internal-Token: $INTERNAL_API_TOKEN" \
  -d '{
    "game": "warframe",
    "id": "frost_prime_set"
  }'
```

Sync страницы поиска:

```bash
curl -sS -X POST "$INTEGRATION_URL/internal/sync/search" \
  -H 'Content-Type: application/json' \
  -H "X-Internal-Token: $INTERNAL_API_TOKEN" \
  -d '{
    "game": "warframe",
    "q": "prime",
    "limit": 5,
    "offset": 0
  }'
```

Negative test без токена:

```bash
curl -i -X POST "$INTEGRATION_URL/internal/sync/item" \
  -H 'Content-Type: application/json' \
  -d '{
    "game": "warframe",
    "id": "frost_prime_set"
  }'
```

Ожидаемый результат:

- с `X-Internal-Token` sync проходит
- без токена приходит `401`

## Сценарий 7. Быстрый автоматический smoke одной командой

```bash
email="smoke.$(date +%s)@example.com"
user_id=$(( (RANDOM % 900000) + 100000 ))

register=$(curl -fsS -X POST http://localhost:8080/api/auth/register -H 'Content-Type: application/json' -d "{\"email\":\"$email\",\"password\":\"secret123\"}")
access=$(printf '%s' "$register" | jq -r '.access_token')
refresh=$(printf '%s' "$register" | jq -r '.refresh_token')

create_user=$(curl -fsS -X POST http://localhost:8080/api/users/users -H "Authorization: Bearer $access" -H 'Content-Type: application/json' -d "{\"user_id\":$user_id,\"display_name\":\"Smoke User\"}")
get_user=$(curl -fsS http://localhost:8080/api/users/users/$user_id -H "Authorization: Bearer $access")
curl -fsS -X PUT http://localhost:8080/api/users/preferences -H "Authorization: Bearer $access" -H 'Content-Type: application/json' -d "{\"user_id\":$user_id,\"currency\":\"usd\",\"notifications_enabled\":true}" >/dev/null
prefs_get=$(curl -fsS "http://localhost:8080/api/users/preferences?user_id=$user_id" -H "Authorization: Bearer $access")
refresh_resp=$(curl -fsS -X POST http://localhost:8080/api/auth/refresh -H 'Content-Type: application/json' -d "{\"refresh_token\":\"$refresh\"}")
search_catalog=$(curl -fsS "http://localhost:8080/api/items/search?q=frost&game=warframe&limit=5&offset=0")
catalog_item_id=$(printf '%s' "$search_catalog" | jq -r '.items[0].id // empty')
search_market=$(curl -fsS "http://localhost:8080/api/market/search?game=warframe&q=frost&limit=3&offset=0")
market_slug=$(printf '%s' "$search_market" | jq -r '.items[0].id // empty')
market_prices=$(curl -fsS "http://localhost:8080/api/market/items/${market_slug}/prices?game=warframe")
sync_item=$(curl -fsS -X POST http://localhost:8083/internal/sync/item -H 'Content-Type: application/json' -H 'X-Internal-Token: change-me-internal-token' -d '{"game":"warframe","id":"frost_prime_set"}')

printf 'REGISTER_OK=%s\n' "$(printf '%s' "$register" | jq -r 'has("access_token") and has("refresh_token")')"
printf 'USER_CREATE_OK=%s\n' "$(printf '%s' "$create_user" | jq -r ".user_id == $user_id")"
printf 'GET_USER_OK=%s\n' "$(printf '%s' "$get_user" | jq -r ".user.user_id == $user_id")"
printf 'PREFS_CURRENCY=%s\n' "$(printf '%s' "$prefs_get" | jq -r '.currency // empty')"
printf 'REFRESH_OK=%s\n' "$(printf '%s' "$refresh_resp" | jq -r 'has("access_token") and has("refresh_token")')"
printf 'CATALOG_SEARCH_COUNT=%s\n' "$(printf '%s' "$search_catalog" | jq -r '.items | length')"
printf 'CATALOG_ITEM_ID=%s\n' "$catalog_item_id"
printf 'MARKET_SEARCH_COUNT=%s\n' "$(printf '%s' "$search_market" | jq -r '.items | length')"
printf 'MARKET_PRICE_OK=%s\n' "$(printf '%s' "$market_prices" | jq -r 'has("price")')"
printf 'SYNC_ITEM_OK=%s\n' "$(printf '%s' "$sync_item" | jq -r 'has("item")')"
```

## Завершение

Остановить стек:

```bash
docker compose -f deploy/docker-compose.yml --env-file deploy/.env down
```
