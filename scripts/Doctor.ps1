#Requires -Version 7.0
[CmdletBinding()]
param()
$ErrorActionPreference = 'Stop'
foreach ($name in @('git', 'go', 'node', 'npm', 'docker', 'codex')) {
    $command = Get-Command $name -ErrorAction Stop
    Write-Host "$name : $($command.Source)"
}
Push-Location (Join-Path (Split-Path $PSScriptRoot -Parent) 'services/engine')
try { go version; if ($LASTEXITCODE -ne 0) { throw 'Project Go toolchain unavailable. Allow Go automatic toolchain download or install the pinned version.' } } finally { Pop-Location }
node --version
docker info --format '{{.OSType}} {{.ServerVersion}}'
if ($LASTEXITCODE -ne 0) { throw 'Start Docker Desktop with the Linux container engine, then rerun Doctor.ps1.' }
Write-Host 'Local runtimes and Docker engine are available. Google Cloud credentials are not required for local development.'
