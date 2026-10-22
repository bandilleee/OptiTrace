#Requires -Version 7.0
[CmdletBinding()]
param(
    [ValidateSet('All', 'Engine', 'Web', 'Infrastructure')][string]$Scope = 'All',
    [string]$RepoRoot = (Split-Path $PSScriptRoot -Parent),
    [switch]$RequireDatabase
)
$ErrorActionPreference = 'Stop'
$RepoRoot = (Resolve-Path -LiteralPath $RepoRoot).Path
if ($RequireDatabase -and -not $env:TEST_DATABASE_URL) { throw 'Independent acceptance requires TEST_DATABASE_URL for a disposable PostgreSQL database. Unit tests alone cannot pass this gate.' }
function Invoke-Checked([string]$Command, [string[]]$Arguments) {
    & $Command @Arguments
    if ($LASTEXITCODE -ne 0) { throw "$Command $($Arguments -join ' ') failed with exit code $LASTEXITCODE" }
}
if ($Scope -in @('All', 'Engine')) {
    Push-Location (Join-Path $RepoRoot 'services/engine')
    try {
        $unformatted = & gofmt -l .
        if ($LASTEXITCODE -ne 0 -or $unformatted) { throw "Go formatting check failed: $unformatted" }
        Invoke-Checked go @('vet', './...')
        Invoke-Checked go @('test', './...', '-count=1')
    } finally { Pop-Location }
}
if ($Scope -in @('All', 'Web')) {
    Push-Location (Join-Path $RepoRoot 'apps/web')
    try {
        if (-not (Test-Path -LiteralPath 'node_modules/.package-lock.json')) { Invoke-Checked npm @('ci') }
        Invoke-Checked npm @('run', 'format:check')
        Invoke-Checked npm @('run', 'typecheck')
        Invoke-Checked npm @('run', 'test', '--', '--run')
        Invoke-Checked npm @('run', 'build')
    } finally { Pop-Location }
}
if ($Scope -in @('All', 'Infrastructure')) {
    Invoke-Checked docker @('compose', '-f', (Join-Path $RepoRoot 'infra/compose.yaml'), 'config', '--quiet')
    if (Get-Command terraform -ErrorAction SilentlyContinue) {
        Push-Location (Join-Path $RepoRoot 'infra/terraform')
        try {
            Invoke-Checked terraform @('fmt', '-check', '-recursive')
            Invoke-Checked terraform @('init', '-backend=false', '-input=false')
            Invoke-Checked terraform @('validate')
            Invoke-Checked terraform @('test')
        } finally { Pop-Location }
    }
    Get-ChildItem (Join-Path $RepoRoot 'scripts') -Filter '*.ps1' | ForEach-Object {
        $parseErrors = $null
        [void][System.Management.Automation.Language.Parser]::ParseFile($_.FullName, [ref]$null, [ref]$parseErrors)
        if ($parseErrors) { throw "PowerShell parsing failed: $($_.Name): $parseErrors" }
    }
}
