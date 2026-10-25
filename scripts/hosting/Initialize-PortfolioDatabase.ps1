#Requires -Version 7.0
[CmdletBinding(SupportsShouldProcess,ConfirmImpact='High')]
param(
    [Parameter(Mandatory)][securestring]$ApplicationPassword,
    [string]$PsqlPath = 'psql',
    [switch]$LocalTest
)
$ErrorActionPreference = 'Stop'
$raw = [Environment]::GetEnvironmentVariable('MIGRATION_DATABASE_URL')
if ([string]::IsNullOrWhiteSpace($raw)) { throw 'Set MIGRATION_DATABASE_URL privately for the approved migration owner.' }
try { $connection = [uri]$raw } catch { throw 'Invalid migration connection URI; its value is suppressed.' }
if ($connection.Scheme -notin @('postgres','postgresql') -or $connection.UserInfo.Split(':',2).Count -ne 2) { throw 'Supply a PostgreSQL owner connection URI; it is never printed.' }
$owner = [uri]::UnescapeDataString($connection.UserInfo.Split(':',2)[0])
$password = [uri]::UnescapeDataString($connection.UserInfo.Split(':',2)[1])
if ($owner -eq 'optitrace_login' -or $owner.StartsWith('optitrace_login.')) { throw 'Application and migration principals must differ.' }
$query = @{}
foreach ($part in $connection.Query.TrimStart('?').Split('&',[StringSplitOptions]::RemoveEmptyEntries)) {
    $pair=$part.Split('=',2)
    if ($pair.Count -ne 2 -or $query.ContainsKey($pair[0])) { throw 'Invalid or duplicate migration connection options.' }
    $query[$pair[0]]=[uri]::UnescapeDataString($pair[1])
}
if ($LocalTest) {
    if ($connection.Host -notin @('localhost','127.0.0.1') -or $connection.Port -ne 55435 -or $query.sslmode -ne 'disable') { throw 'LocalTest is restricted to the isolated localhost:55435 bootstrap database.' }
} elseif ($connection.Host -cnotmatch '^[a-z0-9-]+\.pooler\.supabase\.com$' -or $connection.Port -ne 5432 -or $owner -cnotmatch '^postgres\.[a-z0-9]+$' -or $query.sslmode -ne 'verify-full') {
    throw 'Hosted bootstrap requires the approved Supabase session pooler owner on port 5432 with verified TLS.'
}
$applicationSecret = ConvertFrom-SecureString -SecureString $ApplicationPassword -AsPlainText
if ($applicationSecret -cnotmatch '^[A-Za-z0-9_-]{32,128}$') { throw 'Use a unique 32-128 character random URL-safe application password; no whitespace or punctuation outside _ and -.' }
if (-not $PSCmdlet.ShouldProcess('the explicitly configured database','Create fresh non-bypass optitrace_login; no migrations, seed or provider schema changes')) { return }
$repoRoot=Split-Path (Split-Path $PSScriptRoot -Parent) -Parent
$sql = Get-Content -LiteralPath (Join-Path $repoRoot 'infra/supabase/bootstrap.sql') -Raw
# The constrained URL-safe value cannot introduce a psql command; SQL consumes a quoted variable literal.
$inputSql = "\set app_password $applicationSecret`n$sql"
$start = [Diagnostics.ProcessStartInfo]::new()
$start.FileName=$PsqlPath
$start.UseShellExecute=$false
$start.CreateNoWindow=$true
$start.RedirectStandardInput=$true
$start.RedirectStandardOutput=$true
$start.RedirectStandardError=$true
foreach ($argument in @('-X','--quiet','--set=ON_ERROR_STOP=on','--no-password')) { $start.ArgumentList.Add($argument) }
$start.Environment['PGHOST']=$connection.Host
$start.Environment['PGPORT']=$connection.Port.ToString()
$start.Environment['PGDATABASE']=[uri]::UnescapeDataString($connection.AbsolutePath.TrimStart('/'))
$start.Environment['PGUSER']=$owner
$start.Environment['PGPASSWORD']=$password
$start.Environment['PGSSLMODE']=$query.sslmode
$start.Environment['PGCONNECT_TIMEOUT']='15'
$process=[Diagnostics.Process]::Start($start)
try {
    $output=$process.StandardOutput.ReadToEndAsync()
    $errors=$process.StandardError.ReadToEndAsync()
    $process.StandardInput.Write($inputSql)
    $process.StandardInput.Close()
    if (-not $process.WaitForExit(30000)) { $process.Kill($true); throw 'Database bootstrap timed out; inspect role existence before retrying.' }
    [void]$output.GetAwaiter().GetResult()
    [void]$errors.GetAwaiter().GetResult()
    if ($process.ExitCode -ne 0) { throw "Database bootstrap failed (exit $($process.ExitCode)). Server output is suppressed to protect secrets; verify credentials, owner permissions and that optitrace_login does not already exist." }
    Write-Output 'Fresh optitrace_login created without privileged attributes or memberships. Run reviewed migrations separately with the owner; retain only the application URL in Render.'
} finally {
    $process.Dispose()
    $applicationSecret=$null; $password=$null; $inputSql=$null; $start.Environment.Remove('PGPASSWORD') | Out-Null
}
