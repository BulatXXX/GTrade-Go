#!/usr/bin/env bash
set -euo pipefail

GATEWAY_URL="${GATEWAY_URL:-http://localhost:8080}"
EMAIL="${EMAIL:-sattarowb@yandex.ru}"
PASSWORD="${PASSWORD:-TempPass_2026!}"

REGISTER_OUT="/tmp/gtrade-register.json"
RESET_OUT="/tmp/gtrade-password-reset-request.json"

pretty_print() {
  local file="$1"
  if command -v jq >/dev/null 2>&1; then
    jq . "$file"
  else
    python3 -m json.tool "$file"
  fi
}

echo "==> Register: $EMAIL"
register_status="$(
  curl -sS -o "$REGISTER_OUT" -w "%{http_code}" \
    -X POST "$GATEWAY_URL/api/auth/register" \
    -H 'Content-Type: application/json' \
    -d "{\"email\":\"$EMAIL\",\"password\":\"$PASSWORD\"}"
)"
echo "HTTP $register_status"
pretty_print "$REGISTER_OUT"
echo

if [ "$register_status" != "200" ] && [ "$register_status" != "201" ] && [ "$register_status" != "409" ]; then
  echo "Register request failed. Response saved to $REGISTER_OUT"
  exit 1
fi

if [ "$register_status" = "409" ]; then
  echo "User already exists. Continuing to password reset request."
  echo
fi

echo "==> Request password reset: $EMAIL"
reset_status="$(
  curl -sS -o "$RESET_OUT" -w "%{http_code}" \
    -X POST "$GATEWAY_URL/api/auth/password/reset/request" \
    -H 'Content-Type: application/json' \
    -d "{\"email\":\"$EMAIL\"}"
)"
echo "HTTP $reset_status"
pretty_print "$RESET_OUT"
echo

if [ "$reset_status" != "200" ] && [ "$reset_status" != "202" ]; then
  echo "Password reset request failed. Response saved to $RESET_OUT"
  exit 1
fi

cat <<EOF
Done.
- Register response: $REGISTER_OUT
- Reset response:    $RESET_OUT
- Email:             $EMAIL
- Password:          $PASSWORD
EOF
