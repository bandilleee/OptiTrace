#Requires -Version 7.0
[CmdletBinding()]
param(
    [string]$LoginBaseUrl = 'http://localhost:3000',
    [string]$ApiBaseUrl = 'http://localhost:8080/v1',
    [ValidateRange(1, 50)][int]$Repetitions = 10,
    [int]$Seed = 20261023,
    [string]$OutputFile = (Join-Path (Split-Path $PSScriptRoot -Parent) '.artifacts/performance/workload.json'),
    [switch]$DryRun
)
$ErrorActionPreference = 'Stop'
Set-StrictMode -Version Latest
. (Join-Path $PSScriptRoot 'performance/Analyze-Workload.ps1')
$loginOrigin = [Uri]$LoginBaseUrl
$apiOrigin = [Uri]$ApiBaseUrl
if ($loginOrigin.Host -notin @('localhost', '127.0.0.1') -or $apiOrigin.Host -ne $loginOrigin.Host -or $loginOrigin.Scheme -ne 'http' -or $apiOrigin.Scheme -ne 'http') { throw 'Automatic synthetic sign-in is local HTTP only, using the same loopback hostname.' }
$strategies = @('full_sequential', 'full_parallel', 'staged', 'opa', 'optitrace')
$scenarios = @('clear', 'number_mismatch', 'sim_change', 'device_change', 'all_adverse', 'stale', 'timeout', 'rate_limit', 'permission_denied', 'source_failure', 'correlated')
$profiles = @('complete', 'conditional')
$workloadRandom = [Random]::new($Seed)
$schedule = [Collections.Generic.List[object]]::new()
for ($repeat = 0; $repeat -lt $Repetitions; $repeat++) {
    $cases = @($profiles | ForEach-Object { $profile = $_; $scenarios | ForEach-Object { [pscustomobject]@{ profile = $profile; scenario = $_ } } })
    for ($i = $cases.Count - 1; $i -gt 0; $i--) { $swap = $workloadRandom.Next($i + 1); $temporary = $cases[$i]; $cases[$i] = $cases[$swap]; $cases[$swap] = $temporary }
    foreach ($case in $cases) {
        $offset = $workloadRandom.Next($strategies.Count)
        for ($i = 0; $i -lt $strategies.Count; $i++) { $schedule.Add([pscustomobject]@{ repeat = $repeat; profile = $case.profile; scenario = $case.scenario; strategy = $strategies[($i + $offset) % $strategies.Count] }) }
    }
}
$manifest = [ordered]@{
    schema_version = 1; measured_at = [datetime]::UtcNow.ToString('O'); seed = $Seed; repetitions = $Repetitions
    allocation = 'Equal profile/scenario allocation; deterministic shuffled cases and rotated strategy order; synthetic demonstration workload, not market telemetry.'
    measurement = 'Authenticated Go API HTTP round trip, including authorization, planner, adapters, trace persistence and response transfer. Browser/Next proxy excluded by default.'
    login_url = $LoginBaseUrl; api_url = $ApiBaseUrl; concurrency = 1; expected_samples = $schedule.Count
    observation_semantics = 'Simulator observations are anchored separately to each evaluation observation time, with identical normalized scenario states. No cross-request reuse.'
    cost = 'Configured synthetic supplier charges plus estimated engine runtime cost; not actual cloud expenditure or observed MTN savings.'
    baseline_selection = 'Eligible baseline minimizes mean variable cost, tie-breaking on p95. Research cost comparison restricts candidates to staged and OPA staged controller.'
    limitations = @('Workstation run requires otherwise quiet host; no cloud SLA or live Google latency evidence.', 'Tail estimates are descriptive; each profile/strategy has 11 times repetitions samples.', 'First warm-up calls are recorded separately; they are not Cloud Run cold-start measurements.', '20% cost target is reported independently and may honestly fail acceptance.', 'No commercial cost claim; authentication provider status refresh uses local protocol double.')
    schedule = $schedule.ToArray()
}
if ($DryRun) { $manifest | ConvertTo-Json -Depth 8; return }
$rows = [Collections.Generic.List[object]]::new()
$warmup = [Collections.Generic.List[object]]::new()
$handler = [Net.Http.HttpClientHandler]::new()
$handler.AllowAutoRedirect = $false
$handler.CookieContainer = [Net.CookieContainer]::new()
$client = [Net.Http.HttpClient]::new($handler)
$client.Timeout = [TimeSpan]::FromSeconds(20)
function Invoke-WorkloadHttp([string]$Url, [object]$Body = $null) {
    $target = [Uri]$Url
    if ($target.Scheme -ne 'http' -or $target.Host -ne $loginOrigin.Host) { throw 'Synthetic measurement refused a redirect outside the selected loopback hostname.' }
    $message = [Net.Http.HttpRequestMessage]::new($(if ($null -eq $Body) { [Net.Http.HttpMethod]::Get } else { [Net.Http.HttpMethod]::Post }), $Url)
    if ($null -ne $Body) {
        $message.Headers.Add('Origin', $LoginBaseUrl)
        $message.Headers.Add('X-CSRF-Token', $script:workloadCSRF)
        $message.Headers.Add('Idempotency-Key', [guid]::NewGuid().ToString('N'))
        $message.Content = [Net.Http.StringContent]::new(($Body | ConvertTo-Json -Compress), [Text.Encoding]::UTF8, 'application/json')
    }
    $timer = [Diagnostics.Stopwatch]::StartNew()
    try {
        $response = $client.SendAsync($message).GetAwaiter().GetResult()
        try { $bodyText = $response.Content.ReadAsStringAsync().GetAwaiter().GetResult(); $timer.Stop(); return [pscustomobject]@{ status = [int]$response.StatusCode; location = $response.Headers.Location; text = $bodyText; elapsed_ms = $timer.Elapsed.TotalMilliseconds } } finally { $response.Dispose() }
    } finally { $message.Dispose() }
}
$runError = $null
try {
    $login = Invoke-WorkloadHttp "$LoginBaseUrl/api/v1/auth/login"
    if ($login.status -ne 302) { throw 'Could not start local sign-in.' }
    $provider = Invoke-WorkloadHttp ($login.location.ToString() + '&email=admin%40optitrace.test')
    if ($provider.status -ne 302) { throw 'Local protocol test double did not authorize.' }
    $callback = Invoke-WorkloadHttp $provider.location.ToString()
    if ($callback.status -ne 302) { throw 'Local membership session was not created.' }
    $me = Invoke-WorkloadHttp "$ApiBaseUrl/auth/me"
    if ($me.status -ne 200) { throw 'Authenticated API session unavailable.' }
    $identity = $me.text | ConvertFrom-Json
    $script:workloadCSRF = $identity.csrf_token
    $tenant = @($identity.tenants | Where-Object id -eq 'demo')
    if ($tenant.Count -ne 1) { throw 'Dedicated synthetic demo membership required.' }
    $manifest.tenant_id = $tenant[0].id
    $manifest.commit = (& git -C (Split-Path $PSScriptRoot -Parent) rev-parse HEAD).Trim()
    $manifest.host = [ordered]@{ os = [Runtime.InteropServices.RuntimeInformation]::OSDescription; architecture = [Runtime.InteropServices.RuntimeInformation]::OSArchitecture.ToString(); logical_processors = [Environment]::ProcessorCount }
    if ($IsWindows) {
        $manifest.host.processor_model = @(Get-CimInstance Win32_Processor | Select-Object -ExpandProperty Name)
        $manifest.host.physical_memory_bytes = (Get-CimInstance Win32_ComputerSystem).TotalPhysicalMemory
    }
    $containerData = & docker inspect optitrace-api-1 optitrace-simulator-1 optitrace-postgres-1 2>$null
    if ($LASTEXITCODE -eq 0) {
        $manifest.containers = @((($containerData -join "`n") | ConvertFrom-Json) | ForEach-Object { [pscustomobject]@{name=$_.Name; image_config_digest=$_.Image; image_reference=$_.Config.Image; cpu_quota=$_.HostConfig.NanoCpus; memory_limit=$_.HostConfig.Memory} })
    } else { $manifest.container_metadata = 'Not captured: standard local Compose container names unavailable.' }
    $policyResponse = Invoke-WorkloadHttp "$ApiBaseUrl/tenants/demo/policies"
    if ($policyResponse.status -ne 200) { throw 'Policy lookup failed.' }
    $policies = @((($policyResponse.text | ConvertFrom-Json).items) | Where-Object { $_.status -eq 'active' -and $_.id -in @('policy-complete', 'policy-conditional') })
    if ($policies.Count -ne 2) { throw 'Both seeded active profiles are required.' }
    foreach ($sample in @($profiles | ForEach-Object { $p = $_; $strategies | ForEach-Object { [pscustomobject]@{profile=$p;scenario='clear';strategy=$_;repeat=-1} } }) + $schedule.ToArray()) {
        $policy = @($policies | Where-Object profile -eq $sample.profile)[0]
        $answer = Invoke-WorkloadHttp "$ApiBaseUrl/tenants/demo/evaluations" @{ policy_id = $policy.id; scenario = $sample.scenario; strategy = $sample.strategy }
        if ($answer.status -ne 201) { throw "Evaluation HTTP $($answer.status); see local API logs. No failed response is treated as a successful timing sample." }
        $evaluation = $answer.text | ConvertFrom-Json
        $row = [pscustomobject]@{ repeat=$sample.repeat;profile=$sample.profile;scenario=$sample.scenario;strategy=$sample.strategy;http_status=$answer.status;elapsed_ms=$answer.elapsed_ms;engine_elapsed_ms=$evaluation.elapsed_ms;trace_id=$evaluation.id;outcome=$evaluation.outcome;stopping_reason=$evaluation.stopping_reason;supplier_cost_microrand=$evaluation.supplier_cost_microrand;runtime_cost_microrand=$evaluation.runtime_cost_microrand;checks=$evaluation.checks }
        if ($sample.repeat -eq -1) {
            $warmup.Add($row)
            if ($row.outcome -ne 'verified') { throw "All-clear warm-up did not verify; trace $($row.trace_id). Timing acceptance is blocked." }
        } else { $rows.Add($row) }
    }
} catch { $runError = $_.Exception.Message } finally { $client.Dispose(); $handler.Dispose(); Remove-Variable workloadCSRF -Scope Script -ErrorAction SilentlyContinue }
$analysis = Get-WorkloadAnalysis $rows.ToArray() $Repetitions
$report = [ordered]@{ manifest=$manifest; warmup=$warmup.ToArray(); rows=$rows.ToArray(); analysis=$analysis; execution_error=$runError }
New-Item -ItemType Directory -Force (Split-Path ([IO.Path]::GetFullPath($OutputFile)) -Parent) | Out-Null
$report | ConvertTo-Json -Depth 12 | Set-Content -LiteralPath $OutputFile
$analysis.comparisons | Format-Table
if ($runError -or -not $analysis.acceptance_passed) { throw "Workload acceptance not met. Inspect $OutputFile. $runError" }
