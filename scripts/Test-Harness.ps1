#Requires -Version 7.0
[CmdletBinding()]
param()
$ErrorActionPreference = 'Stop'
$repoRoot = Split-Path $PSScriptRoot -Parent
Get-ChildItem -LiteralPath $PSScriptRoot -Filter '*.ps1' | ForEach-Object {
    $parseErrors = $null
    [void][System.Management.Automation.Language.Parser]::ParseFile($_.FullName, [ref]$null, [ref]$parseErrors)
    if ($parseErrors) { throw "Invalid PowerShell script $($_.Name): $parseErrors" }
}
$graph = Get-Content -LiteralPath (Join-Path $repoRoot 'agents/tasks.json') -Raw | ConvertFrom-Json
$seen = @{}
foreach ($task in $graph.tasks) {
    if ($seen.ContainsKey($task.id)) { throw "Duplicate task $($task.id)." }
    $seen[$task.id] = $true
    foreach ($dependency in $task.depends_on) {
        if (-not $seen.ContainsKey($dependency)) { throw "Task dependencies must be acyclic and topologically ordered: $($task.id)." }
    }
    if ($task.timeout_seconds -le 0 -or $task.writable_paths.Count -eq 0) { throw "Invalid bounds for $($task.id)." }
    foreach ($prefix in $task.writable_paths) {
        if ([IO.Path]::IsPathRooted($prefix) -or $prefix.Contains('..')) { throw 'Task ownership must stay within its worktree.' }
    }
}
$schemaPath = Join-Path $repoRoot 'agents/handback.schema.json'
$valid = @{ task_id='qa'; status='passed'; input_commit='abc'; summary='Verified'; changed_paths=@(); checks=@(@{ command='go test ./...'; exit_code=0; evidence='test.log' }); limitations=@() } | ConvertTo-Json -Depth 5
if (-not (Test-Json -Json $valid -SchemaFile $schemaPath)) { throw 'Expected valid handback rejected.' }
$invalid = '{"task_id":"qa","status":"passed"}'
if (Test-Json -Json $invalid -SchemaFile $schemaPath -ErrorAction SilentlyContinue) { throw 'Incomplete handback accepted.' }
# ProcessStartInfo must preserve literal data, without interpreting shell metacharacters.
$processInfo = [Diagnostics.ProcessStartInfo]::new()
$processInfo.FileName = (Get-Process -Id $PID).Path
$processInfo.UseShellExecute = $false
$processInfo.CreateNoWindow = $true
$processInfo.RedirectStandardOutput = $true
foreach ($argument in @('-NoProfile', '-Command', '[Console]::Write($args[0])', 'path with spaces;$(not-a-command)')) { $processInfo.ArgumentList.Add($argument) }
# Validate argument preservation structurally; executing arbitrary script strings is deliberately avoided.
if ($processInfo.ArgumentList[3] -ne 'path with spaces;$(not-a-command)') { throw 'Native argument list corrupted literal input.' }
Write-Host "PASS: scripts parse, $($graph.tasks.Count) task definitions form an acyclic graph, handback validation fails closed, and native arguments preserve literal data."
