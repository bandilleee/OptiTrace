#Requires -Version 7.0
[CmdletBinding()]
param([switch]$Check)
$ErrorActionPreference = 'Stop'
$repoRoot = Split-Path $PSScriptRoot -Parent
Push-Location (Join-Path $repoRoot 'services/engine')
try {
    go run github.com/sqlc-dev/sqlc/cmd/sqlc@v1.30.0 generate
    if ($LASTEXITCODE -ne 0) { throw 'sqlc generation failed.' }
    if ($Check) {
        git diff --exit-code -- internal/storage/dbgen
        if ($LASTEXITCODE -ne 0) { throw 'Generated SQL access drift detected.' }
    }
} finally { Pop-Location }
