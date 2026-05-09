#!/bin/zsh

# GTrade backend "wow" demo script
# Для записи ролика с акцентом на:
# - массовый импорт данных из разных площадок
# - единый нормализованный каталог
# - пользовательский watchlist как организация данных
# - runtime pricing из разных внешних источников
#
# Идея:
# - подготовку пользователя и токена делаем ДО записи
# - в кадре показываем только сильные части системы
#
# Обязательный pre-setup вне записи:
# export DEMO_USER_ID=<готовый user_id>
# export JWT_TOKEN=<готовый access token>
#
# Если нужен один раз подготовительный сценарий, смотри:
# docs/scenarios/backend-management-demo-script.sh

export GATEWAY_URL=http://localhost:8080
export AUTH_URL=http://localhost:8081
export USER_ASSET_URL=http://localhost:8082
export INTEGRATION_URL=http://localhost:8083
export CATALOG_URL=http://localhost:8084
export NOTIFICATION_URL=http://localhost:8085
export INTERNAL_API_TOKEN=change-me-internal-token

# Пример:
# export DEMO_USER_ID=100001
# export JWT_TOKEN=eyJ...

# 1. Поднять backend-контур
docker compose -f deploy/docker-compose.yml --env-file deploy/.env up -d

docker compose -f deploy/docker-compose.yml --env-file deploy/.env ps

for port in 8080 8081 8082 8083 8084 8085; do
  echo "===== PORT $port ====="
  curl -sS "http://localhost:${port}/health" | jq
done

# 2. Массовый импорт из трех источников
GOCACHE=/tmp/gocache-importer go run ./tools/catalog-importer/cmd/catalog-importer -source warframe -language ru -catalog-url http://localhost:8084

GOCACHE=/tmp/gocache-importer go run ./tools/catalog-importer/cmd/catalog-importer -source eve -language ru -catalog-url http://localhost:8084

GOCACHE=/tmp/gocache-importer go run ./tools/catalog-importer/cmd/catalog-importer -source tarkov -language ru -catalog-url http://localhost:8084

# 3. Показать масштаб данных в БД
docker compose -f deploy/docker-compose.yml --env-file deploy/.env exec -T postgres-catalog \
  psql -U gtrade -d gtrade_catalog \
  -c "SELECT game, source, COUNT(*) AS items_count FROM items GROUP BY game, source ORDER BY game, source;"

docker compose -f deploy/docker-compose.yml --env-file deploy/.env exec -T postgres-catalog \
  psql -U gtrade -d gtrade_catalog \
  -c "SELECT COUNT(*) AS total_items FROM items;"

docker compose -f deploy/docker-compose.yml --env-file deploy/.env exec -T postgres-catalog \
  psql -U gtrade -d gtrade_catalog \
  -c "SELECT COUNT(*) AS translated_items_ru FROM item_translations WHERE language_code='ru';"

# 4. Показать, что единый каталог хранит разные данные в одной модели
curl -sS "$CATALOG_URL/items?game=warframe&limit=3&offset=0&language=ru" \
  | jq '{warframe_items: [.items[] | {id, game, source, name, slug, image_url}]}'

curl -sS "$CATALOG_URL/items?game=eve&limit=3&offset=0&language=ru" \
  | jq '{eve_items: [.items[] | {id, game, source, name, slug, image_url}]}'

curl -sS "$CATALOG_URL/items?game=tarkov&limit=3&offset=0&language=ru" \
  | jq '{tarkov_items: [.items[] | {id, game, source, name, slug, image_url}]}'

# 5. Показать, что поиск работает по разным играм и форматам имен
curl -sS "$GATEWAY_URL/api/items/search?q=frost&game=warframe&language=ru&limit=5&offset=0" \
  | tee /tmp/gtrade-wow-warframe-search.json \
  | jq '{search: "warframe frost", items: [.items[] | {id, game, source, name, slug}]}'

curl -sS "$GATEWAY_URL/api/items/search?q=plagioclase&game=eve&language=ru&limit=5&offset=0" \
  | tee /tmp/gtrade-wow-eve-search.json \
  | jq '{search: "eve plagioclase", items: [.items[] | {id, game, source, name, slug}]}'

curl -sS "$GATEWAY_URL/api/items/search?q=makarov&game=tarkov&language=ru&limit=5&offset=0" \
  | tee /tmp/gtrade-wow-tarkov-search.json \
  | jq '{search: "tarkov makarov", items: [.items[] | {id, game, source, name, slug}]}'

export WF_ITEM_ID="$(jq -r '.items[0].id' /tmp/gtrade-wow-warframe-search.json)"
export EVE_ITEM_ID="$(jq -r '.items[0].id' /tmp/gtrade-wow-eve-search.json)"
export TARKOV_ITEM_ID="$(jq -r '.items[0].id' /tmp/gtrade-wow-tarkov-search.json)"

echo "$WF_ITEM_ID"
echo "$EVE_ITEM_ID"
echo "$TARKOV_ITEM_ID"

# 6. Сформировать уже готовый watchlist пользователя
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

# 7. Показать организацию пользовательских данных
curl -sS "$GATEWAY_URL/api/users/watchlist?user_id=$DEMO_USER_ID" \
  -H "Authorization: Bearer $JWT_TOKEN" \
  | jq '{watchlist: [.items[] | {watch_id: .id, item_id, game: .item.game, source: .item.source, name: .item.name, slug: .item.slug, image_url: .item.image_url}]}'

curl -sS "$GATEWAY_URL/api/users/users/$DEMO_USER_ID" \
  -H "Authorization: Bearer $JWT_TOKEN" \
  | jq '{user: .user, watchlist_count: (.watchlist | length), watchlist: [.watchlist[] | {game: .item.game, source: .item.source, name: .item.name}]}'

# 8. Runtime market data: показать, что по отслеживаемым данным приходят живые цены
curl -sS "$GATEWAY_URL/api/market/search?game=warframe&q=frost&limit=3&offset=0" \
  | tee /tmp/gtrade-wow-wf-market.json \
  | jq '{provider_search: "warframe", items: [.items[] | {id, game, source, name, currency}]}'

curl -sS "$GATEWAY_URL/api/market/search?game=eve&q=plex&limit=3&offset=0" \
  | tee /tmp/gtrade-wow-eve-market.json \
  | jq '{provider_search: "eve", items: [.items[] | {id, game, source, name, currency}]}'

curl -sS "$GATEWAY_URL/api/market/search?game=tarkov&q=makarov&game_mode=regular&limit=3&offset=0" \
  | tee /tmp/gtrade-wow-tarkov-market.json \
  | jq '{provider_search: "tarkov", items: [.items[] | {id, game, source, name, currency}]}'

export WF_MARKET_ID="$(jq -r '.items[0].id' /tmp/gtrade-wow-wf-market.json)"
export EVE_MARKET_ID="$(jq -r '.items[0].id' /tmp/gtrade-wow-eve-market.json)"
export TARKOV_MARKET_ID="$(jq -r '.items[0].id' /tmp/gtrade-wow-tarkov-market.json)"

echo "$WF_MARKET_ID"
echo "$EVE_MARKET_ID"
echo "$TARKOV_MARKET_ID"

curl -sS "$GATEWAY_URL/api/market/items/$WF_MARKET_ID/prices?game=warframe" \
  | jq '{warframe_price_snapshot: {item_id: .price.item_id, game: .price.game, source: .price.source, currency: .price.currency, market_kind: .price.market_kind, pricing: .price.pricing, analytics: .price.analytics}}'

curl -sS "$GATEWAY_URL/api/market/items/$EVE_MARKET_ID/prices?game=eve" \
  | jq '{eve_price_snapshot: {item_id: .price.item_id, game: .price.game, source: .price.source, currency: .price.currency, market_kind: .price.market_kind, pricing: .price.pricing, analytics: .price.analytics}}'

curl -sS "$GATEWAY_URL/api/market/items/$TARKOV_MARKET_ID/prices?game=tarkov&game_mode=regular" \
  | jq '{tarkov_price_snapshot: {item_id: .price.item_id, game: .price.game, game_mode: .price.game_mode, source: .price.source, currency: .price.currency, market_kind: .price.market_kind, pricing: .price.pricing, analytics: .price.analytics}}'

# 9. Свернуть данные до top-price для быстрого пользовательского решения
curl -sS "$GATEWAY_URL/api/market/items/$WF_MARKET_ID/top-price?game=warframe" | jq

curl -sS "$GATEWAY_URL/api/market/items/$EVE_MARKET_ID/top-price?game=eve" | jq

curl -sS "$GATEWAY_URL/api/market/items/$TARKOV_MARKET_ID/top-price?game=tarkov&game_mode=regular" | jq

# 10. Показать recent как быстрый повторный доступ к данным
curl -sS "$GATEWAY_URL/api/users/recent?user_id=$DEMO_USER_ID" \
  -H "Authorization: Bearer $JWT_TOKEN" \
  | jq '{recent_items: [.items[] | {watch_id: .id, game: .item.game, source: .item.source, name: .item.name}]}'

# 11. Финальный акцент на качестве
make test-fast
