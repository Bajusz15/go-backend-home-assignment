#!/usr/bin/env bash
set -euo pipefail

BASE_URL="${BASE_URL:-http://localhost:8080}"
CUSTOMER_EMAIL="${SMOKE_CUSTOMER_EMAIL:-smoke-$(date +%s)@test.com}"
PASS=0
FAIL=0

# --- helpers ---

check() {
  local name="$1" expected="$2" actual="$3"
  if [ "$actual" = "$expected" ]; then
    echo "  ✓ $name"
    PASS=$((PASS + 1))
  else
    echo "  ✗ $name (expected $expected, got $actual)"
    FAIL=$((FAIL + 1))
  fi
}

check_not_empty() {
  local name="$1" value="$2"
  if [ -n "$value" ]; then
    echo "  ✓ $name"
    PASS=$((PASS + 1))
  else
    echo "  ✗ $name (empty)"
    FAIL=$((FAIL + 1))
  fi
}

jq_field() {
  python3 -c "import sys,json; print(json.load(sys.stdin)$1)" 2>/dev/null
}

# --- 1. wait for health ---

echo "Waiting for API at $BASE_URL ..."
for i in $(seq 1 30); do
  if curl -sf "$BASE_URL/health" > /dev/null 2>&1; then
    echo "  ✓ API is healthy"
    PASS=$((PASS + 1))
    break
  fi
  if [ "$i" -eq 30 ]; then
    echo "  ✗ API did not become healthy in 30s"
    exit 1
  fi
  sleep 1
done

# --- 2. register customer ---

echo "Register customer ..."
REGISTER_RESP=$(curl -sf -X POST "$BASE_URL/auth/register" \
  -H 'Content-Type: application/json' \
  -d "{\"email\":\"$CUSTOMER_EMAIL\",\"password\":\"password123\",\"name\":\"Smoke Tester\"}")
CUSTOMER_TOKEN=$(echo "$REGISTER_RESP" | jq_field "['token']")
check "register succeeds" "0" "$?"
check_not_empty "token is present" "$CUSTOMER_TOKEN"

# --- 3. login customer ---

echo "Login customer ..."
LOGIN_RESP=$(curl -sf -X POST "$BASE_URL/auth/login" \
  -H 'Content-Type: application/json' \
  -d "{\"email\":\"$CUSTOMER_EMAIL\",\"password\":\"password123\"}")
LOGIN_TOKEN=$(echo "$LOGIN_RESP" | jq_field "['token']")
check_not_empty "login returns token" "$LOGIN_TOKEN"

# --- 4. who-am-i ---

echo "Who am I ..."
WHOAMI_RESP=$(curl -sf "$BASE_URL/auth/who-am-i" \
  -H "Authorization: Bearer $CUSTOMER_TOKEN")
WHOAMI_EMAIL=$(echo "$WHOAMI_RESP" | jq_field "['email']")
WHOAMI_ROLE=$(echo "$WHOAMI_RESP" | jq_field "['role']")
check "email matches" "$CUSTOMER_EMAIL" "$WHOAMI_EMAIL"
check "role is customer" "customer" "$WHOAMI_ROLE"

# --- 5. list restaurants, pick Mario's, get a menu item ---

echo "List restaurants ..."
RESTAURANTS_RESP=$(curl -sf "$BASE_URL/restaurants")
RESTAURANT_COUNT=$(echo "$RESTAURANTS_RESP" | jq_field ".__len__()")
check "restaurants exist" "true" "$([ "$RESTAURANT_COUNT" -gt 0 ] && echo true || echo false)"

RESTAURANT_ID=$(echo "$RESTAURANTS_RESP" | python3 -c "
import sys, json
rs = json.load(sys.stdin)
mario = [r for r in rs if 'Mario' in r['name']]
print(mario[0]['id'] if mario else rs[0]['id'])
")
check_not_empty "got restaurant id" "$RESTAURANT_ID"

echo "Get restaurant menu ..."
MENU_RESP=$(curl -sf "$BASE_URL/restaurants/$RESTAURANT_ID/menu")
MENU_ITEM_ID=$(echo "$MENU_RESP" | python3 -c "
import sys, json
items = json.load(sys.stdin)
available = [i for i in items if i['available']]
print(available[0]['id'] if available else '')
")
check_not_empty "got available menu item" "$MENU_ITEM_ID"

# --- 6. place order ---

echo "Place order ..."
ORDER_RESP=$(curl -sf -X POST "$BASE_URL/orders" \
  -H 'Content-Type: application/json' \
  -H "Authorization: Bearer $CUSTOMER_TOKEN" \
  -d "{\"restaurant_id\":\"$RESTAURANT_ID\",\"items\":[{\"menu_item_id\":\"$MENU_ITEM_ID\",\"quantity\":2}]}")
ORDER_ID=$(echo "$ORDER_RESP" | jq_field "['id']")
ORDER_STATUS=$(echo "$ORDER_RESP" | jq_field "['status']")
check "order status is received" "received" "$ORDER_STATUS"
check_not_empty "got order id" "$ORDER_ID"

# --- 7. login seeded restaurant ---

echo "Login restaurant ..."
REST_RESP=$(curl -sf -X POST "$BASE_URL/auth/login" \
  -H 'Content-Type: application/json' \
  -d '{"email":"mario@restaurant.com","password":"restaurant123"}')
REST_TOKEN=$(echo "$REST_RESP" | jq_field "['token']")
REST_ROLE=$(echo "$REST_RESP" | jq_field "['user']['role']")
check "restaurant role" "restaurant" "$REST_ROLE"
check_not_empty "restaurant token" "$REST_TOKEN"

# --- 8. list restaurant orders ---

echo "List restaurant orders ..."
ORDERS_RESP=$(curl -sf "$BASE_URL/orders" \
  -H "Authorization: Bearer $REST_TOKEN")
ORDERS_COUNT=$(echo "$ORDERS_RESP" | jq_field ".__len__()")
check "restaurant has orders" "true" "$([ "$ORDERS_COUNT" -gt 0 ] && echo true || echo false)"

# --- 9. fetch order detail ---

echo "Get order detail ..."
DETAIL_RESP=$(curl -sf "$BASE_URL/orders/$ORDER_ID" \
  -H "Authorization: Bearer $REST_TOKEN")
DETAIL_CUSTOMER=$(echo "$DETAIL_RESP" | jq_field "['customer']['email']")
DETAIL_ITEMS=$(echo "$DETAIL_RESP" | jq_field "['items'].__len__()")
check "order has customer email" "$CUSTOMER_EMAIL" "$DETAIL_CUSTOMER"
check "order has items" "true" "$([ "$DETAIL_ITEMS" -gt 0 ] && echo true || echo false)"

# --- 10. advance order status ---

echo "Update order status ..."
UPDATE_RESP=$(curl -sf -X PATCH "$BASE_URL/orders/$ORDER_ID" \
  -H 'Content-Type: application/json' \
  -H "Authorization: Bearer $REST_TOKEN" \
  -d '{"status":"preparing"}')
NEW_STATUS=$(echo "$UPDATE_RESP" | jq_field "['status']")
check "status updated to preparing" "preparing" "$NEW_STATUS"

# --- summary ---

echo ""
echo "===================="
echo "Smoke test complete: $PASS passed, $FAIL failed"
echo "===================="
[ "$FAIL" -eq 0 ] || exit 1
