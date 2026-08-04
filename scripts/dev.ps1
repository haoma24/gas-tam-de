# Gas Tam Đệ — local DX (Windows PowerShell; mirrors Makefile / T9.2.3)
# Usage (from repo root):
#   .\scripts\dev.ps1 help
#   .\scripts\dev.ps1 nats
#   .\scripts\dev.ps1 gateway

[CmdletBinding()]
param(
    [Parameter(Position = 0)]
    [ValidateSet(
        'help', 'nats-up', 'nats-down', 'nats-logs', 'nats-init', 'nats', 'wait-nats',
        'gateway', 'auth', 'catalog', 'geo', 'order', 'inventory', 'billing', 'report',
        'tidy', 'test', 'build', 'compose-up', 'compose-down', 'compose-ps', 'compose-logs',
        'web-up', 'web-logs', 'web-health', 'health', 'stack-health', 'check-env-yaml',
        'flutter-get', 'flutter-create', 'flutter-web', 'flutter-android', 'flutter-ios', 'flutter-devices'
    )]
    [string]$Command = 'help',

    [string]$GatewayUrl = 'http://127.0.0.1:8080',
    [string]$WebUrl = 'http://127.0.0.1:8090',
    [string]$NatsHealthUrl = 'http://127.0.0.1:8222/healthz'
)

$ErrorActionPreference = 'Stop'
$RepoRoot = Split-Path -Parent $PSScriptRoot
Set-Location $RepoRoot

function Show-Help {
    @"
Gas Tam De - scripts/dev.ps1 (Windows DX)

  Infra
    .\scripts\dev.ps1 nats-up      Start NATS JetStream (detached)
    .\scripts\dev.ps1 nats-down    Stop NATS container
    .\scripts\dev.ps1 nats-init    Bootstrap JetStream streams
    .\scripts\dev.ps1 nats         nats-up + wait + nats-init
    .\scripts\dev.ps1 nats-logs    Tail NATS logs
    .\scripts\dev.ps1 compose-up   Full stack docker compose --build (waits for healthy)
    .\scripts\dev.ps1 compose-down Stop full stack
    .\scripts\dev.ps1 compose-ps   Status of every container (incl. health)
    .\scripts\dev.ps1 compose-logs Tail logs of ALL services (not just NATS)
    .\scripts\dev.ps1 check-env-yaml  Ensure .env.example is safe for Cursor Cloud override YAML

  Website (Flutter Web + nginx in Docker, http://127.0.0.1:8090)
    .\scripts\dev.ps1 web-up       Build + start web (and its API deps)
    .\scripts\dev.ps1 web-logs     Tail nginx access/error logs
    .\scripts\dev.ps1 web-health   GET web /web-healthz + proxied /v1/hello
    .\scripts\dev.ps1 stack-health compose-ps + gateway health + web health

  Go services
    .\scripts\dev.ps1 gateway | auth | catalog | geo | order | inventory | billing | report
    .\scripts\dev.ps1 tidy | test | build | health

  Flutter (apps/mobile) — Web + Android + iOS CTA shell (T9.2.4)
    .\scripts\dev.ps1 flutter-get
    .\scripts\dev.ps1 flutter-create
    .\scripts\dev.ps1 flutter-web
    .\scripts\dev.ps1 flutter-android
    .\scripts\dev.ps1 flutter-ios
    .\scripts\dev.ps1 flutter-devices

  Equivalent Makefile targets: make <same-name>
"@ | Write-Host
}

function Invoke-Compose {
    param([Parameter(ValueFromRemainingArguments = $true)][string[]]$ComposeArgs)
    $envArgs = @()
    $envFile = Join-Path $RepoRoot 'deploy/.env'
    if (Test-Path $envFile) {
        $envArgs = @('--env-file', $envFile)
    }
    & docker compose -f deploy/docker-compose.yml @envArgs @ComposeArgs
    if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
}

function Wait-Nats {
    Write-Host "Waiting for NATS health at $NatsHealthUrl ..."
    for ($i = 0; $i -lt 30; $i++) {
        try {
            $null = Invoke-WebRequest -Uri $NatsHealthUrl -UseBasicParsing -TimeoutSec 2
            Write-Host 'NATS healthy'
            return
        } catch {
            Start-Sleep -Seconds 1
        }
    }
    Write-Error 'NATS did not become healthy in time (is Docker Desktop running?)'
}

function Invoke-GoRunService {
    param([string]$RelPath)
    go run $RelPath
    if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
}

switch ($Command) {
    'help' { Show-Help }
    'nats-up' { Invoke-Compose @('up', 'nats', '-d') }
    'nats-down' { Invoke-Compose @('stop', 'nats') }
    'nats-logs' { Invoke-Compose @('logs', '-f', 'nats') }
    'wait-nats' { Wait-Nats }
    'nats-init' {
        go run ./cmd/nats-init
        if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
    }
    'nats' {
        Invoke-Compose @('up', 'nats', '-d')
        Wait-Nats
        go run ./cmd/nats-init
        if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
    }
    'compose-up' {
        Invoke-Compose @('up', '--build', '-d', '--wait')
        Invoke-Compose @('ps', '-a')
    }
    'compose-down' { Invoke-Compose @('down') }
    'compose-ps' { Invoke-Compose @('ps', '-a') }
    'compose-logs' { Invoke-Compose @('logs', '-f', '--tail=100') }
    'check-env-yaml' {
        & bash ./scripts/check-env-yaml-safe.sh
        if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
    }
    'web-up' {
        Invoke-Compose @('up', '--build', '-d', '--wait', 'web')
        Invoke-Compose @('ps', '-a')
    }
    'web-logs' { Invoke-Compose @('logs', '-f', '--tail=100', 'web') }
    'web-health' {
        $wz = Invoke-WebRequest -Uri "$WebUrl/web-healthz" -UseBasicParsing
        Write-Host $wz.Content
        $wHello = Invoke-WebRequest -Uri "$WebUrl/v1/hello" -UseBasicParsing
        Write-Host $wHello.Content
    }
    'stack-health' {
        Invoke-Compose @('ps', '-a')
        & $PSCommandPath 'health' -GatewayUrl $GatewayUrl
        & $PSCommandPath 'web-health' -WebUrl $WebUrl
    }
    'gateway' { Invoke-GoRunService './services/api-gateway' }
    'auth' { Invoke-GoRunService './services/auth-service' }
    'catalog' { Invoke-GoRunService './services/catalog-service' }
    'geo' { Invoke-GoRunService './services/geo-service' }
    'order' { Invoke-GoRunService './services/order-service' }
    'inventory' { Invoke-GoRunService './services/inventory-service' }
    'billing' { Invoke-GoRunService './services/billing-service' }
    'report' { Invoke-GoRunService './services/report-service' }
    'tidy' {
        go mod tidy
        if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
    }
    'test' {
        go test ./...
        if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
    }
    'build' {
        go build ./services/...
        if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
    }
    'health' {
        $hz = Invoke-WebRequest -Uri "$GatewayUrl/healthz" -UseBasicParsing
        Write-Host $hz.Content
        $hello = Invoke-WebRequest -Uri "$GatewayUrl/v1/hello" -UseBasicParsing
        Write-Host $hello.Content
    }
    'flutter-get' {
        Push-Location apps/mobile
        try {
            flutter pub get
            if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
        } finally { Pop-Location }
    }
    'flutter-create' {
        Push-Location apps/mobile
        try {
            flutter create . --project-name gas_tam_de --org vn.gastamde --platforms=web,android,ios
            if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
        } finally { Pop-Location }
    }
    'flutter-web' {
        Push-Location apps/mobile
        try {
            flutter run -d chrome
            if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
        } finally { Pop-Location }
    }
    'flutter-android' {
        Push-Location apps/mobile
        try {
            flutter run -d android
            if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
        } finally { Pop-Location }
    }
    'flutter-ios' {
        Push-Location apps/mobile
        try {
            flutter run -d ios
            if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
        } finally { Pop-Location }
    }
    'flutter-devices' {
        Push-Location apps/mobile
        try {
            flutter devices
            if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
        } finally { Pop-Location }
    }
}
