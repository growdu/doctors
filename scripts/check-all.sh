#!/usr/bin/env bash
# =============================================================================
# scripts/check-all.sh —— 11 服务端口 + /healthz + /readyz + /metrics 一键巡检
# -----------------------------------------------------------------------------
# 用法：bash scripts/check-all.sh
# 退出码：0 全通；非 0 有服务异常
# =============================================================================

set -u

declare -a SVCS=(auth order match message payment review sos user escort wallet admin)
declare -A PORTS=(
  [auth]=8081 [order]=8082 [match]=8083
  [message]=8084 [payment]=8085 [review]=8086
  [sos]=8087 [user]=8088 [escort]=8089
  [wallet]=8090 [admin]=8091
)

# colors
if command -v tput >/dev/null 2>&1 && [ "$(tput colors 2>/dev/null || echo 0)" -ge 8 ]; then
  G='\033[0;32m'; R='\033[0;31m'; Y='\033[0;33m'; C='\033[0;36m'; B='\033[1m'; N='\033[0m'
else
  G=''; R=''; Y=''; C=''; B=''; N=''
fi

printf "${B}${C}%-10s %-5s %-4s %-4s %-8s %s${N}\n" SERVICE PORT HZ RZ METRICS PROC

fail=0
for s in "${SVCS[@]}"; do
  p=${PORTS[$s]}
  pidfile=".data/$s.pid"
  proc="-" ; [[ -f "$pidfile" ]] && proc=$(cat "$pidfile")
  h=$(curl -s -o /dev/null -w '%{http_code}' "http://127.0.0.1:$p/healthz" 2>/dev/null || echo 000)
  r=$(curl -s -o /dev/null -w '%{http_code}' "http://127.0.0.1:$p/readyz"  2>/dev/null || echo 000)
  m=$(curl -s "http://127.0.0.1:$p/metrics" 2>/dev/null | grep -c "^service_info" || echo 0)

  status="${G}OK${N}"
  [[ "$h" != "200" || "$r" != "200" || "$m" == "0" ]] && { status="${R}FAIL${N}"; fail=$((fail+1)); }

  printf "%-10s %-5s %-4s %-4s %-8s %s %s\n" "$s" "$p" "$h" "$r" "$m" "$proc" "$status"
done

if [[ $fail -eq 0 ]]; then
  printf "\n${G}${B}11/11 services healthy${N}\n"
  exit 0
else
  printf "\n${R}${B}$fail/11 services unhealthy${N}\n"
  exit 1
fi