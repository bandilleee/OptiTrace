#Requires -Version 7.0
[CmdletBinding()]
param(
    [string]$BaseUrl = 'http://localhost:3000',
    [ValidateRange(10, 1000)][int]$Samples = 100,
    [string]$OutputFile = (Join-Path (Split-Path $PSScriptRoot -Parent) '.artifacts/performance/warm-api.json')
)
$ErrorActionPreference = 'Stop'
$origin = [Uri]$BaseUrl
if ($origin.Host -notin @('localhost','127.0.0.1')) { throw 'This automatic synthetic login measurement is local-only. Use a reviewed authenticated load profile for hosting.' }
$handler = [Net.Http.HttpClientHandler]::new()
$handler.AllowAutoRedirect = $false
$handler.CookieContainer = [Net.CookieContainer]::new()
$client = [Net.Http.HttpClient]::new($handler)
$client.Timeout = [TimeSpan]::FromSeconds(20)
try {
    $login = $client.GetAsync("$BaseUrl/api/v1/auth/login").GetAwaiter().GetResult()
    if ([int]$login.StatusCode -ne 302) { throw 'Could not start local identity flow.' }
    $authorize = $login.Headers.Location.ToString() + '&email=admin%40optitrace.test'
    $provider = $client.GetAsync($authorize).GetAwaiter().GetResult()
    if ([int]$provider.StatusCode -ne 302) { throw 'Synthetic identity provider did not return an authorization code.' }
    $callback = $client.GetAsync($provider.Headers.Location).GetAwaiter().GetResult()
    if ([int]$callback.StatusCode -ne 302) { throw 'Invitation-based local session creation failed.' }
    $meResponse = $client.GetAsync("$BaseUrl/api/v1/auth/me").GetAwaiter().GetResult()
    [void]$meResponse.EnsureSuccessStatusCode()
    $me = $meResponse.Content.ReadAsStringAsync().GetAwaiter().GetResult() | ConvertFrom-Json
    $tenantId = $me.tenants[0].id
    $policiesResponse = $client.GetAsync("$BaseUrl/api/v1/tenants/$tenantId/policies").GetAwaiter().GetResult()
    [void]$policiesResponse.EnsureSuccessStatusCode()
    $policies = ($policiesResponse.Content.ReadAsStringAsync().GetAwaiter().GetResult() | ConvertFrom-Json).items | Where-Object { $_.status -eq 'active' -and $_.id -in @('policy-complete','policy-conditional') }
    if (@($policies).Count -ne 2) { throw 'The local measurement requires both seeded policy profiles.' }
    $results = @()
    # First request is recorded separately as an un-warmed local request, not a cloud cold-start estimate.
    for ($i = 0; $i -le $Samples; $i++) {
        $policy = @($policies)[$i % @($policies).Count]
        $request = [Net.Http.HttpRequestMessage]::new([Net.Http.HttpMethod]::Post, "$BaseUrl/api/v1/tenants/$tenantId/evaluations")
        $request.Headers.Add('Origin', $BaseUrl)
        $request.Headers.Add('X-CSRF-Token', $me.csrf_token)
        $request.Headers.Add('Idempotency-Key', [guid]::NewGuid().ToString('N'))
        $request.Content = [Net.Http.StringContent]::new((@{policy_id=$policy.id;scenario='clear';strategy='optitrace'} | ConvertTo-Json -Compress), [Text.Encoding]::UTF8, 'application/json')
        $timer = [Diagnostics.Stopwatch]::StartNew()
        $response = $client.SendAsync($request).GetAwaiter().GetResult()
        $body = $response.Content.ReadAsStringAsync().GetAwaiter().GetResult()
        $timer.Stop()
        if (-not $response.IsSuccessStatusCode) { throw "Evaluation failed ($([int]$response.StatusCode)): $body" }
        $evaluation = $body | ConvertFrom-Json
        if ($evaluation.outcome -ne 'verified') { throw 'Unexpected synthetic reference outcome.' }
        $results += [pscustomobject]@{ sample=$i; profile=$policy.profile; elapsed_ms=$timer.Elapsed.TotalMilliseconds; outcome=$evaluation.outcome }
        $request.Dispose(); $response.Dispose()
    }
    $warm = @($results | Select-Object -Skip 1 | Sort-Object elapsed_ms)
    $p95 = $warm[[math]::Ceiling(0.95 * $warm.Count) - 1].elapsed_ms
    $report = [pscustomobject]@{
        measured_at=[datetime]::UtcNow.ToString('O'); measurement='local authenticated HTTP via Next proxy, includes auth/planning/adapters/persistence/response transfer';
        scenario='all clear, alternating active policy profiles'; samples=$Samples; first_unwarmed_request_ms=$results[0].elapsed_ms;
        p50_ms=$warm[[math]::Ceiling(0.5 * $warm.Count)-1].elapsed_ms; p95_ms=$p95; p99_ms=$warm[[math]::Ceiling(0.99 * $warm.Count)-1].elapsed_ms;
        absolute_500ms_target_passed=($p95 -le 500); baseline_regression_gate='not established by this single-strategy smoke profile';
        limitations=@('Local synthetic measurement, not cloud production SLA.', 'Synthetic Google/Identity Platform protocols; no live provider latency measured.', 'Representative workload and strongest-baseline comparison remain separate benchmark gates.'); results=$results
    }
    New-Item -ItemType Directory -Force (Split-Path $OutputFile -Parent) | Out-Null
    $report | ConvertTo-Json -Depth 6 | Set-Content -LiteralPath $OutputFile
    $report | Select-Object measurement,samples,p50_ms,p95_ms,p99_ms,absolute_500ms_target_passed | Format-List
    if ($p95 -gt 500) { throw 'Local warm p95 target was not met; see recorded report.' }
} finally { $client.Dispose(); $handler.Dispose() }
