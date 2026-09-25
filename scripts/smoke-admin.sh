#!/usr/bin/env bash
# admin-service smoke 脚本：
#   1. build
#   2. 启动（后台）
#   3. /healthz 通
#   4. /api/v1/admin/users GET 必须 401（无 token）
#   5. /api/v1/admin/orders/1/force-cancel POST 也必须 401（无 token；RoleAuth 后还能 403）
#
# 前置：go build 通过；当前阶段 main 用 nil pool，业务调用会 panic，因此 smoke
# 只测启动 + 路由 + 鉴权拦截。

set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

ADDR="${ADMIN_ADDR:-:8089}"
BIN="$ROOT/bin/admin"
LOGFILE="$ROOT/.data/admin-smoke.log"
mkdir -p "$ROOT/bin" "$ROOT/.data"

echo "[1/5] building admin-service..."
GOPROXY="${GOPROXY:-https://goproxy.io,https://goproxy.cn,direct}" GOSUMDB="${GOSUMDB:-off}" \
  go build -o "$BIN" ./services/admin/cmd

echo "[2/5] starting admin-service on $ADDR (healthz-only mode; no DSN / Kafka)..."
DOCTORS_ADMIN_HTTP_ADDR="$ADDR" DOCTORS_ADMIN_KAFKA_BROKERS="" "$BIN" > "$LOGFILE" 2>&1 &
PID=$!
trap 'kill $PID 2>/dev/null || true; wait $PID 2>/dev/null || true' EXIT

for i in $(seq 1 30); do
  if curl -fsS "http://127.0.0.1$ADDR/healthz" > /dev/null 2>&1; then
    echo "  /healthz OK after ${i}00ms"
    break
  fi
  sleep 0.1
done

echo "[3/5] curl /healthz"
HEALTH=$(curl -fsS "http://127.0.0.1$ADDR/healthz")
echo "$HEALTH" | grep -q '"status":"ok"' || { echo "healthz unexpected: $HEALTH"; exit 1; }
echo "  -> $HEALTH"

echo "[4/5] curl /api/v1/admin/users (no token → expect 11001)"
RESP=$(curl -sS "http://127.0.0.1$ADDR/api/v1/admin/users")
echo "$RESP" | grep -q '"code":11001' || { echo "unauth unexpected: $RESP"; exit 1; }
echo "  -> $RESP"

echo "[5/5] curl /api/v1/admin/orders/1/force-cancel (no token → expect 11001)"
RESP=$(curl -sS -X POST "http://127.0.0.1$ADDR/api/v1/admin/orders/1/force-cancel")
echo "$RESP" | grep -q '"code":11001' || { echo "unauth unexpected: $RESP"; exit 1; }
echo "  -> $RESP"

echo "smoke OK"