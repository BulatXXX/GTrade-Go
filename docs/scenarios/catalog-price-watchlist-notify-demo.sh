#!/bin/zsh

# GTrade unified demo script
# Сценарий:
# 1. Поиск предметов в едином локальном каталоге
# 2. Получение актуальных цен по разным играм
# 3. Добавление предметов в пользовательский watchlist
# 4. Демонстрация идеи уведомления об изменении цены
#
# Важно:
# - пользователь создается как технический демо-пользователь без auth-flow
# - уведомление показывается как интеграционный сценарий поверх уже готовых данных

export GATEWAY_URL=http://localhost:8080
export USER_ASSET_URL=http://localhost:8082
export NOTIFICATION_URL=http://localhost:8085
export CATALOG_URL=http://localhost:8084
export DEMO_USER_ID=100001

# 0. Базовая проверка контура
docker compose -f deploy/docker-compose.yml --env-file deploy/.env ps

for port in 8080 8082 8084 8085; do
  echo "===== PORT $port ====="
  curl -sS "http://localhost:${port}/health" | jq
done

# 1. Показать масштаб данных каталога
docker compose -f deploy/docker-compose.yml --env-file deploy/.env exec -T postgres-catalog \
  psql -U gtrade -d gtrade_catalog \
  -c "SELECT game, source, COUNT(*) AS items_count FROM items GROUP BY game, source ORDER BY game, source;"

# 2. Поиск предметов в локальном каталоге
curl -sS "$GATEWAY_URL/api/items/search?q=frost&game=warframe&language=ru&limit=1&offset=0" \
  | tee /tmp/demo-wf-catalog.json \
  | jq '{catalog_search: "warframe frost", items: [.items[] | {id, game, source, name, slug}]}'

curl -sS "$GATEWAY_URL/api/items/search?q=plagioclase&game=eve&language=ru&limit=1&offset=0" \
  | tee /tmp/demo-eve-catalog.json \
  | jq '{catalog_search: "eve plagioclase", items: [.items[] | {id, game, source, name, slug}]}'

curl -sS "$GATEWAY_URL/api/items/search?q=makarov&game=tarkov&language=ru&limit=1&offset=0" \
  | tee /tmp/demo-tarkov-catalog.json \
  | jq '{catalog_search: "tarkov makarov", items: [.items[] | {id, game, source, name, slug}]}'

export WF_ITEM_ID="$(jq -r '.items[0].id' /tmp/demo-wf-catalog.json)"
export EVE_ITEM_ID="$(jq -r '.items[0].id' /tmp/demo-eve-catalog.json)"
export TARKOV_ITEM_ID="$(jq -r '.items[0].id' /tmp/demo-tarkov-catalog.json)"

echo "$WF_ITEM_ID"
echo "$EVE_ITEM_ID"
echo "$TARKOV_ITEM_ID"

# 3. Runtime prices: Warframe
curl -sS "$GATEWAY_URL/api/market/search?game=warframe&q=frost&limit=3&offset=0" \
  | tee /tmp/demo-wf-market.json \
  | jq '{provider_search: "warframe", items: [.items[] | {id, game, source, name, currency}]}'

export WF_MARKET_ID="$(jq -r '.items[0].id' /tmp/demo-wf-market.json)"
echo "$WF_MARKET_ID"

curl -sS "$GATEWAY_URL/api/market/items/$WF_MARKET_ID/prices?game=warframe" \
  | jq '{warframe_price: {item_id: .price.item_id, source: .price.source, currency: .price.currency, pricing: .price.pricing, analytics: .price.analytics}}'

curl -sS "$GATEWAY_URL/api/market/items/$WF_MARKET_ID/top-price?game=warframe" | jq

# 4. Runtime prices: EVE по type_id
export EVE_TYPE_ID=34
echo "$EVE_TYPE_ID"

curl -sS "$GATEWAY_URL/api/market/items/$EVE_TYPE_ID?game=eve" \
  | jq '{eve_item: .item}'

curl -sS "$GATEWAY_URL/api/market/items/$EVE_TYPE_ID/prices?game=eve" \
  | jq '{eve_price: {item_id: .price.item_id, source: .price.source, currency: .price.currency, pricing: .price.pricing}}'

curl -sS "$GATEWAY_URL/api/market/items/$EVE_TYPE_ID/top-price?game=eve" | jq

# 5. Runtime prices: Tarkov
curl -sS "$GATEWAY_URL/api/market/search?game=tarkov&q=makarov&game_mode=regular&limit=3&offset=0" \
  | tee /tmp/demo-tarkov-market.json \
  | jq '{provider_search: "tarkov", items: [.items[] | {id, game, source, name, currency}]}'

export TARKOV_MARKET_ID="$(jq -r '.items[0].id' /tmp/demo-tarkov-market.json)"
echo "$TARKOV_MARKET_ID"

curl -sS "$GATEWAY_URL/api/market/items/$TARKOV_MARKET_ID?game=tarkov&game_mode=regular" \
  | jq '{tarkov_item: .item}'

curl -sS "$GATEWAY_URL/api/market/items/$TARKOV_MARKET_ID/prices?game=tarkov&game_mode=regular" \
  | jq '{tarkov_price: {item_id: .price.item_id, source: .price.source, currency: .price.currency, pricing: .price.pricing, analytics: .price.analytics}}'

curl -sS "$GATEWAY_URL/api/market/items/$TARKOV_MARKET_ID/top-price?game=tarkov&game_mode=regular" | jq

# 6. Создать демо-пользователя
curl -sS -X POST "$USER_ASSET_URL/users" \
  -H 'Content-Type: application/json' \
  -d "{
    \"user_id\": $DEMO_USER_ID,
    \"display_name\": \"Demo User\"
  }" | jq

# 7. Добавить найденные в каталоге предметы в watchlist
curl -sS -X POST "$USER_ASSET_URL/watchlist" \
  -H 'Content-Type: application/json' \
  -d "{\"user_id\": $DEMO_USER_ID, \"item_id\": \"$WF_ITEM_ID\"}" | jq

curl -sS -X POST "$USER_ASSET_URL/watchlist" \
  -H 'Content-Type: application/json' \
  -d "{\"user_id\": $DEMO_USER_ID, \"item_id\": \"$EVE_ITEM_ID\"}" | jq

curl -sS -X POST "$USER_ASSET_URL/watchlist" \
  -H 'Content-Type: application/json' \
  -d "{\"user_id\": $DEMO_USER_ID, \"item_id\": \"$TARKOV_ITEM_ID\"}" | jq

curl -sS "$USER_ASSET_URL/watchlist?user_id=$DEMO_USER_ID" \
  | jq '{watchlist: [(.items // [])[] | {watch_id: .id, game: .item.game, source: .item.source, name: .item.name, slug: .item.slug}]}'

curl -sS "$USER_ASSET_URL/users/$DEMO_USER_ID" \
  | jq '{user: .user, watchlist_count: ((.watchlist // []) | length), watchlist: [(.watchlist // [])[] | {game: .item.game, source: .item.source, name: .item.name}]}'

# 8. Демонстрация идеи уведомления о смене цены
curl -sS -X POST "$NOTIFICATION_URL/send-email" \
  -H 'Content-Type: application/json' \
  -d '{
    "to": "demo@example.com",
    "subject": "GTrade price alert",
    "text": "Цена по отслеживаемым предметам обновлена: Frost Prime Set, Tritanium, Makarov PM."
  }' | jq
