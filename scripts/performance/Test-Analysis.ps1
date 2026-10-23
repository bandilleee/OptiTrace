#Requires -Version 7.0
$ErrorActionPreference = 'Stop'
. (Join-Path $PSScriptRoot 'Analyze-Workload.ps1')
function Assert-Workload([bool]$Condition, [string]$Message) { if (-not $Condition) { throw $Message } }
function New-TestRows {
    foreach ($profile in @('complete', 'conditional')) {
        foreach ($scenario in @('clear', 'number_mismatch', 'sim_change', 'device_change', 'all_adverse', 'stale', 'timeout', 'rate_limit', 'permission_denied', 'source_failure', 'correlated')) {
            foreach ($strategy in @('full_sequential', 'full_parallel', 'staged', 'opa', 'optitrace')) {
                $outcome = if ($scenario -eq 'clear') { 'verified' } elseif ($scenario -in @('number_mismatch', 'sim_change', 'device_change', 'all_adverse', 'correlated')) { 'additional_verification_required' } else { 'indeterminate' }
                [pscustomobject]@{ repeat=0;profile=$profile;scenario=$scenario;strategy=$strategy;http_status=201;outcome=$outcome;elapsed_ms=100;supplier_cost_microrand=6;runtime_cost_microrand=1;checks=@(
                    [pscustomobject]@{capability='number_verification';status='available';compulsory=$true;cost_microrand=1},
                    [pscustomobject]@{capability='sim_swap';status='available';compulsory=($profile -eq 'complete');cost_microrand=2},
                    [pscustomobject]@{capability='device_swap';status='available';compulsory=($profile -eq 'complete');cost_microrand=3}
                ) }
            }
        }
    }
}
Assert-Workload ((Get-WorkloadQuantile (1..100) .95) -eq 95) 'Nearest-rank p95 incorrect.'
$rows = @(New-TestRows)
$valid = Get-WorkloadAnalysis $rows 1
Assert-Workload $valid.acceptance_passed 'Valid paired fixture should pass execution acceptance.'
Assert-Workload (-not $valid.comparisons[0].research_20pct_target_passed) 'Equal synthetic costs must not imply a research saving.'
$rows[0].outcome = 'indeterminate'
Assert-Workload (-not (Get-WorkloadAnalysis $rows 1).correctness_passed) 'All-clear indeterminate must fail even if other strategies agree.'
$rows = @(New-TestRows)
$rows[0].checks[0].status = 'skipped'
Assert-Workload (-not (Get-WorkloadAnalysis $rows 1).correctness_passed) 'Skipped compulsory work must fail.'
$rows = @(New-TestRows)
$rows[0].checks[0].cost_microrand = 0
Assert-Workload (-not (Get-WorkloadAnalysis $rows 1).correctness_passed) 'An uncharged started call must fail.'
$rows = @(New-TestRows)
Assert-Workload (-not (Get-WorkloadAnalysis ($rows + $rows[0]) 1).correctness_passed) 'Duplicate rows must fail.'
Assert-Workload (-not (Get-WorkloadAnalysis $rows[1..109] 1).acceptance_passed) 'Incomplete matrix must not pass.'
$rows = @(New-TestRows)
foreach ($row in $rows) { if ($row.strategy -eq 'optitrace') { $row.elapsed_ms = 120 } }
Assert-Workload (-not (Get-WorkloadAnalysis $rows 1).acceptance_passed) 'A20% p95 regression must fail despite absolute500ms success.'
foreach ($row in $rows) { if ($row.strategy -eq 'optitrace') { $row.elapsed_ms = 501 } }
Assert-Workload (-not (Get-WorkloadAnalysis $rows 1).acceptance_passed) 'Absolute latency breach must fail.'
$rows = @(New-TestRows)
$rows[0].runtime_cost_microrand = $null
Assert-Workload (-not (Get-WorkloadAnalysis $rows 1).correctness_passed) 'Unknown runtime cost must fail, not become zero.'
Assert-Workload (-not (Get-WorkloadAnalysis @() 1).acceptance_passed) 'An empty failed run cannot pass.'
$rows = @(New-TestRows)
$rows[0].checks[0].cost_microrand = 2
$rows[0].supplier_cost_microrand = 7
Assert-Workload (-not (Get-WorkloadAnalysis $rows 1).correctness_passed) 'Changing prices between strategies invalidates a comparison.'
Write-Output 'Performance analysis positive and negative-gate checks passed.'
