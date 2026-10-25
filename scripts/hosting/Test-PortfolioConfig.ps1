#Requires -Version 7.0
[CmdletBinding()]
param([ValidateSet('Static','Runtime')][string]$Mode = 'Static')
$ErrorActionPreference = 'Stop'
$repoRoot = Split-Path (Split-Path $PSScriptRoot -Parent) -Parent
$blueprint = Get-Content -LiteralPath (Join-Path $repoRoot 'infra/render.yaml') -Raw | ConvertFrom-Json
$services = @($blueprint.services)
if ($services.Count -ne 1 -or $blueprint.PSObject.Properties.Name -contains 'databases') { throw 'Portfolio must use one web service and an external Supabase database.' }
$service = $services[0]
if ($service.type -ne 'web' -or $service.runtime -ne 'docker' -or $service.plan -ne 'free' -or $service.autoDeployTrigger -ne 'off') { throw 'Require a free Docker web service with automatic deployment disabled.' }
if ($service.dockerCommand -ne '/app/portfolio' -or $service.dockerfilePath -ne './infra/portfolio.Dockerfile' -or $service.healthCheckPath -ne '/healthz') { throw 'Unexpected portfolio startup or health probe.' }
$privateKeys = @('APP_ORIGIN','API_PUBLIC_URL','SUPABASE_URL','SUPABASE_PUBLISHABLE_KEY','SUPABASE_SECRET_KEY','DATABASE_URL')
foreach ($key in $privateKeys) {
    $entry = @($service.envVars | Where-Object key -EQ $key)
    if ($entry.Count -ne 1 -or $entry[0].sync -ne $false -or $entry[0].PSObject.Properties.Name -contains 'value') { throw "Configure $key privately in the hosting dashboard." }
}
$dockerfile = Get-Content -LiteralPath (Join-Path $repoRoot 'infra/portfolio.Dockerfile') -Raw
if ($dockerfile -match '(?m)^ENTRYPOINT' -or $dockerfile -notmatch '(?m)^CMD \["/app/portfolio"\]') { throw 'Portfolio CMD must be directly replaceable by Render, with no inherited API entrypoint.' }
if ($Mode -eq 'Static') { Write-Output 'PASS: one free web service, manual deployment, private configuration placeholders and portfolio CMD.'; return }
function Required-Value([string]$Name) {
    $value = [Environment]::GetEnvironmentVariable($Name)
    if ([string]::IsNullOrWhiteSpace($value)) { throw "Missing $Name; values are never printed." }
    return $value
}
if ((Required-Value 'APP_ENV') -cne 'production' -or (Required-Value 'AUTH_MODE') -cne 'supabase') { throw 'Hosted portfolio requires production configuration and Supabase auth.' }
$origin = Required-Value 'APP_ORIGIN'
if ($origin -cnotmatch '^https://([a-z0-9]([a-z0-9-]*[a-z0-9])?\.)+[a-z0-9]([a-z0-9-]*[a-z0-9])?$') { throw 'APP_ORIGIN must be the fixed HTTPS production DNS origin without path, port, credentials or query.' }
if ((Required-Value 'API_PUBLIC_URL') -cne "$origin/api") { throw 'API_PUBLIC_URL must be APP_ORIGIN followed by /api.' }
$supabase = Required-Value 'SUPABASE_URL'
if ($supabase -cnotmatch '^https://([a-z0-9]+)\.supabase\.co$') { throw 'Use the approved dedicated Supabase project HTTPS URL.' }
$projectRef = $Matches[1]
if ((Required-Value 'SUPABASE_PUBLISHABLE_KEY') -cnotmatch '^sb_publishable_[A-Za-z0-9_-]{16,}$') { throw 'Use a publishable Supabase API key for this project.' }
if ((Required-Value 'SUPABASE_SECRET_KEY') -cnotmatch '^sb_secret_[A-Za-z0-9_-]{16,}$') { throw 'Use a private Supabase secret API key for this project.' }
try { $database = [uri](Required-Value 'DATABASE_URL') } catch { throw 'DATABASE_URL is not a valid connection URI; its value is suppressed.' }
if ($database.Scheme -notin @('postgres','postgresql') -or $database.Port -ne 5432 -or $database.Host -cnotmatch '^[a-z0-9-]+\.pooler\.supabase\.com$') { throw 'Use the dashboard Supavisor IPv4 session pooler on port 5432.' }
$user = [uri]::UnescapeDataString(($database.UserInfo -split ':',2)[0])
if ($user -cne "optitrace_login.$projectRef" -or ($database.UserInfo -split ':',2).Count -ne 2 -or [string]::IsNullOrEmpty(($database.UserInfo -split ':',2)[1])) { throw 'Use the dedicated optitrace_login.<project-ref> account and its own password.' }
$query = @{}
foreach ($part in $database.Query.TrimStart('?').Split('&',[StringSplitOptions]::RemoveEmptyEntries)) {
    $pair = $part.Split('=',2)
    if ($pair.Count -ne 2 -or $pair[0] -cnotmatch '^[a-z_]+$' -or $query.ContainsKey($pair[0])) { throw 'Database connection options must be unique key/value pairs.' }
    $query[$pair[0]] = [uri]::UnescapeDataString($pair[1])
}
if ($query.sslmode -cne 'verify-full') { throw 'Database TLS must validate the host certificate (sslmode=verify-full).' }
$poolSize = 0
if (-not [int]::TryParse($query.pool_max_conns,[ref]$poolSize) -or $poolSize -lt 1 -or $poolSize -gt 6) { throw 'Set pool_max_conns between 1 and 6; four is the portfolio default.' }
foreach ($name in @('MIGRATION_DATABASE_URL','SIMULATOR_URL','SIMULATOR_AUDIENCE','BENCHMARK_JOB_NAME','REPORT_BUCKET','MOCK_ISSUER','MOCK_INTERNAL_URL','NEXT_PUBLIC_SUPABASE_SECRET_KEY')) {
    if (-not [string]::IsNullOrWhiteSpace([Environment]::GetEnvironmentVariable($name))) { throw "Do not configure $name in the hosted portfolio runtime." }
}
Write-Output 'PASS: hosted portfolio variables have valid boundaries. No provider connection, secret value, deployment or account ownership was checked.'
