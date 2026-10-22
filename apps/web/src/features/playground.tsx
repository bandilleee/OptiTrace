'use client';
import { type Props } from '@/lib/workspace-types';
import { useItems } from '@/lib/use-items';
import { type Policy, type Evaluation, label, scenarios, strategies, can, cost } from '@/lib/api';
import { createEvaluation } from '@/lib/sdk';
import { useState } from 'react';
import { useQueryClient, useMutation } from '@tanstack/react-query';
import { Panel, ErrorMessage, Badge } from '@/components/ui';
import { Button } from '@/components/ui/button';
import { SelectField } from '@/components/select-field';
import { Input } from '@/components/ui/input';
import { RadioGroup, RadioGroupItem } from '@/components/ui/radio-group';
import { Skeleton } from '@/components/ui/skeleton';
import {
  ShieldCheck,
  Play,
  Layers,
  LockKeyhole,
  RotateCcw,
  Download,
  Phone,
  ScanLine,
  Smartphone,
  ArrowUpRight,
} from 'lucide-react';
import { Trace, TraceRail, exportTrace } from './traces';
import { BrandLoader } from '@/components/brand';

const scenarioHints: Record<string, string> = {
  clear: 'Number matches, with no recent SIM or device change.',
  sim_change: 'A SIM change occurred within the 24-hour policy window.',
  device_change: 'A device change occurred within the 24-hour policy window.',
  number_mismatch: 'The synthetic number does not match.',
  stale: 'Evidence exceeds the 60-second freshness limit.',
  timeout: 'A source call exceeds its timeout.',
  rate_limit: 'A source responds with a rate limit.',
  permission_denied: 'The source refuses the requested capability.',
  source_failure: 'A source returns an error.',
  all_adverse: 'All three capabilities contain adverse observations.',
  correlated: 'Joint SIM and device observations are replayed together.',
};
const strategyHints: Record<string, string> = {
  optitrace: 'Choose a valid plan. Stop optional work only when the outcome is established.',
  full_sequential: 'Fetch all inputs one at a time, including optional work.',
  full_parallel: 'Start all checks together. Every started billable call is charged.',
  staged: 'A competent staged workflow with safe conditional stopping.',
  opa: 'Staged fetching with the generated OPA policy as its decision engine.',
};
export function Playground({
  tenant,
  identity,
  onOpenTrace,
}: Props & { onOpenTrace: (id: string) => void }) {
  const policies = useItems<Policy>(tenant.id, 'policies');
  const history = useItems<Evaluation>(tenant.id, 'evaluations');
  const [policy, setPolicy] = useState('');
  const [scenario, setScenario] = useState<(typeof scenarios)[number]>('sim_change');
  const [strategy, setStrategy] = useState<(typeof strategies)[number]>('optitrace');
  const [search, setSearch] = useState('');
  const [previous, setPrevious] = useState<Evaluation>();
  const client = useQueryClient();
  const eligible = policies.data?.items.filter((p) => p.status === 'active') ?? [];
  const selected =
    eligible.find((p) => p.id === policy) ??
    eligible.find((p) => p.id === 'policy-conditional') ??
    eligible.find((p) => p.profile === 'conditional') ??
    eligible[0];
  const evaluation = useMutation({
    mutationFn: () =>
      createEvaluation({
        path: { tenant: tenant.id },
        csrf: identity.csrf_token,
        idempotency: crypto.randomUUID(),
        body: { policy_id: selected?.id ?? '', scenario, strategy },
      }),
    onSuccess: () => {
      setPrevious(undefined);
      client.invalidateQueries({ queryKey: [tenant.id, 'evaluations'] });
    },
  });
  const result = previous ?? evaluation.data;
  const recent = (history.data?.items ?? [])
    .filter((r) =>
      (r.scenario + ' ' + r.outcome + ' ' + r.id).toLowerCase().includes(search.toLowerCase()),
    )
    .slice(0, 6);
  return (
    <div className="playground-layout">
      <div className="main-column">
        <Panel
          title="Configure a verification"
          aside={
            <Button
              variant="ghost"
              size="sm"
              aria-label="Reset playground"
              onClick={() => {
                setPolicy('');
                setScenario('sim_change');
                setStrategy('optitrace');
                setPrevious(undefined);
                evaluation.reset();
              }}
            >
              <RotateCcw size={14} />
              <span>Reset</span>
            </Button>
          }
        >
          <form
            onSubmit={(e) => {
              e.preventDefault();
              setPrevious(undefined);
              evaluation.mutate();
            }}
          >
            <div className="form-body">
              <fieldset>
                <legend>Policy profile</legend>
                {policies.isPending ? (
                  <div className="policy-options">
                    <Skeleton className="h-24" />
                    <Skeleton className="h-24" />
                  </div>
                ) : (
                  <RadioGroup
                    className="policy-options"
                    value={selected?.profile ?? ''}
                    onValueChange={(value) => {
                      const next = eligible.find((p) => p.profile === value);
                      if (next) setPolicy(next.id);
                    }}
                  >
                    {['conditional', 'complete'].map((profile) => (
                      <label
                        key={profile}
                        className="policy-choice"
                        data-selected={selected?.profile === profile}
                      >
                        <RadioGroupItem
                          value={profile}
                          disabled={!eligible.some((p) => p.profile === profile)}
                          aria-label={label(profile) + ' verification'}
                        />
                        <strong>{label(profile)} verification</strong>
                        <small>
                          {profile === 'conditional'
                            ? 'Number Verification is compulsory. Stop optional checks on sufficient evidence.'
                            : 'All three checks are compulsory. Every check completes or follows its failure path.'}
                        </small>
                      </label>
                    ))}
                  </RadioGroup>
                )}
              </fieldset>
              <label className="field-label">
                Approved policy
                <SelectField
                  aria-label="Approved policy"
                  value={selected?.id ?? ''}
                  onValueChange={(value) => setPolicy(value)}
                >
                  {eligible.map((p) => (
                    <option key={p.id} value={p.id}>
                      {p.name === p.profile ? `${label(p.profile)} verification` : p.name} · v
                      {p.version}
                    </option>
                  ))}
                </SelectField>
              </label>
              <div className="two-col">
                <label className="field-label">
                  Observation scenario
                  <SelectField
                    aria-label="Observation scenario"
                    value={scenario}
                    onValueChange={(value) => setScenario(value as typeof scenario)}
                  >
                    {scenarios.map((s) => (
                      <option key={s} value={s}>
                        {label(s)}
                      </option>
                    ))}
                  </SelectField>
                  <small>{scenarioHints[scenario]}</small>
                </label>
                <label className="field-label">
                  Execution strategy
                  <SelectField
                    aria-label="Execution strategy"
                    value={strategy}
                    onValueChange={(value) => setStrategy(value as typeof strategy)}
                  >
                    {strategies.map((s) => (
                      <option key={s} value={s}>
                        {label(s)}
                      </option>
                    ))}
                  </SelectField>
                  <small>{strategyHints[strategy]}</small>
                </label>
              </div>
              <div className="policy-version">
                <LockKeyhole size={14} />
                {selected
                  ? `Approved policy v${selected.version} · read-only in this playground`
                  : 'An active approved policy is required.'}
              </div>
              <ErrorMessage error={policies.error ?? evaluation.error} />
            </div>
            <div className="run-footer">
              <span>No real number or device data is used.</span>
              <Button
                type="submit"
                disabled={
                  evaluation.isPending || !selected || !can(tenant.roles, 'integration_developer')
                }
              >
                <Play size={15} />
                {evaluation.isPending ? 'Running verification…' : 'Run verification'}
              </Button>
            </div>
            {!can(tenant.roles, 'integration_developer') && (
              <p className="permission-note">
                An integration developer role is required to run evaluations.
              </p>
            )}
          </form>
        </Panel>
        <Panel
          title="Execution result"
          className="result-panel"
          aside={
            result && !evaluation.isPending ? (
              <>
                <Button size="sm" variant="link" onClick={() => onOpenTrace(result.id)}>
                  Open trace
                  <ArrowUpRight size={13} />
                </Button>
                <Button variant="ghost" size="sm" onClick={() => exportTrace(result)}>
                  <Download size={14} />
                  Export
                </Button>
              </>
            ) : (
              <span className="subtle">On request</span>
            )
          }
        >
          {result &&
            !evaluation.isPending &&
            (result.policy_id !== selected?.id ||
              result.scenario !== scenario ||
              result.strategy !== strategy) && (
              <div className="pending-inputs" role="status">
                Showing a recorded result. Run verification to evaluate the selected inputs.
              </div>
            )}
          {evaluation.isPending ? (
            <BrandLoader label="Fetching synthetic evidence and recording the decision…" />
          ) : result ? (
            <Trace result={result} />
          ) : (
            <div className="trace-placeholder">
              <TraceRail />
              <h3>See the path to a decision</h3>
              <p>
                Run a verification to inspect the fetched evidence, skipped work, configured
                charges, and stopping reason.
              </p>
            </div>
          )}
        </Panel>
      </div>
      <aside className="context-column" aria-label="Execution context">
        <Panel title="Selected policy" aside={<Layers size={16} />}>
          <div className="context-body">
            {selected ? (
              <>
                <Badge value={selected.status} />
                <h3 className="policy-name">
                  {selected.name === selected.profile
                    ? `${label(selected.profile)} verification`
                    : selected.name}
                </h3>
                <p className="subtle">Version {selected.version} · immutable</p>
                <div className="capability-list">
                  {[
                    { icon: Phone, name: 'Number Verification' },
                    { icon: ScanLine, name: 'SIM Swap' },
                    { icon: Smartphone, name: 'Device Swap' },
                  ].map((c, i) => (
                    <div key={c.name}>
                      <span>
                        <c.icon size={14} />
                        {c.name}
                      </span>
                      <small>
                        {i === 0 || selected.profile === 'complete' ? 'Required' : 'Optional'}
                      </small>
                    </div>
                  ))}
                </div>
                <p className="context-rule">
                  {selected.profile === 'complete'
                    ? 'All three checks must complete or follow their explicit failure path.'
                    : 'Optional checks stop only when sufficient evidence establishes the outcome.'}
                </p>
                <p className="context-approval">
                  <ShieldCheck size={13} />
                  Independent approval required
                </p>
              </>
            ) : (
              <p className="subtle">
                No active policy is available. Ask an author and reviewer to publish one.
              </p>
            )}
          </div>
        </Panel>
        <Panel title="Configured charges">
          <div className="context-body">
            {result ? (
              <div className="cost-list">
                {result.checks.map((c) => (
                  <div key={c.capability}>
                    <span>{label(c.capability)}</span>
                    <strong>
                      {c.status === 'skipped' ? 'Not started' : cost(c.cost_microrand)}
                    </strong>
                  </div>
                ))}
              </div>
            ) : (
              <p className="subtle">
                Run a verification to inspect the engine's charge for each source call.
              </p>
            )}
            <p className="context-rule">
              Synthetic ZAR charges from the selected trace. Every started billable call is charged.
              Prices cannot be edited here.
            </p>
          </div>
        </Panel>
        <Panel
          title="Recent evaluations"
          aside={<span className="subtle">{history.data?.items.length ?? 0}</span>}
        >
          <div className="history-body">
            <Input
              aria-label="Search recent evaluations"
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              placeholder="Search scenario or outcome…"
            />
            <ErrorMessage error={history.error} />
            {recent.map((r) => (
              <Button
                key={r.id}
                variant="ghost"
                className="history-item"
                onClick={() => {
                  setPrevious(r);
                  evaluation.reset();
                }}
              >
                <span>
                  <strong>{label(r.scenario)}</strong>
                  <small>{label(r.outcome)}</small>
                  <span className="history-meta">
                    <span>
                      {r.id.slice(0, 8)} · {label(r.strategy)}
                    </span>
                    <span>{cost(r.supplier_cost_microrand)}</span>
                  </span>
                </span>
                <ChevronIcon />
              </Button>
            ))}
            {!history.isPending && !recent.length && (
              <p className="panel-copy">
                No matching evaluations. Run a verification to record its evidence.
              </p>
            )}
          </div>
        </Panel>
      </aside>
    </div>
  );
}
function ChevronIcon() {
  return <ArrowUpRight size={13} />;
}
