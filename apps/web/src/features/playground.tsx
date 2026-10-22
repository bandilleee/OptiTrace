'use client';
import { type Props } from '../lib/workspace-types';
import { useItems } from '../lib/use-items';
import { type Policy, api, type Evaluation, label, scenarios, strategies, can } from '../lib/api';
import { useState } from 'react';
import { useQueryClient, useMutation } from '@tanstack/react-query';
import { Panel, ErrorMessage } from '../components/ui';
import { ShieldCheck, Play, GitBranch, Workflow, CheckCircle2 } from 'lucide-react';
import { Trace, TraceRail } from './traces';

export function Playground({ tenant, identity }: Props) {
  const policies = useItems<Policy>(tenant.id, 'policies');
  const [policy, setPolicy] = useState('');
  const [scenario, setScenario] = useState('sim_change');
  const [strategy, setStrategy] = useState('optitrace');
  const client = useQueryClient();
  const eligible = policies.data?.items.filter((p) => p.status === 'active') ?? [];
  const selected = policy || eligible[0]?.id || '';
  const evaluation = useMutation({
    mutationFn: () =>
      api<Evaluation>(`/tenants/${tenant.id}/evaluations`, {
        method: 'POST',
        csrf: identity.csrf_token,
        idempotency: crypto.randomUUID(),
        body: { policy_id: selected, scenario, strategy },
      }),
    onSuccess: () => client.invalidateQueries({ queryKey: [tenant.id, 'evaluations'] }),
  });
  return (
    <>
      <div className="workbench">
        <Panel
          title="Configure a verification"
          aside={<span className="subtle">Synthetic request</span>}
        >
          <form
            onSubmit={(e) => {
              e.preventDefault();
              evaluation.mutate();
            }}
            className="form-stack"
          >
            <label>
              Approved policy
              <select value={selected} onChange={(e) => setPolicy(e.target.value)}>
                {eligible.map((p) => (
                  <option value={p.id} key={p.id}>
                    {p.name} · v{p.version}
                  </option>
                ))}
              </select>
            </label>
            <label>
              Observation scenario
              <select value={scenario} onChange={(e) => setScenario(e.target.value)}>
                {scenarios.map((s) => (
                  <option value={s} key={s}>
                    {label(s)}
                  </option>
                ))}
              </select>
            </label>
            <label>
              Execution strategy
              <select value={strategy} onChange={(e) => setStrategy(e.target.value)}>
                {strategies.map((s) => (
                  <option key={s} value={s}>
                    {label(s)}
                  </option>
                ))}
              </select>
            </label>
            <div className="policy-note">
              <ShieldCheck size={19} />
              <p>
                Compulsory checks always complete or follow their explicit failure path. Unknown
                evidence never counts as clear.
              </p>
            </div>
            <ErrorMessage error={policies.error ?? evaluation.error} />
            <button
              className="button primary"
              disabled={
                evaluation.isPending || !selected || !can(tenant.roles, 'integration_developer')
              }
            >
              <Play size={16} />
              {evaluation.isPending ? 'Running verification…' : 'Run verification'}
            </button>
            {!can(tenant.roles, 'integration_developer') && (
              <p className="subtle">
                An integration developer role is required to run evaluations.
              </p>
            )}
          </form>
        </Panel>
        <Panel
          title="Execution trace"
          aside={
            <span className="live-label">
              <span />
              On request
            </span>
          }
          className="trace-panel"
        >
          {evaluation.data ? (
            <Trace result={evaluation.data} />
          ) : (
            <div className="trace-placeholder">
              <TraceRail />
              <h3>See the path to a decision</h3>
              <p>
                Run a synthetic verification to inspect fetched evidence, skipped work, latency, and
                configured cost.
              </p>
            </div>
          )}
        </Panel>
      </div>
      <div className="principle-strip">
        <div>
          <GitBranch size={20} />
          <span>
            <strong>Policy decides</strong>
            <small>Approved, immutable versions</small>
          </span>
        </div>
        <div>
          <Workflow size={20} />
          <span>
            <strong>Planner orchestrates</strong>
            <small>Only safe stopping is allowed</small>
          </span>
        </div>
        <div>
          <CheckCircle2 size={20} />
          <span>
            <strong>Evidence explains</strong>
            <small>Every started call is charged</small>
          </span>
        </div>
      </div>
    </>
  );
}
