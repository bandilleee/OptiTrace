#Requires -Version 7.0
[CmdletBinding()]
param([ValidateSet('up', 'down', 'logs', 'status')][string]$Action = 'up')
$ErrorActionPreference = 'Stop'
$repoRoot = Split-Path $PSScriptRoot -Parent
$composeFile = Join-Path $repoRoot 'infra/compose.yaml'
switch ($Action) {
    'up' { & "$PSScriptRoot/Doctor.ps1"; docker compose -f $composeFile up --build -d --wait }
    'down' { docker compose -f $composeFile down }
    'logs' { docker compose -f $composeFile logs --tail 100 }
    'status' { docker compose -f $composeFile ps }
}
if ($LASTEXITCODE -ne 0) { throw "Docker Compose $Action failed." }
