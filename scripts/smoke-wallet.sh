#!/usr/bin/env bash
# wallet-service smoke 脚本：起服务 + /healthz + 4 个 endpoint 401 拦截。
#
# 用法：bash scripts/smoke-wallet.sh
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
ADDR="${WALLET_ADDR:-:8088}"
BIN="$ROOT/bin/wallet"
LOGFILE="$ROOT/.data/wallet-smoke.log"
mkdir -p "$ROOT/bin" "$ROOT/.data"

echo "[1/5] building wallet-service..."
GOPROXY="${GOPROXY:-https://goproxy.io,https://goproxy.cn,direct}" GOSUMDB="${GOSUMDB:-off}" \
  go build -o "$BIN" ./services/wallet/cmd

echo "[2/5] starting wallet-service on $ADDR (healthz-only mode; no DSN)..."
DOCTORS_WALLET_HTTP_ADDR="$ADDR" "$BIN" > "$LOGFILE" 2>&1 &
PID=$!
trap 'kill $PID 2>/dev/null || true; wait $PID 2>/dev/null || true' EXIT

# wait for /healthz
for i in {1..10}; do
  if curl -fsS "http://127.0.0.1$ADDR/healthz" > /dev/null 2>&1; then
    echo "  /healthz OK after ${i}00ms"
    break
  fi
  sleep 0.1
done

echo "[3/5] curl /healthz"
curl -fsS "http://127.0.0.1$ADDR/healthz" | head -c 200
echo

echo "[4/5] curl 4 个 endpoint（无 token → 401）"
for url in \
  "/api/v1/users/me/wallet" \
  "/api/v1/escorts/me/wallet" \
  "/api/v1/wallet/transactions"; do
  code=$(curl -sS -o /dev/null -w "%{http_code}" "http://127.0.0.1$ADDR$url")
  echo "  GET $url → HTTP $code"
done
code=$(curl -sS -o /dev/null -w "%{http_code}" -X POST -H "Content-Type: application/json" \
  -d '{"amount":"200","channel":"wx","account":"138****0000"}' \
  "http://127.0.0.1$ADDR/api/v1/escorts/me/wallet/withdraw")
echo "  POST /api/v1/escorts/me/wallet/withdraw → HTTP $code"
echo "[5/5] done"
echo "smoke OK"