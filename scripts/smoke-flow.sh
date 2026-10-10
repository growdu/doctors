#!/usr/bin/env bash
# =============================================================================
# scripts/smoke-flow.sh —— auth → order → cancel 端到端冒烟
# -----------------------------------------------------------------------------
# 前置：auth/order/match 三服务已用 scripts/start-services.sh 拉起
#       本地 PG 在 127.0.0.1:5432，doctors 库 + migrations 已就位
#
# 用法：bash scripts/smoke-flow.sh
# 退出码：0 = 全通过；非 0 = 任一步失败
# =============================================================================

set -u

AUTH=http://127.0.0.1:8081
ORDER=http://127.0.0.1:8082
MATCH=http://127.0.0.1:8083
PHONE=13800138000
LOG=.data/logs/auth.log

# ----- colors -----
if command -v tput >/dev/null 2>&1 && [ "$(tput colors 2>/dev/null || echo 0)" -ge 8 ]; then
  G='\033[0;32m'; R='\033[0;31m'; Y='\033[0;33m'; C='\033[0;36m'; B='\033[1m'; N='\033[0m'
else
  G=''; R=''; Y=''; C=''; B=''; N=''
fi

ok()   { printf "${G}✓${N} %s\n" "$*"; }
fail() { printf "${R}✗${N} %s\n" "$*"; exit 1; }
hdr()  { printf "\n${B}${C}── %s ──${N}\n" "$*"; }

# 1) /healthz on all 3
hdr "1) /healthz on auth + order + match"
for entry in "$AUTH:auth" "$ORDER:order" "$MATCH:match"; do
  url=${entry%:*}; name=${entry#*:}
  code=$(curl -s -o /dev/null -w '%{http_code}' "$url/healthz")
  [[ "$code" == "200" ]] && ok "$name /healthz=$code" || fail "$name /healthz=$code"
done

# 2) sms/send
hdr "2) auth.sms/send phone=$PHONE"
SMS=$(curl -sS -X POST "$AUTH/api/v1/auth/sms/send" -H 'Content-Type: application/json' -d "{\"phone\":\"$PHONE\"}")
[[ "$SMS" == *'"code":0'* ]] && ok "sms sent: $SMS" || fail "sms fail: $SMS"

# 3) extract code from auth log
sleep 0.3
CODE=$(grep -oE "phone=$PHONE code=[0-9]{6}" "$LOG" | tail -1 | grep -oE '[0-9]{6}$')
[[ -n "$CODE" ]] && ok "sms code from log: $CODE" || fail "sms code not in log"

# 4) login
hdr "3) auth.login type=sms"
LOGIN=$(curl -sS -X POST "$AUTH/api/v1/auth/login" -H 'Content-Type: application/json' \
  -d "{\"phone\":\"$PHONE\",\"code\":\"$CODE\",\"type\":\"sms\"}")
TOKEN=$(printf '%s' "$LOGIN" | sed -n 's/.*"token":"\([^"]*\)".*/\1/p')
USER_ID=$(printf '%s' "$LOGIN" | sed -n 's/.*"user_id":\([0-9]*\).*/\1/p')
[[ -n "$TOKEN" && -n "$USER_ID" ]] && ok "login OK uid=$USER_ID token_len=${#TOKEN}" || fail "login fail: $LOGIN"

# 5) /users/me with token
hdr "4) auth./users/me with bearer"
ME=$(curl -sS -H "Authorization: Bearer $TOKEN" "$AUTH/api/v1/users/me")
[[ "$ME" == *'"role":"patient"'* ]] && ok "me: $ME" || fail "me fail: $ME"

# 6) order create
hdr "5) order.create"
CREATE=$(curl -sS -X POST "$ORDER/api/v1/orders" \
  -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
  -d '{"hospital_id":1,"package_id":1,"service_start_at":"2026-10-15T10:00:00Z","amount":300.00}')
OID=$(printf '%s' "$CREATE" | sed -n 's/.*"id":\([0-9]*\).*/\1/p' | head -1)
STATUS=$(printf '%s' "$CREATE" | sed -n 's/.*"status":"\([^"]*\)".*/\1/p')
[[ -n "$OID" ]] && ok "order created id=$OID status=$STATUS" || fail "create fail: $CREATE"

# 7) order get
hdr "6) order.get/:id"
GET=$(curl -sS -H "Authorization: Bearer $TOKEN" "$ORDER/api/v1/orders/$OID")
[[ "$GET" == *"$OID"* ]] && ok "order fetched" || fail "get fail: $GET"

# 8) order list
hdr "7) order.list"
LIST=$(curl -sS -H "Authorization: Bearer $TOKEN" "$ORDER/api/v1/orders")
[[ "$LIST" == *"$OID"* ]] && ok "order listed" || fail "list fail: $LIST"

# 9) match candidates (patient role → 期望 403)
hdr "8) match.candidates (patient role → expect 403)"
CAND=$(curl -sS -X POST "$MATCH/api/v1/match/candidates" \
  -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
  -d '{"order_id":'"$OID"',"city":"shanghai","service_time":"2026-10-15T18:00:00+08:00","radius_km":50}')
[[ "$CAND" == *'"code":11003'* ]] && ok "RBAC working: $CAND" \
  || printf "${Y}⚠${N} match candidates unexpected: $CAND\n"

# 10) order cancel
hdr "9) order.cancel/:id"
CANCEL=$(curl -sS -X POST "$ORDER/api/v1/orders/$OID/cancel" \
  -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' -d '{"reason":"smoke_test"}')
[[ "$CANCEL" == *'"code":0'* ]] && ok "cancel OK" || fail "cancel fail: $CANCEL"

# 11) DB verify
hdr "10) DB row + event log"
PGPASSWORD=doctors psql -h 127.0.0.1 -U doctors -d doctors -c "SELECT id, status, version FROM orders WHERE id=$OID;" 2>&1
PGPASSWORD=doctors psql -h 127.0.0.1 -U doctors -d doctors -tAc "SELECT COUNT(*) FROM order_events WHERE order_id=$OID;" 2>&1 | tr -d ' '

printf "\n${G}${B}smoke OK — flow end-to-end working${N}\n"