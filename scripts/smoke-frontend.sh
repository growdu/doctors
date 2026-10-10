#!/usr/bin/env bash
# =============================================================================
# scripts/smoke-frontend.sh —— unified-app 前端 prod build 冒烟
# -----------------------------------------------------------------------------
# 用法：bash scripts/smoke-frontend.sh
# 前置：unified-app 已 pnpm build:h5
# =============================================================================

set -u

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT/frontend/unified-app"

LOG=/home/ubuntu/doctors/.data/logs/unified-dist.log
PORT=5180

ok()   { printf "${G}✓${N} %s\n" "$*"; }
fail() { printf "${R}✗${N} %s\n" "$*"; exit 1; }

if command -v tput >/dev/null 2>&1 && [ "$(tput colors 2>/dev/null || echo 0)" -ge 8 ]; then
  G='\033[0;32m'; R='\033[0;31m'; Y='\033[0;33m'; N='\033[0m'
else
  G=''; R=''; Y=''; N=''
fi

# 1) build
echo "--- build:h5 ---"
if timeout 120 pnpm build:h5 > "$LOG" 2>&1; then
  ok "build complete"
else
  fail "build failed; tail log: $(tail -10 "$LOG")"
fi

# 1a) copy vite.svg into dist (uni-app build doesn't auto-include public/)
cp /home/ubuntu/doctors/frontend/unified-app/public/vite.svg dist/build/h5/vite.svg 2>/dev/null && ok "vite.svg copied to dist" || printf "${Y}⚠${N} vite.svg copy skipped\n"

# 2) serve
echo "--- http.server :$PORT ---"
pkill -f "http.server $PORT" 2>/dev/null
sleep 1
cd dist/build/h5
nohup python3 -m http.server $PORT > "$LOG" 2>&1 &
SRV=$!
sleep 2
if kill -0 $SRV 2>/dev/null; then ok "served pid=$SRV"; else fail "server failed"; fi

# 3) curl
echo "--- curl checks ---"
code=$(curl -s -o /dev/null -w '%{http_code}' http://127.0.0.1:$PORT/)
[[ "$code" == "200" ]] && ok "/ = 200" || fail "/ = $code"

ASSET=$(ls assets/ 2>/dev/null | grep -E "^index-.*\.js$" | head -1)
[[ -n "$ASSET" ]] && ok "found main JS: $ASSET" || fail "no main JS found"
code=$(curl -s -o /dev/null -w '%{http_code}' "http://127.0.0.1:$PORT/assets/$ASSET")
[[ "$code" == "200" ]] && ok "/assets/$ASSET = 200" || fail "asset = $code"

code=$(curl -s -o /dev/null -w '%{http_code}' http://127.0.0.1:$PORT/vite.svg)
[[ "$code" == "200" ]] && ok "/vite.svg = 200" || fail "vite.svg = $code"

# 4) tabbar icons
echo "--- tabbar icons ---"
for f in patient-home patient-home-active patient-orders patient-orders-active patient-msg patient-msg-active patient-me patient-me-active; do
  code=$(curl -s -o /dev/null -w '%{http_code}' "http://127.0.0.1:$PORT/static/icons/tabbar/$f.png")
  [[ "$code" == "200" ]] && ok "$f.png = 200" || fail "$f.png = $code"
done

# 5) SPA fallback
echo "--- SPA fallback ---"
code=$(curl -s -o /dev/null -w '%{http_code}' http://127.0.0.1:$PORT/pages/patient/order/list)
if [[ "$code" == "200" ]]; then
  ok "/pages/patient/order/list = 200 (SPA fallback served index.html)"
else
  printf "${Y}⚠${N} /pages/patient/order/list = $code (this is the SPA fallback — only nginx supports it; python http.server returns 404). To test SPA fallback, run docker compose with nginx image.\n"
fi

# 6) cleanup
kill $SRV 2>/dev/null
echo ""
ok "frontend smoke OK"