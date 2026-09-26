# scripts/run-tests.ps1 —— 一键跑全栈测试（PowerShell 版本，与 run-tests.sh 输出格式对齐）。
#
# 用法：
#   powershell -ExecutionPolicy Bypass -File scripts/run-tests.ps1
#   powershell -File scripts/run-tests.ps1 -SkipBackend
#   powershell -File scripts/run-tests.ps1 -Only integration
#
# 跨平台：
#   - Windows PowerShell 5.1+ / PowerShell Core 7+。
#   - 输出使用 ANSI 颜色，PowerShell ISE 不显示颜色（自动退化）。
#
# 退出码：
#   - 0 = 全部通过或可跳过失败
#   - 1 = 后端 / 集成 FAIL（CI 阻塞）

[CmdletBinding()]
param(
    [switch]$SkipBackend,
    [switch]$SkipFrontend,
    [switch]$SkipIntegration,
    [ValidateSet('', 'backend', 'frontend', 'integration')]
    [string]$Only = ''
)

$ErrorActionPreference = 'Continue'  # 单步失败不立即退出，让汇总判断

$ROOT = (Resolve-Path -Path "$PSScriptRoot/..").Path
Set-Location -LiteralPath $ROOT

# ---------- 颜色 ----------
# $UseColor 检测虚拟终端支持（PowerShell 7+ / Windows Terminal / 现代 SSH）。
# 注意：-match 是字符串比较操作符；用括号分组确保优先级。
$UseColor = ($Host.UI.SupportsVirtualTerminal) -and (($env:WT_SESSION) -or ($env:TERM -match 'xterm'))
function Write-Step { param([string]$Name) Write-Host "`n▶ $Name" -ForegroundColor Cyan }
function Write-Pass  { param([int]$Sec) Write-Host "  ✓ PASS  (${Sec}s)" -ForegroundColor Green }
function Write-Fail  { param([int]$Sec, [int]$Code) Write-Host "  ✗ FAIL  (${Sec}s, exit=$Code)" -ForegroundColor Red }
function Write-Skip  { param([string]$Reason) Write-Host "  - SKIP  ($Reason)" -ForegroundColor Yellow }

# ---------- 状态 ----------
$script:Total   = 0
$script:Passed  = 0
$script:Failed  = 0
$script:Skipped = 0
$script:FailedNames = New-Object System.Collections.Generic.List[string]

# run_step <name> <scriptblock>
function Run-Step {
    param([string]$Name, [scriptblock]$Cmd)
    $script:Total++
    Write-Step $Name
    $sw = [System.Diagnostics.Stopwatch]::StartNew()
    & $Cmd
    $code = $LASTEXITCODE
    if ($null -eq $code) { $code = 0 }
    $sw.Stop()
    if ($code -eq 0) {
        $script:Passed++
        Write-Pass $sw.Elapsed.Seconds
    } else {
        $script:Failed++
        $script:FailedNames.Add($Name)
        Write-Fail $sw.Elapsed.Seconds $code
    }
}

function Skip-Step {
    param([string]$Name, [string]$Reason)
    $script:Total++
    $script:Skipped++
    Write-Step $Name
    Write-Skip $Reason
}

# require <tool> <hint>  → $true / $false
function Require-Tool {
    param([string]$Tool, [string]$Hint)
    if (-not (Get-Command $Tool -ErrorAction SilentlyContinue)) {
        Write-Host "  skip: $Tool not found  ($Hint)" -ForegroundColor Yellow
        return $false
    }
    return $true
}

function Should-Run {
    param([string]$Cat)
    if ($Only -ne '') {
        return ($Only -eq $Cat)
    }
    switch ($Cat) {
        'backend'     { return -not $SkipBackend }
        'frontend'    { return -not $SkipFrontend }
        'integration' { return -not $SkipIntegration }
    }
    return $true
}

# ---------- 头部 ----------
Write-Host '=============================================' -ForegroundColor Cyan
Write-Host '  Doctors · run-tests.ps1  (全栈一键测试)'  -ForegroundColor Cyan
Write-Host "  ROOT: $ROOT"                              -ForegroundColor Cyan
Write-Host "  TIME: $(Get-Date -Format 'yyyy-MM-dd HH:mm:ss')" -ForegroundColor Cyan
Write-Host '=============================================' -ForegroundColor Cyan

# ---------- 1. 后端 ----------
if (Should-Run 'backend') {
    if (-not (Get-Command 'go' -ErrorAction SilentlyContinue)) {
        Write-Host 'ERROR: go not found; 后端测试是必选，请先安装 Go 1.24+' -ForegroundColor Red
        exit 1
    }
    Run-Step 'go test ./... (unit)' {
        & go test -race -count=1 -timeout=180s ./...
    }
} else {
    Skip-Step 'go test ./... (unit)' '-SkipBackend / -Only'
}

# ---------- 2. 前端 ----------
if (Should-Run 'frontend') {
    # 2.1 admin-web (Vitest)
    if (Require-Tool 'pnpm' '安装 pnpm 后重试') {
        Run-Step 'admin-web: pnpm install + vitest' {
            Set-Location -LiteralPath "$ROOT/frontend/admin-web"
            & pnpm install --silent
            & npx vitest run --reporter=basic
            Set-Location -LiteralPath $ROOT
        }
    } else {
        Skip-Step 'admin-web: pnpm install + vitest' 'pnpm 缺失'
    }

    # 2.2 patient-miniapp (Jest)
    if (Require-Tool 'npm' '安装 Node 20+ 后重试') {
        Run-Step 'patient-miniapp: npm install + jest' {
            Set-Location -LiteralPath "$ROOT/frontend/patient-miniapp"
            & npm install --silent
            & npm test --silent
            Set-Location -LiteralPath $ROOT
        }
    } else {
        Skip-Step 'patient-miniapp: npm install + jest' 'npm 缺失'
    }

    # 2.3 escort-app (Flutter test)
    if (Require-Tool 'flutter' '安装 Flutter 3.24+ 后重试') {
        Run-Step 'escort-app: flutter pub get + flutter test' {
            Set-Location -LiteralPath "$ROOT/frontend/escort-app"
            & flutter pub get --no-version-check
            & flutter test
            Set-Location -LiteralPath $ROOT
        }
    } else {
        Skip-Step 'escort-app: flutter pub get + flutter test' 'flutter 缺失'
    }
} else {
    Skip-Step 'frontend tests - 3 apps' '-SkipFrontend / -Only'
}

# ---------- 3. 集成 ----------
if (Should-Run 'integration') {
    if (Require-Tool 'go' 'Go 缺失') {
        Run-Step 'go test -tags=integration (./migrations + ./services)' {
            & go test -race -count=1 -tags=integration -timeout=300s ./migrations/... ./services/...
        }
    } else {
        Skip-Step 'go test -tags=integration' 'go 缺失'
    }
} else {
    Skip-Step 'integration tests' '-SkipIntegration / -Only'
}

# ---------- 汇总 ----------
Write-Host ''
Write-Host '=============================================' -ForegroundColor Cyan
Write-Host '  汇总'                                       -ForegroundColor Cyan
Write-Host '=============================================' -ForegroundColor Cyan
Write-Host ("  total:    {0}" -f $script:Total)
Write-Host ("  passed:   {0}" -f $script:Passed)
Write-Host ("  failed:   {0}" -f $script:Failed)
Write-Host ("  skipped:  {0}" -f $script:Skipped)

if ($script:Failed -gt 0) {
    Write-Host ''
    Write-Host 'FAILED steps:' -ForegroundColor Red
    foreach ($n in $script:FailedNames) {
        Write-Host "  ✗ $n" -ForegroundColor Red
    }
    Write-Host ''
    Write-Host 'RESULT: FAIL' -ForegroundColor Red
    exit 1
}

Write-Host ''
Write-Host 'RESULT: PASS' -ForegroundColor Green
exit 0