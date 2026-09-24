#!/usr/bin/env bash
# auth-service 端到端 smoke：
#   1. 启动 auth-service（后台）
#   2. 等待 /healthz 通
#   3. 调用 sms/send + login（当前阶段 DB 未接通，login 会返回"repo not wired"，仅校验路由生效）
#   4. 关闭服务
#
# 用法：bash scripts/smoke-auth.sh
#
# 前置：
#   - go build ./services/auth/cmd 正常
#   - config/auth.yaml 存在（默认随仓库提供）
#
# 真正跑业务流需要 DB 接通（阶段 2.10），那会在 scripts/smoke-e2e.sh 覆盖。

set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

ADDR=":8081"
BIN="$ROOT/bin/auth"
LOGFILE="$ROOT/.data/auth-smoke.log"
mkdir -p "$ROOT/bin" "$ROOT/.data"

echo "[1/5] building auth-service..."
GOFLAGS="-mod=mod" GOPROXY="${GOPROXY:-https://goproxy.io,https://goproxy.cn,direct}" GOSUMDB="${GOSUMDB:-off}" \
  go build -o "$BIN" ./services/auth/cmd

echo "[2/5] starting auth-service on $ADDR..."
DOCTORS_AUTH_HTTP_ADDR="$ADDR" "$BIN" > "$LOGFILE" 2>&1 &
PID=$!
trap 'kill $PID 2>/dev/null || true; wait $PID 2>/dev/null || true' EXIT

# 等 /healthz
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

echo "[4/5] curl /api/v1/auth/sms/send"
SMS=$(curl -fsS -X POST "http://127.0.0.1$ADDR/api/v1/auth/sms/send" \
  -H 'Content-Type: application/json' \
  -d '{"phone":"13800138000"}')
echo "  -> $SMS"

echo "[5/5] curl /api/v1/users/me (no token → expect non-zero code)"
ME=$(curl -sS "http://127.0.0.1$ADDR/api/v1/users/me")
echo "$ME" | grep -q '"code":11001' || { echo "unauth /me unexpected: $ME"; exit 1; }
echo "  -> $ME"

echo "smoke OK"