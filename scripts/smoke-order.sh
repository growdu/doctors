#!/usr/bin/env bash
# order-service 端到端 smoke：
#   1. build
#   2. 启动（后台）
#   3. /healthz 通
#   4. /api/v1/orders GET 必须 401（无 token）
#   5. /api/v1/orders POST 无 token 也必须 401
#
# 前置：go build 通过；当前阶段 main 用 nil pool，业务调用会 panic，因此 smoke
# 只测启动 + 路由 + 鉴权拦截。

set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

ADDR=":8082"
BIN="$ROOT/bin/order"
LOGFILE="$ROOT/.data/order-smoke.log"
mkdir -p "$ROOT/bin" "$ROOT/.data"

echo "[1/5] building order-service..."
GOFLAGS="-mod=mod" GOPROXY="${GOPROXY:-https://goproxy.io,https://goproxy.cn,direct}" GOSUMDB="${GOSUMDB:-off}" \
  go build -o "$BIN" ./services/order/cmd

echo "[2/5] starting order-service on $ADDR..."
DOCTORS_ORDER_HTTP_ADDR="$ADDR" "$BIN" > "$LOGFILE" 2>&1 &
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

echo "[4/5] curl /api/v1/orders (no token → expect 401)"
RESP=$(curl -sS "http://127.0.0.1$ADDR/api/v1/orders")
echo "$RESP" | grep -q '"code":11001' || { echo "unauth unexpected: $RESP"; exit 1; }
echo "  -> $RESP"

echo "[5/5] curl /api/v1/orders/1 (no token → expect 401)"
RESP=$(curl -sS "http://127.0.0.1$ADDR/api/v1/orders/1")
echo "$RESP" | grep -q '"code":11001' || { echo "unauth unexpected: $RESP"; exit 1; }
echo "  -> $RESP"

echo "smoke OK"