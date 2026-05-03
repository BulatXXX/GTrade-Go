#!/bin/zsh

# GTrade pricing + user demo script
# Сценарий для записи ролика с акцентом на:
# - готового пользователя
# - пользовательский watchlist
# - единый каталог из разных источников
# - получение актуальных цен и top-price
#
# Предполагается, что пользователь уже существует.
# Нужна только переменная DEMO_USER_ID.
#
# Если user-asset-service вызывается напрямую, JWT не нужен.
# Это удобно для короткого сильного ролика без auth-шумов.

export GATEWAY_URL=http://localhost:8080
export USER_ASSET_URL=http://localhost:8082
export INTEGRATION_URL=http://localhost:8083
export CATALOG_URL=http://localhost:8084

# Пример:
# export DEMO_USER_ID=100001

# 1. Показать, что backend-контур поднят
docker compose -f deploy/docker-compose.yml --env-file deploy/.env ps

for port in 8080 8082 8083 8084; do
  echo "===== PORT $port ====="
  curl -sS "http://localhost:${port}/health" | jq
done

# 2. Показать масштаб каталога
docker compose -f deploy/docker-compose.yml --env-file deploy/.env exec -T postgres-catalog \
  psql -U gtrade -d gtrade_catalog \
  -c "SELECT game, source, COUNT(*) AS items_count FROM items GROUP BY game, source ORDER BY game, source;"

docker compose -f deploy/docker-compose.yml --env-file deploy/.env exec -T postgres-catalog \
  psql -U gtrade -d gtrade_catalog \
  -c "SELECT COUNT(*) AS total_items FROM items;"

# 3. Показать единый каталог по разным играм
curl -sS "$CATALOG_URL/items?game=warframe&limit=3&offset=0&language=ru" \
  | jq '{warframe_items: [.items[] | {id, game, source, name, slug, image_url}]}'

curl -sS "$CATALOG_URL/items?game=eve&limit=3&offset=0&language=ru" \
  | jq '{eve_items: [.items[] | {id, game, source, name, slug, image_url}]}'

curl -sS "$CATALOG_URL/items?game=tarkov&limit=3&offset=0&language=ru" \
  | jq '{tarkov_items: [.items[] | {id, game, source, name, slug, image_url}]}'

# 4. Найти предметы в локальном каталоге и сохранить их в watchlist
curl -sS "$GATEWAY_URL/api/items/search?q=frost&game=warframe&language=ru&limit=3&offset=0" \
  | tee /tmp/pricing-demo-warframe-search.json \
  | jq '{search: "warframe frost", items: [.items[] | {id, game, source, name, slug}]}'

curl -sS "$GATEWAY_URL/api/items/search?q=plagioclase&game=eve&language=ru&limit=3&offset=0" \
  | tee /tmp/pricing-demo-eve-search.json \
  | jq '{search: "eve plagioclase", items: [.items[] | {id, game, source, name, slug}]}'

curl -sS "$GATEWAY_URL/api/items/search?q=makarov&game=tarkov&language=ru&limit=3&offset=0" \
  | tee /tmp/pricing-demo-tarkov-search.json \
  | jq '{search: "tarkov makarov", items: [.items[] | {id, game, source, name, slug}]}'

export WF_ITEM_ID="$(jq -r '.items[0].id' /tmp/pricing-demo-warframe-search.json)"
export EVE_ITEM_ID="$(jq -r '.items[0].id' /tmp/pricing-demo-eve-search.json)"
export TARKOV_ITEM_ID="$(jq -r '.items[0].id' /tmp/pricing-demo-tarkov-search.json)"

echo "$WF_ITEM_ID"
echo "$EVE_ITEM_ID"
echo "$TARKOV_ITEM_ID"

curl -sS -X POST "$USER_ASSET_URL/watchlist" \
  -H 'Content-Type: application/json' \
  -d "{
    \"user_id\": $DEMO_USER_ID,
    \"item_id\": \"$WF_ITEM_ID\"
  }" | jq

curl -sS -X POST "$USER_ASSET_URL/watchlist" \
  -H 'Content-Type: application/json' \
  -d "{
    \"user_id\": $DEMO_USER_ID,
    \"item_id\": \"$EVE_ITEM_ID\"
  }" | jq

curl -sS -X POST "$USER_ASSET_URL/watchlist" \
  -H 'Content-Type: application/json' \
  -d "{
    \"user_id\": $DEMO_USER_ID,
    \"item_id\": \"$TARKOV_ITEM_ID\"
  }" | jq

# 5. Показать организацию пользовательских данных
curl -sS "$USER_ASSET_URL/watchlist?user_id=$DEMO_USER_ID" \
  | jq '{watchlist: [.items[] | {watch_id: .id, item_id, game: .item.game, source: .item.source, name: .item.name, slug: .item.slug, image_url: .item.image_url}]}'

curl -sS "$USER_ASSET_URL/users/$DEMO_USER_ID" \
  | jq '{user: .user, watchlist_count: (.watchlist | length), watchlist: [.watchlist[] | {game: .item.game, source: .item.source, name: .item.name}]}'

# 6. Найти предметы уже на внешних площадках
curl -sS "$GATEWAY_URL/api/market/search?game=warframe&q=frost&limit=3&offset=0" \
  | tee /tmp/pricing-demo-wf-market.json \
  | jq '{provider_search: "warframe", items: [.items[] | {id, game, source, name, currency}]}'

curl -sS "$GATEWAY_URL/api/market/search?game=eve&q=plex&limit=3&offset=0" \
  | tee /tmp/pricing-demo-eve-market.json \
  | jq '{provider_search: "eve", items: [.items[] | {id, game, source, name, currency}]}'

curl -sS "$GATEWAY_URL/api/market/search?game=tarkov&q=makarov&game_mode=regular&limit=3&offset=0" \
  | tee /tmp/pricing-demo-tarkov-market.json \
  | jq '{provider_search: "tarkov", items: [.items[] | {id, game, source, name, currency}]}'

export WF_MARKET_ID="$(jq -r '.items[0].id' /tmp/pricing-demo-wf-market.json)"
export EVE_MARKET_ID="$(jq -r '.items[0].id' /tmp/pricing-demo-eve-market.json)"
export TARKOV_MARKET_ID="$(jq -r '.items[0].id' /tmp/pricing-demo-tarkov-market.json)"

echo "$WF_MARKET_ID"
echo "$EVE_MARKET_ID"
echo "$TARKOV_MARKET_ID"

# 7. Показать полные ценовые snapshots
curl -sS "$GATEWAY_URL/api/market/items/$WF_MARKET_ID/prices?game=warframe" \
  | jq '{warframe_price_snapshot: {item_id: .price.item_id, game: .price.game, source: .price.source, currency: .price.currency, market_kind: .price.market_kind, pricing: .price.pricing, analytics: .price.analytics}}'

curl -sS "$GATEWAY_URL/api/market/items/$EVE_MARKET_ID/prices?game=eve" \
  | jq '{eve_price_snapshot: {item_id: .price.item_id, game: .price.game, source: .price.source, currency: .price.currency, market_kind: .price.market_kind, pricing: .price.pricing, analytics: .price.analytics}}'

curl -sS "$GATEWAY_URL/api/market/items/$TARKOV_MARKET_ID/prices?game=tarkov&game_mode=regular" \
  | jq '{tarkov_price_snapshot: {item_id: .price.item_id, game: .price.game, game_mode: .price.game_mode, source: .price.source, currency: .price.currency, market_kind: .price.market_kind, pricing: .price.pricing, analytics: .price.analytics}}'

# 8. Показать короткие top-price ответы
curl -sS "$GATEWAY_URL/api/market/items/$WF_MARKET_ID/top-price?game=warframe" | jq

curl -sS "$GATEWAY_URL/api/market/items/$EVE_MARKET_ID/top-price?game=eve" | jq

curl -sS "$GATEWAY_URL/api/market/items/$TARKOV_MARKET_ID/top-price?game=tarkov&game_mode=regular" | jq

# 9. Показать recent как повторный доступ к пользовательским данным
curl -sS "$USER_ASSET_URL/recent?user_id=$DEMO_USER_ID" \
  | jq '{recent_items: [.items[] | {watch_id: .id, game: .item.game, source: .item.source, name: .item.name}]}'
