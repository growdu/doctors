#!/usr/bin/env bash
# =============================================================================
# scripts/start-services.sh —— 一键拉起全 11 后端服务（本地 dev 模式）
# -----------------------------------------------------------------------------
# 前置：本地 PG 在 127.0.0.1:5432（doctors 库 + migrations 已就位）
#       本地 Redis 在 127.0.0.1:6379（apt install redis-server）
#       Kafka 未装 → NopPublisher fallback（业务 endpoint 仍可跑）
#
# 用法：
#   bash scripts/start-services.sh start    # 启动 11 服务
#   bash scripts/start-services.sh stop     # 停所有
#   bash scripts/start-services.sh status   # 看端口 + 健康
#   bash scripts/start-services.sh restart  # stop + start
# =============================================================================

set -u

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

PG_DSN='postgres://doctors:doctors@127.0.0.1:5432/doctors?sslmode=disable'
JWT_SECRET='dev-secret-change-me'
DATA_DIR='.data'
mkdir -p "$DATA_DIR/logs"

export PATH=/usr/local/go/bin:$PATH

# 11 服务 + 端口（与 config/<svc>.yaml 对齐；smoke-e2e.sh 同步）
SERVICES=(auth:8081 order:8082 match:8083 message:8084 payment:8085 review:8086 sos:8087 user:8088 escort:8089 wallet:8090 admin:8091)

cmd=${1:-start}

stop_all() {
  for entry in "${SERVICES[@]}"; do
    n=${entry%%:*}
    [[ -f "$DATA_DIR/$n.pid" ]] || continue
    pid=$(cat "$DATA_DIR/$n.pid")
    if kill -0 "$pid" 2>/dev/null; then
      kill "$pid" && echo "stopped $n (pid=$pid)"
    fi
    rm -f "$DATA_DIR/$n.pid"
  done
}

start_one() {
  local name=$1 port=$2
  local logfile="$DATA_DIR/logs/$name.log"
  local pidfile="$DATA_DIR/$name.pid"

  if [[ -f "$pidfile" ]] && kill -0 "$(cat "$pidfile")" 2>/dev/null; then
    echo "[$name] already running pid=$(cat "$pidfile")"
    return
  fi

  if [[ ! -x "./bin/$name" ]]; then
    echo "[$name] SKIP: bin/$name not built"
    return
  fi

  # order-service 在 dev 模式跳过 auth user lookup（devFakeUsers）
  local extra=""
  if [[ "$name" == "order" ]]; then
    extra="DOCTORS_DEV_FAKE_USER=1"
  fi

  # shellcheck disable=SC2086
  env DOCTORS_DB_DSN="$PG_DSN" \
      DOCTORS_AUTH_JWT_SECRET="$JWT_SECRET" \
      DOCTORS_REDIS_ADDR="127.0.0.1:6379" \
      DOCTORS_KAFKA_BROKERS="" \
      DOCTORS_TRACING_OTLP_ENDPOINT="" \
      DOCTORS_LOGGING_LEVEL="info" \
      $extra \
      ./bin/"$name" > "$logfile" 2>&1 &
  local pid=$!
  echo "$pid" > "$pidfile"
  sleep 0.2
  if kill -0 "$pid" 2>/dev/null; then
    echo "[$name] started :$port pid=$pid"
  else
    echo "[$name] FAILED; tail log:"
    tail -10 "$logfile"
  fi
}

start_all() {
  echo "==> build (incremental)"
  for entry in "${SERVICES[@]}"; do
    n=${entry%%:*}
    go build -o "./bin/$n" "./services/$n/cmd" 2>&1 | tail -2
  done

  for entry in "${SERVICES[@]}"; do
    n=${entry%%:*}; p=${entry##*:}
    start_one "$n" "$p"
  done

  echo "==> waiting 6s for ready"
  sleep 6
  status_all
}

status_all() {
  echo "==> status:"
  for entry in "${SERVICES[@]}"; do
    n=${entry%%:*}; p=${entry##*:}
    if [[ -f "$DATA_DIR/$n.pid" ]] && kill -0 "$(cat "$DATA_DIR/$n.pid")" 2>/dev/null; then
      h=$(curl -s -o /dev/null -w '%{http_code}' "http://127.0.0.1:$p/healthz" 2>/dev/null || echo 000)
      r=$(curl -s -o /dev/null -w '%{http_code}' "http://127.0.0.1:$p/readyz" 2>/dev/null || echo 000)
      printf "  [%-8s] :%s  pid=%-6s  healthz=%s  readyz=%s\n" "$n" "$p" "$(cat "$DATA_DIR/$n.pid")" "$h" "$r"
    else
      printf "  [%-8s] :%s  NOT RUNNING\n" "$n" "$p"
    fi
  done
}

case "$cmd" in
  start)   start_all ;;
  stop)    stop_all ;;
  status)  status_all ;;
  restart) stop_all; sleep 2; start_all ;;
  *)       echo "usage: $0 {start|stop|status|restart}"; exit 1 ;;
esac