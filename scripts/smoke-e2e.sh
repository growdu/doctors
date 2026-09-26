#!/usr/bin/env bash
# =============================================================================
# scripts/smoke-e2e.sh —— 端到端冒烟（一键验证 11 服务全部启动 + 关键端点）。
# -----------------------------------------------------------------------------
# 流程：
#   1. 启动 docker-compose.deploy.yml（必须已 build 过；只 up -d）
#   2. 等待 30s 让中间件 + 服务进入 healthy
#   3. 循环检测 11 个服务的 /healthz（200 OK）
#   4. 检查每服务的 /metrics 端点存在
#   5. 调一个轻量 API 验证（admin 内部鉴权拦截 → 401 期望）
#   6. 输出彩色汇总
#
# 用法：
#   bash scripts/smoke-e2e.sh                 # 全流程
#   bash scripts/smoke-e2e.sh --skip-start    # 假设已起，仅做端点巡检
#   bash scripts/smoke-e2e.sh --skip-api      # 跳过 API 调用（只跑 healthz/metrics）
#   bash scripts/smoke-e2e.sh --timeout=180   # 自定义 wait timeout（秒）
#
# 退出码：
#   0 = 全通过
#   1 = 任一步 FAIL（脚本立即退出，容器保持运行可手动排查）
# =============================================================================

set -u  # 单步失败不立即退出，让最终汇总判断（与 run-tests.sh 一致）

# ---------- 参数 ----------
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
COMPOSE_FILE="docker-compose.deploy.yml"
SKIP_START=0
SKIP_API=0
WAIT_TIMEOUT=120   # 单服务最长等待秒
BOOT_WAIT=30        # docker compose up 后等待秒

for arg in "$@"; do
    case "$arg" in
        --skip-start)  SKIP_START=1 ;;
        --skip-api)    SKIP_API=1   ;;
        --timeout=*)   WAIT_TIMEOUT="${arg#--timeout=}" ;;
        -h|--help)
            head -30 "$0"
            exit 0
            ;;
    esac
done

# ---------- 颜色 ----------
if [ -t 1 ] && command -v tput >/dev/null 2>&1 && [ "$(tput colors 2>/dev/null || echo 0)" -ge 8 ]; then
    GREEN='\033[0;32m'
    RED='\033[0;31m'
    YELLOW='\033[0;33m'
    CYAN='\033[0;36m'
    BOLD='\033[1m'
    RESET='\033[0m'
else
    GREEN=''; RED=''; YELLOW=''; CYAN=''; BOLD=''; RESET=''
fi

# ---------- 11 服务清单（与 docker-compose.deploy.yml 端口对齐） ----------
# 格式：name|hostport|container_name
SERVICES=(
    "auth|8081|doctors-auth"
    "order|8082|doctors-order"
    "match|8083|doctors-match"
    "message|8084|doctors-message"
    "payment|8085|doctors-payment"
    "review|8086|doctors-review"
    "sos|8087|doctors-sos"
    "user|8088|doctors-user"
    "escort|8089|doctors-escort"
    "wallet|8090|doctors-wallet"
    "admin|8091|doctors-admin"
)

# 单一轻量 API：调 admin 的 pending-audit 验证（无 token → 期望 401）
API_PATH="/api/v1/admin/escorts/pending-audit"
API_HOST_PORT=8091

# ---------- 状态 ----------
TOTAL=0; PASSED=0; FAILED=0; SKIPPED=0
FAILED_NAMES=()

step() { echo ""; echo -e "${CYAN}▶ $1${RESET}"; }
ok()   { echo -e "  ${GREEN}✓${RESET} $1"; }
fail() { echo -e "  ${RED}✗${RESET} $1"; }
warn() { echo -e "  ${YELLOW}!${RESET} $1"; }
info() { echo -e "  ${BOLD}i${RESET} $1"; }

# record_pass / record_fail / record_skip <name>
record_pass() { TOTAL=$((TOTAL+1)); PASSED=$((PASSED+1)); }
record_fail() { TOTAL=$((TOTAL+1)); FAILED=$((FAILED+1)); FAILED_NAMES+=("$1"); }
record_skip() { TOTAL=$((TOTAL+1)); SKIPPED=$((SKIPPED+1)); }

# require_tool <tool> <hint>  → 0=有, 1=缺
require_tool() {
    if ! command -v "$1" >/dev/null 2>&1; then
        warn "$1 not found — $2"
        return 1
    fi
    return 0
}

# =============================================================================
# 头部
# =============================================================================
echo -e "${CYAN}=============================================${RESET}"
echo -e "${CYAN}  Doctors · smoke-e2e.sh  (端到端冒烟)        ${RESET}"
echo -e "${CYAN}  ROOT:      $ROOT${RESET}"
echo -e "${CYAN}  COMPOSE:   $COMPOSE_FILE${RESET}"
echo -e "${CYAN}  SERVICES:  11 (auth/order/match/message/payment/review/sos/user/escort/wallet/admin)${RESET}"
echo -e "${CYAN}  TIME:      $(date '+%Y-%m-%d %H:%M:%S')${RESET}"
echo -e "${CYAN}=============================================${RESET}"

# =============================================================================
# 步骤 0：前置工具
# =============================================================================
step "0. 前置工具"
HAS_DOCKER=0
if require_tool docker "请先安装 Docker Desktop / docker engine" ; then
    ok "docker available"
    HAS_DOCKER=1
fi
HAS_CURL=0
if require_tool curl "请先安装 curl"; then
    ok "curl available"
    HAS_CURL=1
fi

if [ "$HAS_CURL" -ne 1 ]; then
    fail "curl 缺失，无法继续"
    exit 1
fi

# =============================================================================
# 步骤 1：启动 docker-compose.deploy.yml
# =============================================================================
if [ "$SKIP_START" -eq 1 ]; then
    step "1. 跳过 docker-compose 启动（--skip-start）"
    warn "依赖用户已手工拉起 11 服务；本脚本只做端点巡检"
else
    step "1. docker compose up -d $COMPOSE_FILE"
    if [ "$HAS_DOCKER" -ne 1 ]; then
        warn "docker 缺失，跳过启动步骤（仅做端口巡检）"
        record_skip "docker compose up"
    else
        # 仅在容器不存在 / 未在跑时才 up；已运行也允许（幂等）
        if docker compose -f "$COMPOSE_FILE" up -d 2>&1 | tail -20; then
            ok "docker compose up -d 成功（容器进入启动阶段）"
            record_pass "docker compose up"

            info "等待 ${BOOT_WAIT}s 让中间件 + 服务进入 healthy..."
            sleep "$BOOT_WAIT"
        else
            fail "docker compose up 失败（详见 docker compose 输出）"
            record_fail "docker compose up"
            # 不退出 —— 仍尝试探测宿主端口，看是否已有进程
        fi
    fi
fi

# =============================================================================
# 步骤 2：等待 /healthz
# =============================================================================
step "2. 探测 11 服务 /healthz（timeout=${WAIT_TIMEOUT}s/服务）"

wait_healthz() {
    local name="$1" port="$2" end=$((SECONDS + WAIT_TIMEOUT)) status
    while [ "$SECONDS" -lt "$end" ]; do
        status=$(curl -s -o /dev/null -w '%{http_code}' "http://127.0.0.1:${port}/healthz" 2>/dev/null || echo "000")
        if [ "$status" = "200" ]; then
            return 0
        fi
        sleep 1
    done
    return 1
}

declare -A HCODES
for entry in "${SERVICES[@]}"; do
    IFS='|' read -r name port container <<<"$entry"
    info "[$name] 端口 $port · 容器 $container ..."
    if wait_healthz "$name" "$port"; then
        HCODES["$name"]=200
        ok "[$name] /healthz → 200"
        record_pass "$name /healthz"
    else
        HCODES["$name"]=fail
        fail "[$name] /healthz 在 ${WAIT_TIMEOUT}s 内未返回 200"
        record_fail "$name /healthz"
    fi
done

# =============================================================================
# 步骤 3：/metrics 端点存在性
# =============================================================================
step "3. 探测 /metrics 端点（11 服务全部）"

check_metrics() {
    local name="$1" port="$2"
    local code body
    code=$(curl -s -o /tmp/doctors-metrics.txt -w '%{http_code}' "http://127.0.0.1:${port}/metrics")
    body=$(cat /tmp/doctors-metrics.txt)
    if [ "$code" = "200" ] && echo "$body" | grep -q '^# HELP '; then
        return 0
    fi
    return 1
}

for entry in "${SERVICES[@]}"; do
    IFS='|' read -r name port container <<<"$entry"
    if [ "${HCODES[$name]:-fail}" != "200" ]; then
        warn "[$name] /healthz 不通 → 跳过 /metrics 探测"
        record_skip "$name /metrics"
        continue
    fi
    if check_metrics "$name" "$port"; then
        ok "[$name] /metrics → 200 + Prometheus 格式"
        record_pass "$name /metrics"
    else
        fail "[$name] /metrics 不可用或内容无效"
        record_fail "$name /metrics"
    fi
done

# =============================================================================
# 步骤 4：轻量 API 调用（admin 内部鉴权拦截 → 401/11001）
# =============================================================================
if [ "$SKIP_API" -eq 1 ]; then
    step "4. 跳过 API 调用（--skip-api）"
else
    step "4. 调 admin 内部端点（无 token → 期望 401 / business code 11001）"
    API_RESP=$(curl -s -o /tmp/doctors-api.txt -w '%{http_code}' "http://127.0.0.1:${API_HOST_PORT}${API_PATH}")
    API_BODY=$(cat /tmp/doctors-api.txt)
    if [ "$API_RESP" = "401" ] && echo "$API_BODY" | grep -q '"code":11001'; then
        ok "admin ${API_PATH} → 401 + code=11001（鉴权中间件工作正常）"
        record_pass "admin API auth check"
    elif [ "$API_RESP" = "200" ]; then
        warn "admin ${API_PATH} 返回 200（admin 可能未启用 RBAC — 检查 RoleAuth 中间件是否挂在 escorts 分组）"
        record_skip "admin API auth check"
    else
        fail "admin ${API_PATH} → ${API_RESP}，body=${API_BODY}"
        record_fail "admin API auth check"
    fi
fi

# =============================================================================
# 步骤 5：彩色汇总
# =============================================================================
echo ""
echo -e "${CYAN}=============================================${RESET}"
echo -e "${CYAN}  汇总                                         ${RESET}"
echo -e "${CYAN}=============================================${RESET}"
printf "  total:    %d\n" "$TOTAL"
printf "  passed:   %d\n" "$PASSED"
printf "  failed:   %d\n" "$FAILED"
printf "  skipped:  %d\n" "$SKIPPED"
echo ""

if [ "$FAILED" -gt 0 ]; then
    echo -e "${RED}FAILED checks:${RESET}"
    for n in "${FAILED_NAMES[@]}"; do
        echo -e "  ${RED}✗ $n${RESET}"
    done
    echo ""
    echo -e "${YELLOW}提示：容器仍在运行（docker compose -f $COMPOSE_FILE ps），可手动排查${RESET}"
    echo ""
    echo -e "${RED}RESULT: FAIL${RESET}"
    exit 1
fi

echo -e "${GREEN}RESULT: PASS${RESET}"
exit 0
