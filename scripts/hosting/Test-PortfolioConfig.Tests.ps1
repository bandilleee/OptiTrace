#Requires -Version 7.0
[CmdletBinding()]
param()
$ErrorActionPreference='Stop'
$validator=Join-Path $PSScriptRoot 'Test-PortfolioConfig.ps1'
$baseline=@{
    APP_ENV='production'; AUTH_MODE='supabase'; APP_ORIGIN='https://optitrace-demo.vercel.app'; API_PUBLIC_URL='https://optitrace-demo.vercel.app/api';
    SUPABASE_URL='https://syntheticprojectref.supabase.co'; SUPABASE_PUBLISHABLE_KEY='sb_publishable_000000000000000000000000'; SUPABASE_SECRET_KEY='sb_secret_000000000000000000000000';
    DATABASE_URL='postgresql://optitrace_login.syntheticprojectref:synthetic-password@aws-0-eu-central-1.pooler.supabase.com:5432/postgres?sslmode=verify-full&pool_max_conns=4';
    MIGRATION_DATABASE_URL=$null; SIMULATOR_URL=$null; SIMULATOR_AUDIENCE=$null; BENCHMARK_JOB_NAME=$null; REPORT_BUCKET=$null; MOCK_ISSUER=$null; MOCK_INTERNAL_URL=$null; NEXT_PUBLIC_SUPABASE_SECRET_KEY=$null
}
$saved=@{}
foreach($key in $baseline.Keys){$saved[$key]=[Environment]::GetEnvironmentVariable($key)}
function Set-Baseline { foreach($key in $baseline.Keys){[Environment]::SetEnvironmentVariable($key,$baseline[$key])} }
$negative=@(
    @{name='test provider in hosting'; values=@{AUTH_MODE='mock'}},
    @{name='origin wildcard'; values=@{APP_ORIGIN='https://*.vercel.app'}},
    @{name='origin query'; values=@{APP_ORIGIN='https://optitrace-demo.vercel.app?next=bad'}},
    @{name='callback mismatch'; values=@{API_PUBLIC_URL='https://other.vercel.app/api'}},
    @{name='secret key missing'; values=@{SUPABASE_SECRET_KEY=$null}},
    @{name='publishable key used as secret'; values=@{SUPABASE_SECRET_KEY='sb_publishable_000000000000000000000000'}},
    @{name='transaction pooler'; values=@{DATABASE_URL=$baseline.DATABASE_URL.Replace(':5432/',':6543/')}},
    @{name='database owner at runtime'; values=@{DATABASE_URL=$baseline.DATABASE_URL.Replace('optitrace_login.syntheticprojectref','postgres.syntheticprojectref')}},
    @{name='unverified database TLS'; values=@{DATABASE_URL=$baseline.DATABASE_URL.Replace('verify-full','require')}},
    @{name='duplicate database options'; values=@{DATABASE_URL=$baseline.DATABASE_URL+'&sslmode=disable'}},
    @{name='excessive connection pool'; values=@{DATABASE_URL=$baseline.DATABASE_URL.Replace('pool_max_conns=4','pool_max_conns=50')}},
    @{name='external simulator'; values=@{SIMULATOR_URL='https://other.example'}},
    @{name='migration secret in runtime'; values=@{MIGRATION_DATABASE_URL='synthetic-owner-value'}},
    @{name='secret in public bundle'; values=@{NEXT_PUBLIC_SUPABASE_SECRET_KEY='synthetic-secret'}}
)
try {
    Set-Baseline
    & $validator -Mode Static | Out-Null
    & $validator -Mode Runtime | Out-Null
    foreach($test in $negative){
        Set-Baseline
        foreach($key in $test.values.Keys){[Environment]::SetEnvironmentVariable($key,$test.values[$key])}
        $rejected=$false
        try { & $validator -Mode Runtime | Out-Null } catch { $rejected=$true }
        if(-not $rejected){throw "Unsafe configuration accepted: $($test.name)"}
    }
    Write-Output "PASS: static Blueprint, valid synthetic runtime and $($negative.Count) unsafe configuration rejections. No provider contacted."
} finally { foreach($key in $saved.Keys){[Environment]::SetEnvironmentVariable($key,$saved[$key])} }
