param(
  [string]$DatabaseURL = 'postgres://postgres:acceptance-only@localhost:55433/postgres?sslmode=disable'
)
$ErrorActionPreference = 'Stop'
$database = [uri]$DatabaseURL
if ($database.Host -notin @('localhost', '127.0.0.1') -or $database.Port -ne 55433) {
  throw 'Acceptance requires the isolated localhost PostgreSQL on port 55433.'
}
$repository = [IO.Path]::GetFullPath((Join-Path $PSScriptRoot '../..'))
$artifacts = Join-Path $repository '.artifacts/acceptance-next'
New-Item -ItemType Directory -Force $artifacts | Out-Null
$overlay = @{ Replace = @{} }
$overlay.Replace[(Join-Path $repository 'services/engine/internal/server/integration_test.go')] = Join-Path $PSScriptRoot 'acceptance-next-probes.go.txt'
$overlayPath = Join-Path $artifacts 'overlay.json'
$overlay | ConvertTo-Json -Depth 4 | Set-Content -Encoding utf8 $overlayPath
$originalDatabaseURL = $env:TEST_DATABASE_URL
try {
  $env:TEST_DATABASE_URL = $DatabaseURL
  Push-Location (Join-Path $repository 'services/engine')
  try {
    go test -overlay $overlayPath -count=1 -timeout 90s -v ./internal/server -run TestIndependentAcceptanceWorkflow
    if ($LASTEXITCODE -ne 0) { throw 'Independent HTTP acceptance failed.' }
  } finally { Pop-Location }
  Get-Content (Join-Path $PSScriptRoot 'acceptance-next-storage.sql') -Raw | docker exec -i optitrace-acceptance-next psql -U postgres -d postgres
  if ($LASTEXITCODE -ne 0) { throw 'Independent PostgreSQL acceptance failed.' }
} finally {
  $env:TEST_DATABASE_URL = $originalDatabaseURL
}
