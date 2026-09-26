#!/usr/bin/env bash
# scripts/run-tests.sh —— 一键跑全栈测试（后端 + 3 前端 + 集成测试）。
#
# 用法：
#   bash scripts/run-tests.sh                  # 跑全部步骤
#   bash scripts/run-tests.sh --skip-backend    # 跳过后端
#   bash scripts/run-tests.sh --only=integration  # 只跑集成测试
#
# 跨平台：
#   - Linux / macOS / WSL：直接 bash 执行。
#   - Windows：推荐用 PowerShell 版 scripts/run-tests.ps1（输出格式一致）。
#
# 输出：
#   - 每个步骤带耗时（秒）+ 状态（GREEN ✓ / RED ✗ / SKIP -）。
#   - 最后一行汇总：总数 / 通过 / 失败 / 跳过。
#   - 退出码：0=全部通过或可跳过失败；1=后端 FAIL（CI 阻塞）。
#
# 依赖（缺失会 SKIP，不阻塞）：
#   - go（必选；缺失 → 脚本退出 1）
#   - pnpm（admin-web 测试；缺失 SKIP）
#   - npm（patient-miniapp 测试；缺失 SKIP）
#   - flutter（escort-app 测试；缺失 SKIP）

set -u  # 不开 -e：单步失败要 SKIP，不立即退出

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

# ---------- 颜色 / ANSI ----------
if [ -t 1 ] && command -v tput >/dev/null 2>&1 && [ "$(tput colors 2>/dev/null || echo 0)" -ge 8 ]; then
    GREEN='\033[0;32m'
    RED='\033[0;31m'
    YELLOW='\033[0;33m'
    CYAN='\033[0;36m'
    RESET='\033[0m'
else
    GREEN=''; RED=''; YELLOW=''; CYAN=''; RESET=''
fi

# ---------- 步骤执行器 ----------
# Globals: TOTAL_STEPS / PASSED / FAILED / SKIPPED / FAILED_NAMES
TOTAL_STEPS=0
PASSED=0
FAILED=0
SKIPPED=0
FAILED_NAMES=()

# run_step <name> <cmd...>
# - 退出 0 → 绿色 ✓
# - 退出非 0 → 红色 ✗ + 记录失败名
# - cmd 之前 set +e，调用方用 skip_if_missing 工具函数代替 SKIP 控制
run_step() {
    local name="$1"
    shift
    TOTAL_STEPS=$((TOTAL_STEPS + 1))
    echo ""
    echo -e "${CYAN}▶ [${TOTAL_STEPS}] ${name}${RESET}"
    local start end elapsed status
    start=$(date +%s)
    "$@"
    status=$?
    end=$(date +%s)
    elapsed=$((end - start))
    if [ "$status" -eq 0 ]; then
        PASSED=$((PASSED + 1))
        echo -e "  ${GREEN}✓ PASS${RESET}  (${elapsed}s)"
    else
        FAILED=$((FAILED + 1))
        FAILED_NAMES+=("$name")
        echo -e "  ${RED}✗ FAIL${RESET}  (${elapsed}s, exit=$status)"
    fi
}

# skip_step <name> <reason>
# 用于工具缺失或前置不满足：黄色 - 标记，不计入 FAIL。
skip_step() {
    local name="$1"
    local reason="$2"
    TOTAL_STEPS=$((TOTAL_STEPS + 1))
    SKIPPED=$((SKIPPED + 1))
    echo ""
    echo -e "${CYAN}▶ [${TOTAL_STEPS}] ${name}${RESET}"
    echo -e "  ${YELLOW}- SKIP${RESET}  (${reason})"
}

# require <tool> <install-hint>
# 检查工具是否存在，不存在 echo + return 1。
require() {
    local tool="$1"
    local hint="$2"
    if ! command -v "$tool" >/dev/null 2>&1; then
        echo "  ${YELLOW}skip: $tool not found${RESET}  ($hint)"
        return 1
    fi
    return 0
}

# ---------- 参数解析 ----------
SKIP_BACKEND=0
SKIP_FRONTEND=0
SKIP_INTEGRATION=0
ONLY=""
for arg in "$@"; do
    case "$arg" in
        --skip-backend)      SKIP_BACKEND=1 ;;
        --skip-frontend)     SKIP_FRONTEND=1 ;;
        --skip-integration)  SKIP_INTEGRATION=1 ;;
        --only=*)            ONLY="${arg#--only=}" ;;
        -h|--help)
            head -30 "$0"
            exit 0
            ;;
    esac
done

# should_run <category> → 0 = 跑，1 = 跳过
should_run() {
    local cat="$1"
    case "$ONLY" in
        "") ;;              # 无 --only，按 SKIP_* 走
        backend)     [ "$cat" = "backend" ]     && return 0; return 1 ;;
        frontend)    [ "$cat" = "frontend" ]    && return 0; return 1 ;;
        integration) [ "$cat" = "integration" ] && return 0; return 1 ;;
    esac
    case "$cat" in
        backend)     [ "$SKIP_BACKEND" -eq 0 ]     && return 0; return 1 ;;
        frontend)    [ "$SKIP_FRONTEND" -eq 0 ]    && return 0; return 1 ;;
        integration) [ "$SKIP_INTEGRATION" -eq 0 ] && return 0; return 1 ;;
    esac
}

# ---------- 头部 ----------
echo -e "${CYAN}=============================================${RESET}"
echo -e "${CYAN}  Doctors · run-tests.sh  (全栈一键测试)${RESET}"
echo -e "${CYAN}  ROOT: $ROOT${RESET}"
echo -e "${CYAN}  TIME: $(date '+%Y-%m-%d %H:%M:%S')${RESET}"
echo -e "${CYAN}=============================================${RESET}"

# ---------- 1. 后端单元测试 ----------
if should_run backend; then
    if ! command -v go >/dev/null 2>&1; then
        echo -e "${RED}ERROR: go not found; 后端测试是必选，请先安装 Go 1.24+${RESET}"
        exit 1
    fi
    run_step "go test ./... (unit)" \
        bash -c "go test -race -count=1 -timeout=180s ./..."
else
    skip_step "go test ./... (unit)" "--skip-backend / --only"
fi

# ---------- 2. 前端单元测试 ----------
if should_run frontend; then
    # 2.1 admin-web (Vitest)
    if require pnpm "安装 pnpm 后重试"; then
        run_step "admin-web: pnpm install + vitest" \
            bash -c "cd '$ROOT/frontend/admin-web' && pnpm install --silent && npx vitest run --reporter=basic"
    else
        skip_step "admin-web: pnpm install + vitest" "pnpm 缺失"
    fi

    # 2.2 patient-miniapp (Jest)
    if require npm "安装 Node 20+ 后重试"; then
        run_step "patient-miniapp: npm install + jest" \
            bash -c "cd '$ROOT/frontend/patient-miniapp' && npm install --silent && npm test --silent"
    else
        skip_step "patient-miniapp: npm install + jest" "npm 缺失"
    fi

    # 2.3 escort-app (Flutter test)
    if require flutter "安装 Flutter 3.24+ 后重试"; then
        run_step "escort-app: flutter pub get + flutter test" \
            bash -c "cd '$ROOT/frontend/escort-app' && flutter pub get --no-version-check && flutter test"
    else
        skip_step "escort-app: flutter pub get + flutter test" "flutter 缺失"
    fi
else
    skip_step "frontend tests (3 apps)" "--skip-frontend / --only"
fi

# ---------- 3. 集成测试 ----------
if should_run integration; then
    if require go "Go 缺失"; then
        run_step "go test -tags=integration (./migrations + ./services)" \
            bash -c "go test -race -count=1 -tags=integration -timeout=300s ./migrations/... ./services/..."
    else
        skip_step "go test -tags=integration" "go 缺失"
    fi
else
    skip_step "integration tests" "--skip-integration / --only"
fi

# ---------- 汇总 ----------
echo ""
echo -e "${CYAN}=============================================${RESET}"
echo -e "${CYAN}  汇总${RESET}"
echo -e "${CYAN}=============================================${RESET}"
printf "  total:    %d\n" "$TOTAL_STEPS"
printf "  passed:   %d\n" "$PASSED"
printf "  failed:   %d\n" "$FAILED"
printf "  skipped:  %d\n" "$SKIPPED"

if [ "$FAILED" -gt 0 ]; then
    echo ""
    echo -e "${RED}FAILED steps:${RESET}"
    for n in "${FAILED_NAMES[@]}"; do
        echo -e "  ${RED}✗ $n${RESET}"
    done
    echo ""
    echo -e "${RED}RESULT: FAIL${RESET}"
    exit 1
fi

echo ""
echo -e "${GREEN}RESULT: PASS${RESET}"
exit 0