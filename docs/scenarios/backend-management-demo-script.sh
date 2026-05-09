#!/bin/zsh

# GTrade backend demo script
# Сценарий записи ролика без фронта:
# - запуск backend-контура
# - полный импорт данных из разных площадок
# - формирование пользовательского watchlist
# - получение runtime-цен с разных источников
# - демонстрация organization/management данных
#
# Использование:
# 1. Подними стек или копируй команды по шагам вручную.
# 2. Для красивого JSON можно выполнить: alias pp='jq'

export GATEWAY_URL=http://localhost:8080
export AUTH_URL=http://localhost:8081
export USER_ASSET_URL=http://localhost:8082
export INTEGRATION_URL=http://localhost:8083
export CATALOG_URL=http://localhost:8084
export NOTIFICATION_URL=http://localhost:8085
export INTERNAL_API_TOKEN=change-me-internal-token

# 1. Поднять систему
docker compose -f deploy/docker-compose.yml --env-file deploy/.env up -d

docker compose -f deploy/docker-compose.yml --env-file deploy/.env ps

for port in 8080 8081 8082 8083 8084 8085; do
  echo "===== PORT $port ====="
  curl -sS "http://localhost:${port}/health" | jq
done

# 2. Полный импорт каталога из разных источников
GOCACHE=/tmp/gocache-importer go run ./tools/catalog-importer/cmd/catalog-importer -source warframe -language ru -catalog-url http://localhost:8084

GOCACHE=/tmp/gocache-importer go run ./tools/catalog-importer/cmd/catalog-importer -source eve -language ru -catalog-url http://localhost:8084

GOCACHE=/tmp/gocache-importer go run ./tools/catalog-importer/cmd/catalog-importer -source tarkov -language ru -catalog-url http://localhost:8084

# 3. Показать, что каталог содержит данные из нескольких игр и источников
curl -sS "$CATALOG_URL/items?game=warframe&limit=5&offset=0&language=ru" | jq '{items: [.items[] | {id, game, source, name, slug}]}'

curl -sS "$CATALOG_URL/items?game=eve&limit=5&offset=0&language=ru" | jq '{items: [.items[] | {id, game, source, name, slug}]}'

curl -sS "$CATALOG_URL/items?game=tarkov&limit=5&offset=0&language=ru" | jq '{items: [.items[] | {id, game, source, name, slug}]}'

# 4. Зарегистрировать пользователя и создать профиль
export DEMO_EMAIL="demo.$(date +%s)@example.com"
export DEMO_PASSWORD="secret123"

curl -sS -X POST "$GATEWAY_URL/api/auth/register" \
  -H 'Content-Type: application/json' \
  -d "{
    \"email\": \"$DEMO_EMAIL\",
    \"password\": \"$DEMO_PASSWORD\"
  }" | tee /tmp/gtrade-register.json | jq

export JWT_TOKEN="$(jq -r '.access_token' /tmp/gtrade-register.json)"
export REFRESH_TOKEN="$(jq -r '.refresh_token' /tmp/gtrade-register.json)"
export DEMO_USER_ID=$(( (RANDOM % 900000) + 100000 ))

curl -sS -X POST "$GATEWAY_URL/api/users/users" \
  -H "Authorization: Bearer $JWT_TOKEN" \
  -H 'Content-Type: application/json' \
  -d "{
    \"user_id\": $DEMO_USER_ID,
    \"display_name\": \"Bulat Demo User\"
  }" | jq

# 5. Найти предметы из разных игр в локальном каталоге
curl -sS "$GATEWAY_URL/api/items/search?q=frost&game=warframe&language=ru&limit=3&offset=0" \
  | tee /tmp/warframe-search.json | jq '{items: [.items[] | {id, game, source, name, slug}]}'

export WF_ITEM_ID="$(jq -r '.items[0].id' /tmp/warframe-search.json)"
echo "$WF_ITEM_ID"

curl -sS "$GATEWAY_URL/api/items/search?q=plagioclase&game=eve&language=ru&limit=3&offset=0" \
  | tee /tmp/eve-search.json | jq '{items: [.items[] | {id, game, source, name, slug}]}'

export EVE_ITEM_ID="$(jq -r '.items[0].id' /tmp/eve-search.json)"
echo "$EVE_ITEM_ID"

curl -sS "$GATEWAY_URL/api/items/search?q=makarov&game=tarkov&language=ru&limit=3&offset=0" \
  | tee /tmp/tarkov-search.json | jq '{items: [.items[] | {id, game, source, name, slug}]}'

export TARKOV_ITEM_ID="$(jq -r '.items[0].id' /tmp/tarkov-search.json)"
echo "$TARKOV_ITEM_ID"

# 6. Сформировать персональный список отслеживаемых предметов
curl -sS -X POST "$GATEWAY_URL/api/users/watchlist" \
  -H "Authorization: Bearer $JWT_TOKEN" \
  -H 'Content-Type: application/json' \
  -d "{
    \"user_id\": $DEMO_USER_ID,
    \"item_id\": \"$WF_ITEM_ID\"
  }" | jq

curl -sS -X POST "$GATEWAY_URL/api/users/watchlist" \
  -H "Authorization: Bearer $JWT_TOKEN" \
  -H 'Content-Type: application/json' \
  -d "{
    \"user_id\": $DEMO_USER_ID,
    \"item_id\": \"$EVE_ITEM_ID\"
  }" | jq

curl -sS -X POST "$GATEWAY_URL/api/users/watchlist" \
  -H "Authorization: Bearer $JWT_TOKEN" \
  -H 'Content-Type: application/json' \
  -d "{
    \"user_id\": $DEMO_USER_ID,
    \"item_id\": \"$TARKOV_ITEM_ID\"
  }" | jq

curl -sS "$GATEWAY_URL/api/users/watchlist?user_id=$DEMO_USER_ID" \
  -H "Authorization: Bearer $JWT_TOKEN" \
  | jq '{items: [.items[] | {watch_id: .id, user_id, item_id, game: .item.game, source: .item.source, name: .item.name, slug: .item.slug}]}'

# 7. Показать профиль пользователя вместе с watchlist
curl -sS "$GATEWAY_URL/api/users/users/$DEMO_USER_ID" \
  -H "Authorization: Bearer $JWT_TOKEN" \
  | jq

# 8. Получить runtime-цены по данным из разных площадок
curl -sS "$GATEWAY_URL/api/market/search?game=warframe&q=frost&limit=3&offset=0" \
  | tee /tmp/wf-market.json | jq '{items: [.items[] | {id, game, source, name, currency}]}'

export WF_MARKET_ID="$(jq -r '.items[0].id' /tmp/wf-market.json)"
echo "$WF_MARKET_ID"

curl -sS "$GATEWAY_URL/api/market/items/$WF_MARKET_ID/prices?game=warframe" | jq

curl -sS "$GATEWAY_URL/api/market/items/$WF_MARKET_ID/top-price?game=warframe" | jq

curl -sS "$GATEWAY_URL/api/market/search?game=eve&q=plex&limit=3&offset=0" \
  | tee /tmp/eve-market.json | jq '{items: [.items[] | {id, game, source, name, currency}]}'

export EVE_MARKET_ID="$(jq -r '.items[0].id' /tmp/eve-market.json)"
echo "$EVE_MARKET_ID"

curl -sS "$GATEWAY_URL/api/market/items/$EVE_MARKET_ID/prices?game=eve" | jq

curl -sS "$GATEWAY_URL/api/market/items/$EVE_MARKET_ID/top-price?game=eve" | jq

curl -sS "$GATEWAY_URL/api/market/search?game=tarkov&q=makarov&game_mode=regular&limit=3&offset=0" \
  | tee /tmp/tarkov-market.json | jq '{items: [.items[] | {id, game, source, name, currency}]}'

export TARKOV_MARKET_ID="$(jq -r '.items[0].id' /tmp/tarkov-market.json)"
echo "$TARKOV_MARKET_ID"

curl -sS "$GATEWAY_URL/api/market/items/$TARKOV_MARKET_ID/prices?game=tarkov&game_mode=regular" | jq

curl -sS "$GATEWAY_URL/api/market/items/$TARKOV_MARKET_ID/top-price?game=tarkov&game_mode=regular" | jq

# 9. Показать recent-данные пользователя
curl -sS "$GATEWAY_URL/api/users/recent?user_id=$DEMO_USER_ID" \
  -H "Authorization: Bearer $JWT_TOKEN" \
  | jq '{items: [.items[] | {watch_id: .id, game: .item.game, source: .item.source, name: .item.name}]}'

# 10. Показать сервисный пользовательский сценарий с email
curl -sS -X POST "$AUTH_URL/password/reset/request" \
  -H 'Content-Type: application/json' \
  -d "{
    \"email\": \"$DEMO_EMAIL\"
  }" | jq

# 11. Финальная проверка тестами
make test-all
