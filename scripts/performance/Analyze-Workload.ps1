#Requires -Version 7.0
Set-StrictMode -Version Latest

function Get-WorkloadQuantile([double[]]$Values, [double]$Quantile) {
    if (-not $Values.Count) { throw 'A quantile requires observed samples.' }
    $ordered = @($Values | Sort-Object)
    return $ordered[[math]::Max(0, [math]::Ceiling($Quantile * $ordered.Count) - 1)]
}

function Test-WorkloadNumber($Value) {
    return $null -ne $Value -and $Value -is [ValueType] -and $Value -isnot [bool] -and $Value -isnot [datetime] -and [double]::IsFinite([double]$Value)
}

function Get-WorkloadAnalysis([object[]]$Rows, [int]$Repetitions) {
    $strategies = @('full_sequential', 'full_parallel', 'staged', 'opa', 'optitrace')
    $profiles = @('complete', 'conditional')
    $scenarios = @('clear', 'number_mismatch', 'sim_change', 'device_change', 'all_adverse', 'stale', 'timeout', 'rate_limit', 'permission_denied', 'source_failure', 'correlated')
    $adverse = @('number_mismatch', 'sim_change', 'device_change', 'all_adverse', 'correlated')
    $violations = [Collections.Generic.List[string]]::new()
    $seen = [Collections.Generic.HashSet[string]]::new()
    $observedPrices = @{}
    foreach ($row in $Rows) {
        $key = "$($row.repeat)|$($row.profile)|$($row.scenario)|$($row.strategy)"
        if (-not $seen.Add($key)) { $violations.Add("Duplicate sample: $key") }
        if ($row.repeat -lt 0 -or $row.repeat -ge $Repetitions -or $row.profile -notin $profiles -or $row.scenario -notin $scenarios -or $row.strategy -notin $strategies) { $violations.Add("Unexpected sample: $key"); continue }
        $expected = if ($row.scenario -eq 'clear') { 'verified' } elseif ($row.scenario -in $adverse) { 'additional_verification_required' } else { 'indeterminate' }
        if ($row.http_status -ne 201 -or $row.outcome -ne $expected) { $violations.Add("Outcome/status mismatch: $key expected=$expected") }
        if (-not (Test-WorkloadNumber $row.elapsed_ms) -or $row.elapsed_ms -lt 0) { $violations.Add("Invalid elapsed time: $key") }
        if (-not (Test-WorkloadNumber $row.supplier_cost_microrand) -or -not (Test-WorkloadNumber $row.runtime_cost_microrand)) { $violations.Add("Unknown cost: $key") }
        $checks = @($row.checks)
        $capabilities = @('number_verification', 'sim_swap', 'device_swap')
        if ($checks.Count -ne 3 -or @($checks.capability | Sort-Object -Unique).Count -ne 3) { $violations.Add("Incomplete check states: $key") }
        $charge = 0L
        foreach ($capability in $capabilities) {
            $check = @($checks | Where-Object capability -eq $capability)
            if ($check.Count -ne 1) { $violations.Add("Missing capability $capability`: $key"); continue }
            $check = $check[0]
            $required = $row.profile -eq 'complete' -or $capability -eq 'number_verification'
            if ($check.compulsory -ne $required -or ($required -and $check.status -eq 'skipped')) { $violations.Add("Compulsory obligation violation: $key/$capability") }
            if ($check.status -notin @('available', 'stale', 'timeout', 'rate_limited', 'permission_denied', 'unavailable', 'skipped')) { $violations.Add("Unknown observation state: $key/$capability") }
            if (-not (Test-WorkloadNumber $check.cost_microrand) -or $check.cost_microrand -lt 0 -or ($check.status -eq 'skipped' -and $check.cost_microrand -ne 0) -or ($check.status -ne 'skipped' -and $check.cost_microrand -le 0)) { $violations.Add("Invalid started-call charge: $key/$capability") }
            if ($check.status -ne 'skipped' -and (Test-WorkloadNumber $check.cost_microrand) -and $check.cost_microrand -gt 0) {
                if ($observedPrices.ContainsKey($capability) -and $observedPrices[$capability] -ne $check.cost_microrand) { $violations.Add("Supplier price changed during comparison: $key/$capability") }
                $observedPrices[$capability] = $check.cost_microrand
            }
            $charge += [long]$check.cost_microrand
        }
        if ($charge -ne $row.supplier_cost_microrand -or $row.runtime_cost_microrand -lt 0) { $violations.Add("Cost breakdown mismatch: $key") }
    }
    $expectedCount = $Repetitions * $profiles.Count * $scenarios.Count * $strategies.Count
    if ($Rows.Count -ne $expectedCount) { $violations.Add("Expected $expectedCount samples; received $($Rows.Count).") }
    $metrics = @()
    foreach ($profile in @('combined') + $profiles) {
        foreach ($strategy in $strategies) {
            $samples = @($Rows | Where-Object { $_.strategy -eq $strategy -and ($profile -eq 'combined' -or $_.profile -eq $profile) })
            if (-not $samples.Count) { continue }
            $metrics += [pscustomobject]@{
                profile = $profile; strategy = $strategy; count = $samples.Count
                p50_ms = Get-WorkloadQuantile $samples.elapsed_ms .5
                p95_ms = Get-WorkloadQuantile $samples.elapsed_ms .95
                p99_ms = Get-WorkloadQuantile $samples.elapsed_ms .99
                mean_supplier_microrand = ($samples.supplier_cost_microrand | Measure-Object -Average).Average
                mean_runtime_microrand = ($samples.runtime_cost_microrand | Measure-Object -Average).Average
                mean_variable_microrand = ($samples | ForEach-Object { $_.supplier_cost_microrand + $_.runtime_cost_microrand } | Measure-Object -Average).Average
            }
        }
    }
    $correct = $violations.Count -eq 0
    $comparisons = @()
    foreach ($profile in @('combined') + $profiles) {
        $group = @($metrics | Where-Object profile -eq $profile)
        $opt = @($group | Where-Object strategy -eq 'optitrace')
        # Correctness failures disqualify the entire comparison, not only an inconvenient baseline.
        $eligible = @($group | Where-Object { $correct -and $_.strategy -ne 'optitrace' -and $_.p95_ms -le 500 } | Sort-Object mean_variable_microrand, p95_ms)
        $staged = @($eligible | Where-Object strategy -in @('staged', 'opa'))
        $costFirst = if ($eligible.Count) { $eligible[0] } else { $null }
        $fastest = @($eligible | Sort-Object p95_ms, mean_variable_microrand)
        $baseline = if ($fastest.Count) { $fastest[0] } else { $null }
        $costBaseline = if ($staged.Count) { $staged[0] } else { $null }
        $absolute = $opt.Count -eq 1 -and $opt[0].p95_ms -le 500
        $relative = $null -ne $baseline -and $opt.Count -eq 1 -and $opt[0].p95_ms -le (1.1 * $baseline.p95_ms)
        $savings = if ($costBaseline -and $costBaseline.mean_variable_microrand -gt 0 -and $opt.Count -eq 1) { 1 - $opt[0].mean_variable_microrand / $costBaseline.mean_variable_microrand } else { $null }
        $comparisons += [pscustomobject]@{
            profile = $profile; correctness_passed = $correct
            latency_baseline = if ($baseline) { $baseline.strategy } else { $null }
            latency_baseline_p95_ms = if ($baseline) { $baseline.p95_ms } else { $null }
            absolute_500ms_passed = $absolute; relative_10pct_passed = $relative
            cost_first_baseline = if ($costFirst) { $costFirst.strategy } else { $null }
            relative_10pct_vs_cost_first_passed = $null -ne $costFirst -and $opt.Count -eq 1 -and $opt[0].p95_ms -le (1.1 * $costFirst.p95_ms)
            cost_baseline = if ($costBaseline) { $costBaseline.strategy } else { $null }
            synthetic_variable_cost_savings_fraction = $savings
            research_20pct_target_passed = $correct -and $absolute -and $relative -and $null -ne $savings -and $savings -ge .2
        }
    }
    return [pscustomobject]@{ correctness_passed = $correct; violations = $violations.ToArray(); observed_synthetic_prices_microrand = $observedPrices; metrics = $metrics; comparisons = $comparisons; acceptance_passed = $correct -and @($comparisons | Where-Object { -not $_.absolute_500ms_passed -or -not $_.relative_10pct_passed }).Count -eq 0 }
}
