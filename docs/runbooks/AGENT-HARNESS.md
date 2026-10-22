# Windows agent harness

The harness runs Codex CLI tasks in isolated Git worktrees. It never merges, commits or deploys automatically. Every dependency must first be reviewed and integrated into the selected base commit. Tasks in the same wave may run concurrently only when they own disjoint paths.

Prerequisites: reviewed committed repository, clean primary checkout, Codex authenticated, Go/Node/Docker dependencies prefetched where sandbox networking is restricted. Do not grant deployment credentials to workers.

```powershell
# Inspect a task without launching an agent.
./scripts/Run-Harness.ps1 -TaskIds architect -DryRun

# After reviewing/integrating contracts, launch backend in an isolated worktree.
./scripts/Run-Harness.ps1 -TaskIds backend -IntegratedTaskIds architect

# After reviewed backend and algorithm integrations, parallelise independent UI/infrastructure work.
./scripts/Run-Harness.ps1 -TaskIds frontend,devops -IntegratedTaskIds architect,backend,algorithm -MaxParallel 2

# After reviewing/integrating both workstreams, request independent acceptance.
./scripts/Run-Harness.ps1 -TaskIds qa -IntegratedTaskIds architect,backend,algorithm,frontend,devops
```

Use the reviewed task graph for all other stages. Evidence goes into `.artifacts/harness/<run>/`; worktrees are preserved under the sibling `OptiTrace-worktrees/<run>/` directory. Inspect `results.json`, handbacks, JSONL events and independently executed acceptance logs. Passing tasks are `passed_pending_review`, never automatically accepted.

The controller validates task identity/input commit and JSON schema, checks actual Git changes against ownership, reruns its own reviewed acceptance script, enforces timeouts and permits at most two repair attempts. Changing commit history, out-of-scope writes, failed commands and invalid/missing handbacks fail the task. Ownership checks detect unintended changes after execution; they are not an OS security boundary. Codex workspace sandboxing remains the execution boundary. Do not run untrusted tasks on a host containing sensitive credentials.

Use `scripts/Test-Harness.ps1` to check the task DAG, schema rejection and script syntax. Use `scripts/Check.ps1` for canonical local checks. Package/vulnerability scans and real PostgreSQL/browser checks remain CI and release requirements; a handback is not a substitute for those checks.
