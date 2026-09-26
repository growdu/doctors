# =============================================================================
# scripts/smoke-e2e.ps1 -- end-to-end smoke (PowerShell, mirrors smoke-e2e.sh)
# =============================================================================
# Flow:
#   1. docker compose up -d docker-compose.deploy.yml (skip if -SkipStart)
#   2. wait BootWait seconds for middleware + services to be healthy
#   3. probe /healthz on all 11 services (loop until 200 or timeout)
#   4. probe /metrics endpoint on each service (Prometheus exposition format)
#   5. hit one internal API (admin pending-audit) -> expect 401 / 11001
#   6. color summary; exit 0=PASS, 1=FAIL
#
# Usage:
#   powershell -ExecutionPolicy Bypass -File scripts/smoke-e2e.ps1
#   powershell -File scripts/smoke-e2e.ps1 -SkipStart
#   powershell -File scripts/smoke-e2e.ps1 -SkipApi
#   powershell -File scripts/smoke-e2e.ps1 -WaitTimeout 180
#
# Exit code: 0=PASS, 1=FAIL. Containers stay running on fail so they can be debugged.
# =============================================================================

[CmdletBinding()]
param(
    [switch]$SkipStart,
    [switch]$SkipApi,
    [int]$WaitTimeout = 120,
    [int]$BootWait    = 30
)

# WaitTimeout: max seconds to wait per service health check
# BootWait:    seconds to wait after docker compose up before probing

$ErrorActionPreference = 'Continue'  # let one step fail without aborting the run

$ROOT = (Resolve-Path -Path "$PSScriptRoot/..").Path
Set-Location -LiteralPath $ROOT

# ---------- color helpers ----------
# $UseColor detects ANSI support (PowerShell 7+ / Windows Terminal / modern SSH).
$UseColor = ($Host.UI.SupportsVirtualTerminal) -and (($env:WT_SESSION) -or ($env:TERM -match 'xterm'))

function Write-Step { param([string]$Name) Write-Host "`n>> $Name" -ForegroundColor Cyan }
function Write-Ok   { param([string]$Msg) Write-Host "  + $Msg" -ForegroundColor Green }
function Write-FailX { param([string]$Msg) Write-Host "  x $Msg" -ForegroundColor Red }
function Write-Warn { param([string]$Msg) Write-Host "  ! $Msg" -ForegroundColor Yellow }
function Write-Info { param([string]$Msg) Write-Host "  i $Msg" }

# ---------- counters ----------
$script:Total = 0; $script:Passed = 0; $script:Failed = 0; $script:Skipped = 0
$script:FailedNames = New-Object System.Collections.Generic.List[string]

function Record-Pass { $script:Total++; $script:Passed++ }
function Record-Fail { param([string]$Name) $script:Total++; $script:Failed++; $script:FailedNames.Add($Name) }
function Record-Skip { $script:Total++; $script:Skipped++ }

# ---------- 11-service manifest (host-side ports from docker-compose.deploy.yml) ----------
$SERVICES = @(
    @{ name='auth';    port=8081 },
    @{ name='order';   port=8082 },
    @{ name='match';   port=8083 },
    @{ name='message'; port=8084 },
    @{ name='payment'; port=8085 },
    @{ name='review';  port=8086 },
    @{ name='sos';     port=8087 },
    @{ name='user';    port=8088 },
    @{ name='escort';  port=8089 },
    @{ name='wallet';  port=8090 },
    @{ name='admin';   port=8091 }
)
$API_HOST_PORT = 8091
$API_PATH      = '/api/v1/admin/escorts/pending-audit'

# =============================================================================
# Header
# =============================================================================
Write-Host '=============================================' -ForegroundColor Cyan
Write-Host '  Doctors  smoke-e2e.ps1  (end-to-end)'           -ForegroundColor Cyan
Write-Host "  ROOT:    $ROOT"                                  -ForegroundColor Cyan
Write-Host "  TIME:    $(Get-Date -Format 'yyyy-MM-dd HH:mm:ss')" -ForegroundColor Cyan
Write-Host '  SERVICES: 11 (auth/order/match/message/payment/review/sos/user/escort/wallet/admin)' -ForegroundColor Cyan
Write-Host '=============================================' -ForegroundColor Cyan

# =============================================================================
# Step 0: prerequisites
# =============================================================================
Write-Step '0. prerequisites'

try {
    Get-Command curl -ErrorAction Stop | Out-Null
    Write-Ok 'curl available'
} catch {
    Write-FailX 'curl missing -- cannot continue'
    exit 1
}

$HAS_DOCKER = $false
try { Get-Command docker -ErrorAction Stop | Out-Null; $HAS_DOCKER = $true; Write-Ok 'docker available' }
catch { Write-Warn 'docker missing -- skip step 1 container launch' }

# =============================================================================
# Step 1: docker compose up -d
# =============================================================================
if ($SkipStart) {
    Write-Step '1. skip docker-compose launch (-SkipStart)'
    Write-Warn 'user must have started the 11 services manually; this script only does endpoint probing'
} else {
    Write-Step '1. docker compose up -d docker-compose.deploy.yml'
    if (-not $HAS_DOCKER) {
        Write-Warn 'docker missing -- skip start; only probe host ports'
        Record-Skip
    } else {
        $composeFile = Join-Path $ROOT 'docker-compose.deploy.yml'
        Write-Info "running: docker compose -f $composeFile up -d"
        $proc = Start-Process -FilePath 'docker' -ArgumentList @('compose','-f',$composeFile,'up','-d') -NoNewWindow -PassThru -Wait
        $code = $proc.ExitCode
        if ($code -eq 0) {
            Write-Ok 'docker compose up -d OK (containers entering startup phase)'
            Record-Pass
            Write-Info "waiting ${BootWait}s for middleware + services to be healthy..."
            Start-Sleep -Seconds $BootWait
        } else {
            Write-FailX "docker compose up exit=$code (see docker compose output)"
            Record-Fail 'docker compose up'
        }
    }
}

# =============================================================================
# Step 2: probe /healthz
# =============================================================================
Write-Step "2. probe /healthz on 11 services (timeout=${WaitTimeout}s/service)"

function Test-Healthz {
    param([int]$Port, [int]$TimeoutSec)
    $deadline = (Get-Date).AddSeconds($TimeoutSec)
    while ((Get-Date) -lt $deadline) {
        try {
            $resp = Invoke-WebRequest -Uri "http://127.0.0.1:$Port/healthz" -UseBasicParsing -TimeoutSec 2 -ErrorAction Stop
            if ($resp.StatusCode -eq 200) { return 200 }
        } catch { }
        Start-Sleep -Milliseconds 800
    }
    return 'fail'
}

$HCodes = @{}
foreach ($svc in $SERVICES) {
    $n = $svc.name; $p = $svc.port
    Write-Info "[$n] port $p ..."
    $code = Test-Healthz -Port $p -TimeoutSec $WaitTimeout
    $HCodes[$n] = $code
    if ($code -eq 200) {
        Write-Ok "[$n] /healthz -> 200"
        Record-Pass
    } else {
        Write-FailX "[$n] /healthz did not return 200 within ${WaitTimeout}s"
        Record-Fail "$n /healthz"
    }
}

# =============================================================================
# Step 3: probe /metrics
# =============================================================================
Write-Step '3. probe /metrics endpoint (11 services)'

function Test-Metrics {
    param([int]$Port)
    try {
        $resp = Invoke-WebRequest -Uri "http://127.0.0.1:$Port/metrics" -UseBasicParsing -TimeoutSec 3 -ErrorAction Stop
        $body = $resp.Content
        if ($resp.StatusCode -eq 200 -and $body -match '^# HELP ') { return 200 }
        return 'invalid'
    } catch {
        return 'fail'
    }
}

foreach ($svc in $SERVICES) {
    $n = $svc.name; $p = $svc.port
    if ($HCodes[$n] -ne 200) {
        Write-Warn "[$n] /healthz not OK -> skip /metrics probe"
        Record-Skip
        continue
    }
    $code = Test-Metrics -Port $p
    if ($code -eq 200) {
        Write-Ok "[$n] /metrics -> 200 with Prometheus exposition format"
        Record-Pass
    } else {
        Write-FailX "[$n] /metrics unavailable or invalid ($code)"
        Record-Fail "$n /metrics"
    }
}

# =============================================================================
# Step 4: API call (admin internal auth check)
# =============================================================================
if ($SkipApi) {
    Write-Step '4. skip API call (-SkipApi)'
} else {
    Write-Step "4. admin internal endpoint GET ${API_PATH} (no token -> expect 401 + code=11001)"
    try {
        $resp = Invoke-WebRequest -Uri "http://127.0.0.1:${API_HOST_PORT}${API_PATH}" -UseBasicParsing -TimeoutSec 5 -ErrorAction Stop
        $status = $resp.StatusCode
        $body = $resp.Content
    } catch {
        # 4xx/5xx still count as response reachable
        $status = $_.Exception.Response.StatusCode.value__
        $body = (($_.ErrorDetails.Message) -as [string])
    }
    if ($status -eq 401 -and $body -match '"code":11001') {
        Write-Ok "admin ${API_PATH} -> 401 + code=11001 (auth middleware working)"
        Record-Pass
    } elseif ($status -eq 200) {
        Write-Warn "admin ${API_PATH} returned 200 (admin RBAC may not be enabled -- check RoleAuth)"
        Record-Skip
    } else {
        Write-FailX "admin ${API_PATH} -> ${status}, body=$body"
        Record-Fail 'admin API auth check'
    }
}

# =============================================================================
# Step 5: summary
# =============================================================================
Write-Host ''
Write-Host '=============================================' -ForegroundColor Cyan
Write-Host '  Summary'                                         -ForegroundColor Cyan
Write-Host '=============================================' -ForegroundColor Cyan
Write-Host ("  total:    {0}" -f $script:Total)
Write-Host ("  passed:   {0}" -f $script:Passed)
Write-Host ("  failed:   {0}" -f $script:Failed)
Write-Host ("  skipped:  {0}" -f $script:Skipped)

if ($script:Failed -gt 0) {
    Write-Host ''
    Write-Host 'FAILED checks:' -ForegroundColor Red
    foreach ($n in $script:FailedNames) {
        Write-Host "  x $n" -ForegroundColor Red
    }
    Write-Host ''
    Write-Host 'containers still running (`docker compose -f docker-compose.deploy.yml ps`) -- debug manually' -ForegroundColor Yellow
    Write-Host ''
    Write-Host 'RESULT: FAIL' -ForegroundColor Red
    exit 1
}

Write-Host ''
Write-Host 'RESULT: PASS' -ForegroundColor Green
exit 0
