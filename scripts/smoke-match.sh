#!/usr/bin/env bash
# match-service 端到端 smoke：
#   1. build
#   2. 启动
#   3. /healthz 通
#   4. /api/v1/match/feed 必须 401（无 token）
#   5. /internal/match/dispatch 必须 401（无 token）
#
# 真正端到端（auth + order + match 三服务联动）需要 docker compose + 三服务同启；
# 见 scripts/smoke-e2e.sh（待做）。

set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

ADDR=":8083"
BIN="$ROOT/bin/match"
LOGFILE="$ROOT/.data/match-smoke.log"
mkdir -p "$ROOT/bin" "$ROOT/.data"

echo "[1/5] building match-service..."
GOFLAGS="-mod=mod" GOPROXY="${GOPROXY:-https://goproxy.io,https://goproxy.cn,direct}" GOSUMDB="${GOSUMDB:-off}" \
  go build -o "$BIN" ./services/match/cmd

echo "[2/5] starting match-service on $ADDR..."
DOCTORS_MATCH_HTTP_ADDR="$ADDR" "$BIN" > "$LOGFILE" 2>&1 &
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

echo "[4/5] curl /api/v1/match/feed (no token → expect 401)"
RESP=$(curl -sS "http://127.0.0.1$ADDR/api/v1/match/feed?order_id=1")
echo "$RESP" | grep -q '"code":11001' || { echo "unauth unexpected: $RESP"; exit 1; }
echo "  -> $RESP"

echo "[5/5] curl /internal/match/dispatch (no token → expect 401)"
RESP=$(curl -sS -X POST "http://127.0.0.1$ADDR/internal/match/dispatch" \
  -H 'Content-Type: application/json' \
  -d '{"order_id": 1}')
echo "$RESP" | grep -q '"code":11001' || { echo "unauth unexpected: $RESP"; exit 1; }
echo "  -> $RESP"

echo "smoke OK"