#Requires -Version 7.0
[CmdletBinding()]
param([string]$Path = (Join-Path (Split-Path $PSScriptRoot -Parent) '.env'))
$ErrorActionPreference = 'Stop'
foreach ($line in Get-Content -LiteralPath $Path) {
    if ($line -match '^\s*(#|$)') { continue }
    if ($line -notmatch '^([A-Z][A-Z0-9_]*)=(.*)$') { throw 'Invalid environment file line. Use literal KEY=value entries.' }
    [Environment]::SetEnvironmentVariable($Matches[1], $Matches[2], 'Process')
}
Write-Host 'Loaded local process environment. Values were treated as literal data.'
