#!/bin/zsh

# GTrade demo curl script
# Copy-paste commands step by step during the recording.

export GATEWAY_URL=http://localhost:8080
export AUTH_URL=http://localhost:8081
export USER_ASSET_URL=http://localhost:8082
export INTEGRATION_URL=http://localhost:8083
export CATALOG_URL=http://localhost:8084
export NOTIFICATION_URL=http://localhost:8085

export SMOKE_EMAIL="smoke.$(date +%s)@example.com"
export SMOKE_PASSWORD="secret123"
export SMOKE_USER_ID=$(( (RANDOM % 900000) + 100000 ))

# 1. Health checks
for port in 8080 8081 8082 8083 8084 8085; do
  echo "PORT=$port"
  curl -fsS "http://localhost:${port}/health"
  echo
done

# 2. Register through gateway
curl -sS -X POST "$GATEWAY_URL/api/auth/register" \
  -H 'Content-Type: application/json' \
  -d "{
    \"email\": \"$SMOKE_EMAIL\",
    \"password\": \"$SMOKE_PASSWORD\"
  }" | tee /tmp/gtrade-register.json

# 3. Save tokens
export JWT_TOKEN="$(jq -r '.access_token' /tmp/gtrade-register.json)"
export REFRESH_TOKEN="$(jq -r '.refresh_token' /tmp/gtrade-register.json)"

echo "$JWT_TOKEN"
echo "$REFRESH_TOKEN"

# 4. Login through gateway
curl -sS -X POST "$GATEWAY_URL/api/auth/login" \
  -H 'Content-Type: application/json' \
  -d "{
    \"email\": \"$SMOKE_EMAIL\",
    \"password\": \"$SMOKE_PASSWORD\"
  }"

# 5. Refresh token through gateway
curl -sS -X POST "$GATEWAY_URL/api/auth/refresh" \
  -H 'Content-Type: application/json' \
  -d "{
    \"refresh_token\": \"$REFRESH_TOKEN\"
  }"

# 6. Trigger email flow: password reset
curl -sS -X POST "$AUTH_URL/password/reset/request" \
  -H 'Content-Type: application/json' \
  -d "{
    \"email\": \"$SMOKE_EMAIL\"
  }"

# 7. Trigger email flow: email verification
curl -sS -X POST "$AUTH_URL/email/verify" \
  -H 'Content-Type: application/json' \
  -d "{
    \"email\": \"$SMOKE_EMAIL\"
  }"

# 8. Create user profile through protected gateway route
curl -sS -X POST "$GATEWAY_URL/api/users/users" \
  -H "Authorization: Bearer $JWT_TOKEN" \
  -H 'Content-Type: application/json' \
  -d "{
    \"user_id\": $SMOKE_USER_ID,
    \"display_name\": \"Smoke User\"
  }"

# 9. Get user profile
curl -sS "$GATEWAY_URL/api/users/users/$SMOKE_USER_ID" \
  -H "Authorization: Bearer $JWT_TOKEN"

# 10. Update user profile
curl -sS -X PUT "$GATEWAY_URL/api/users/users/$SMOKE_USER_ID" \
  -H "Authorization: Bearer $JWT_TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{
    "display_name": "Smoke User Updated",
    "avatar_url": "https://example.com/avatar.png",
    "bio": "Trader and collector"
  }'

# 11. Update preferences
curl -sS -X PUT "$GATEWAY_URL/api/users/preferences" \
  -H "Authorization: Bearer $JWT_TOKEN" \
  -H 'Content-Type: application/json' \
  -d "{
    \"user_id\": $SMOKE_USER_ID,
    \"currency\": \"usd\",
    \"notifications_enabled\": true
  }"

# 12. Get preferences
curl -sS "$GATEWAY_URL/api/users/preferences?user_id=$SMOKE_USER_ID" \
  -H "Authorization: Bearer $JWT_TOKEN"

# 13. Search items in local catalog
curl -sS "$GATEWAY_URL/api/items/search?q=frost&game=warframe&language=ru&limit=5&offset=0" \
  | tee /tmp/gtrade-catalog-search.json

# 14. Save catalog item id
export CATALOG_ITEM_ID="$(jq -r '.items[0].id' /tmp/gtrade-catalog-search.json)"
echo "$CATALOG_ITEM_ID"

# 15. Get local catalog item
curl -sS "$GATEWAY_URL/api/items/$CATALOG_ITEM_ID?language=ru"

# 16. Search runtime market items
curl -sS "$GATEWAY_URL/api/market/search?game=warframe&q=frost&limit=3&offset=0" \
  | tee /tmp/gtrade-market-search.json

# 17. Save market provider item id
export MARKET_ITEM_ID="$(jq -r '.items[0].id' /tmp/gtrade-market-search.json)"
echo "$MARKET_ITEM_ID"

# 18. Get runtime market item
curl -sS "$GATEWAY_URL/api/market/items/$MARKET_ITEM_ID?game=warframe"

# 19. Get runtime market prices
curl -sS "$GATEWAY_URL/api/market/items/$MARKET_ITEM_ID/prices?game=warframe"

# 20. Get runtime top price
curl -sS "$GATEWAY_URL/api/market/items/$MARKET_ITEM_ID/top-price?game=warframe"

# 21. Add item to watchlist
curl -sS -X POST "$GATEWAY_URL/api/users/watchlist" \
  -H "Authorization: Bearer $JWT_TOKEN" \
  -H 'Content-Type: application/json' \
  -d "{
    \"user_id\": $SMOKE_USER_ID,
    \"item_id\": \"$CATALOG_ITEM_ID\"
  }" | tee /tmp/gtrade-watch-create.json

# 22. Get watchlist
curl -sS "$GATEWAY_URL/api/users/watchlist?user_id=$SMOKE_USER_ID" \
  -H "Authorization: Bearer $JWT_TOKEN"

# 23. Get recent items
curl -sS "$GATEWAY_URL/api/users/recent?user_id=$SMOKE_USER_ID" \
  -H "Authorization: Bearer $JWT_TOKEN"

# 24. Save watchlist id
export WATCHLIST_ID="$(jq -r '.id' /tmp/gtrade-watch-create.json)"
echo "$WATCHLIST_ID"

# 25. Delete watchlist item
curl -i -X DELETE "$GATEWAY_URL/api/users/watchlist/$WATCHLIST_ID?user_id=$SMOKE_USER_ID" \
  -H "Authorization: Bearer $JWT_TOKEN"

# 26. Request with explicit request id for log demo
curl -i "$GATEWAY_URL/api/items/$CATALOG_ITEM_ID?language=ru" \
  -H 'X-Request-ID: demo-request-001'
