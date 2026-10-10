#!/usr/bin/env bash
# =============================================================================
# scripts/dev.sh —— 一键全栈 dev 启动（后端 + 前端 + 健康巡检）
# -----------------------------------------------------------------------------
# 前置：本地 PG / Redis 已起；Kafka 用 NopPublisher 降级
# 用法：bash scripts/dev.sh start|stop|status|test|all
# =============================================================================

set -u

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

cmd=${1:-status}

start() {
  echo "=== starting backends (11 services) ==="
  bash scripts/start-services.sh restart
  echo ""
  echo "=== starting unified-app dev server (port 5174) ==="
  if ss -tnlp 2>/dev/null | grep -q ':5174 '; then
    echo "    :5174 already up"
  else
    cd frontend/unified-app
    nohup pnpm dev:h5 > /home/ubuntu/doctors/.data/logs/unified-app.log 2>&1 &
    echo "    started pid=$! (waiting 8s for vite ready)"
    sleep 8
    cd "$ROOT"
  fi
  echo ""
  echo "=== starting admin-web prod server (port 5192) ==="
  if [[ ! -d frontend/_archive_v1/admin-web/dist ]]; then
    echo "    no dist/ — run 'cd frontend/_archive_v1/admin-web && npm run build' first"
  elif ss -tnlp 2>/dev/null | grep -q ':5192 '; then
    echo "    :5192 already up"
  else
    cd frontend/_archive_v1/admin-web/dist
    nohup python3 -m http.server 5192 > /home/ubuntu/doctors/.data/logs/admin-dist.log 2>&1 &
    echo "    started pid=$!"
    sleep 1
    cd "$ROOT"
  fi
  status
}

stop() {
  echo "=== stopping all ==="
  bash scripts/start-services.sh stop
  pkill -f "uni$" 2>/dev/null && echo "    killed dev:h5"
  pkill -f "http.server 5180" 2>/dev/null && echo "    killed unified-app prod"
  pkill -f "http.server 5192" 2>/dev/null && echo "    killed admin-web prod"
  echo "OK"
}

status() {
  echo "=== backend status ==="
  bash scripts/check-all.sh
  echo ""
  echo "=== frontend status ==="
  for entry in "5174:unified-app-dev" "5180:unified-app-prod" "5192:admin-web-prod"; do
    port=${entry%%:*}; name=${entry##*:}
    code=$(curl -s -o /dev/null -w '%{http_code}' "http://127.0.0.1:$port/" 2>/dev/null || echo 000)
    printf "  [%-22s] :%s = %s\n" "$name" "$port" "$code"
  done
}

test() {
  echo "=== backend smoke ==="
  bash scripts/smoke-flow.sh | tail -8
  echo ""
  echo "=== frontend smoke ==="
  bash scripts/smoke-frontend.sh | tail -8
}

case "$cmd" in
  start) start ;;
  stop)  stop ;;
  status) status ;;
  test) test ;;
  all)   start; echo ""; test ;;
  *)     echo "usage: $0 {start|stop|status|test|all}"; exit 1 ;;
esac