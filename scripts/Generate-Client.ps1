#Requires -Version 7.0
[CmdletBinding()]
param([switch]$Check)
$ErrorActionPreference = 'Stop'
$repoRoot = Split-Path $PSScriptRoot -Parent
Push-Location (Join-Path $repoRoot 'apps/web')
try {
    npm run api:generate
    if ($LASTEXITCODE -ne 0) { throw 'OpenAPI client generation failed.' }
    if ($Check) {
        git diff --exit-code -- src/lib/api-schema.d.ts
        if ($LASTEXITCODE -ne 0) { throw 'Generated client drift detected. Regenerate and review the client.' }
    }
} finally { Pop-Location }
