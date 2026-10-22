#Requires -Version 7.0
[CmdletBinding()]
param([ValidateSet('start','stop','restart','status')][string]$Action = 'start')
$ErrorActionPreference = 'Stop'
$repoRoot = Split-Path $PSScriptRoot -Parent
$stateFile = Join-Path $repoRoot '.artifacts/local-processes.json'
function Get-OwnedProcesses {
    if (-not (Test-Path -LiteralPath $stateFile)) { return @() }
    $records = @(Get-Content -LiteralPath $stateFile -Raw | ConvertFrom-Json)
    foreach ($record in $records) {
        $process = Get-Process -Id $record.pid -ErrorAction SilentlyContinue
        if ($process -and $process.StartTime.ToUniversalTime().Ticks -eq ([datetime]$record.start_time).ToUniversalTime().Ticks -and $process.Path -eq $record.path) { [pscustomobject]@{Record=$record;Process=$process} }
    }
}
$owned = @(Get-OwnedProcesses)
if ($Action -eq 'status') {
    $owned | ForEach-Object { [pscustomobject]@{name=$_.Record.name;pid=$_.Process.Id;started=$_.Process.StartTime} } | Format-Table
    Write-Host 'Console: http://localhost:3000'
    return
}
if ($Action -in @('stop','restart')) {
    foreach ($entry in $owned) { Stop-Process -Id $entry.Process.Id -ErrorAction Stop; [void]$entry.Process.WaitForExit(5000) }
    if ($Action -eq 'stop') { Write-Host 'Stopped verified OptiTrace native processes. PostgreSQL volume is preserved.'; return }
} elseif ($owned.Count -gt 0) { throw 'OptiTrace native processes already exist. Use -Action status or -Action restart.' }
$listeners = @(Get-NetTCPConnection -State Listen -LocalPort 3000,8080,8090 -ErrorAction SilentlyContinue)
if ($listeners.Count -gt 0) { throw 'Application ports are already in use. Stop the Docker application or other listeners before starting native services.' }
Push-Location $repoRoot
try {
    & "$PSScriptRoot/Doctor.ps1"
    New-Item -ItemType Directory -Force -Path '.artifacts/bin','.artifacts/logs' | Out-Null
    docker compose -f infra/compose.yaml up -d --wait postgres
    if ($LASTEXITCODE -ne 0) { throw 'Local PostgreSQL failed to start.' }
    $devEnvironment = @{
        APP_ENV='development'; APP_ORIGIN='http://localhost:3000'; API_PUBLIC_URL='http://localhost:3000/api'; AUTH_MODE='mock';
        DATABASE_URL='postgres://optitrace_login:local-app-only@localhost:5432/optitrace?sslmode=disable'; SIMULATOR_URL='http://localhost:8090';
        MOCK_ISSUER='http://localhost:8090'; MOCK_INTERNAL_URL='http://localhost:8090'; REPORT_BUCKET=''; BENCHMARK_JOB_NAME=''; SIMULATOR_AUDIENCE=''
    }
    Push-Location services/engine
    try {
        $previousDatabase = $env:DATABASE_URL
        $previousAppEnvironment = $env:APP_ENV
        try {
            $env:DATABASE_URL='postgres://postgres:local-postgres-only@localhost:5432/optitrace?sslmode=disable'
            $env:APP_ENV='development'
            go run ./cmd/migrate -seed
            if ($LASTEXITCODE -ne 0) { throw 'Migration/seed failed.' }
        } finally { $env:DATABASE_URL=$previousDatabase; $env:APP_ENV=$previousAppEnvironment }
        foreach ($service in @('api','simulator','worker')) {
            go build -o "../../.artifacts/bin/$service.exe" "./cmd/$service"
            if ($LASTEXITCODE -ne 0) { throw "Build failed: $service" }
        }
    } finally { Pop-Location }
    Push-Location apps/web
    try {
        npm ci
        if ($LASTEXITCODE -ne 0) { throw 'Web dependency installation failed.' }
        npm run build
        if ($LASTEXITCODE -ne 0) { throw 'Web production build failed.' }
        Copy-Item -LiteralPath '.next/static' -Destination '.next/standalone/.next/static' -Recurse -Force
    } finally { Pop-Location }
    $records = @()
    foreach ($service in @('simulator','api','worker','web')) {
        $exePath = Join-Path $repoRoot ".artifacts/bin/$service.exe"
        $arguments = @()
        $processEnvironment = $devEnvironment.Clone()
        if ($service -eq 'web') {
            $exePath = (Get-Command node -CommandType Application | Select-Object -First 1).Source
            $arguments = @('apps/web/.next/standalone/server.js')
            $processEnvironment = @{API_BASE_URL='http://localhost:8080';HOSTNAME='127.0.0.1';PORT='3000';NEXT_TELEMETRY_DISABLED='1'}
        }
        $startArguments = @{FilePath=$exePath;WorkingDirectory=$repoRoot;Environment=$processEnvironment;WindowStyle='Hidden';RedirectStandardOutput=".artifacts/logs/$service.stdout.log";RedirectStandardError=".artifacts/logs/$service.stderr.log";PassThru=$true}
        if ($arguments.Count -gt 0) { $startArguments.ArgumentList=$arguments }
        $process = Start-Process @startArguments
        $records += [pscustomobject]@{name=$service;pid=$process.Id;start_time=$process.StartTime.ToString('O');path=$exePath}
        $records | ConvertTo-Json | Set-Content -LiteralPath $stateFile
    }
    $ready = $false
    for ($attempt=0;$attempt -lt 30;$attempt++) {
        try { [void](Invoke-WebRequest 'http://localhost:3000' -TimeoutSec 2); [void](Invoke-WebRequest 'http://localhost:8080/healthz' -TimeoutSec 2); $ready=$true; break } catch { Start-Sleep -Milliseconds 500 }
    }
    if (-not $ready) { throw 'Services did not become ready. Inspect .artifacts/logs and use -Action stop before retrying.' }
    Write-Host 'OptiTrace is ready: http://localhost:3000. Choose an invited synthetic Google identity after Continue with Google.'
} finally { Pop-Location }
