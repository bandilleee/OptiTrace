#Requires -Version 7.0
[CmdletBinding()]
param([Parameter(Mandatory)][string]$RateFile, [ValidateSet('staging','production')][string]$Environment = 'staging')
$ErrorActionPreference = 'Stop'
$rates = Get-Content -LiteralPath $RateFile -Raw | ConvertFrom-Json
if ($rates.region -ne 'africa-south1' -or $rates.currency -ne 'USD') { throw 'Supply Johannesburg USD rates; currency conversions must be recorded separately.' }
$items = @()
foreach ($component in $rates.components) {
    if ($null -eq $component.unit_price_usd -or $component.unit_price_usd -lt 0 -or $component.units -lt 0 -or -not $component.source -or -not $component.verified_on) { throw "Missing verified price/provenance for $($component.name)." }
    if (([datetime]::UtcNow - [datetime]$component.verified_on).TotalDays -gt 30) { throw "Rate older than 30 days: $($component.name). Refresh before release." }
    if ($component.environment -notin @('all', $Environment)) { continue }
    $items += [pscustomobject]@{ name=$component.name; units=$component.units; unit=$component.unit; unit_price_usd=$component.unit_price_usd; estimated_usd=[math]::Round($component.units * $component.unit_price_usd, 4); source=$component.source; verified_on=$component.verified_on }
}
if ($items.Count -lt 6) { throw 'Include SQL compute/storage/backup, Cloud Run, Identity Platform, and reports/logs/egress in the estimate.' }
[pscustomobject]@{
    environment=$Environment; region=$rates.region; currency='USD'; generated_at=[datetime]::UtcNow.ToString('O');
    monthly_estimate_usd=[math]::Round(($items | Measure-Object estimated_usd -Sum).Sum, 2); components=$items;
    limitations=@('Estimate, not observed billing.', 'Include HA SQL pricing in production and 730 hours of always-on database capacity.', 'Account for scheduler/job execution, secrets, Artifact Registry, network egress and telemetry.', 'Budget notifications do not stop spending. Free-tier/discount assumptions must be documented explicitly.')
} | ConvertTo-Json -Depth 6
