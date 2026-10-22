'use client';
import { type Evaluation, label, duration, cost } from '@/lib/api';
import { Badge, ErrorMessage, Panel, Empty } from '@/components/ui';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { SelectField } from '@/components/select-field';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import {
  Table,
  TableHeader,
  TableBody,
  TableRow,
  TableHead,
  TableCell,
} from '@/components/ui/table';
import {
  Check,
  ChevronDown,
  CircleAlert,
  Minus,
  Clock,
  ShieldCheck,
  Download,
  Search,
  X,
} from 'lucide-react';
import { type Props } from '@/lib/workspace-types';
import { useItems } from '@/lib/use-items';
import { useState } from 'react';

export function exportTrace(result: Evaluation) {
  const blob = new Blob([JSON.stringify(result, null, 2)], { type: 'application/json' });
  const url = URL.createObjectURL(blob);
  const a = document.createElement('a');
  a.href = url;
  a.download = `optitrace-${result.id}.json`;
  a.click();
  setTimeout(() => URL.revokeObjectURL(url), 1000);
}
function time(value?: string) {
  return value && value !== '0001-01-01T00:00:00Z'
    ? new Date(value).toLocaleString('en-ZA', { timeZone: 'UTC' }) + ' UTC'
    : 'No source observation';
}
export function TraceRail() {
  return (
    <div className="trace-awaiting" aria-label="Three verification capabilities">
      {['Number Verification', 'SIM Swap', 'Device Swap'].map((c) => (
        <div key={c}>
          <span className="trace-dot neutral">
            <Minus size={11} />
          </span>
          <span>{c}</span>
          <small>Awaiting execution</small>
        </div>
      ))}
    </div>
  );
}
export function Trace({ result }: { result: Evaluation }) {
  const verified = result.outcome === 'verified';
  const uncertain = result.outcome === 'indeterminate';
  return (
    <div className="trace-detail">
      <div className="result-body">
        <div className="outcome-row">
          <span
            className={'outcome-icon ' + (verified ? 'good' : uncertain ? 'uncertain' : 'warning')}
          >
            {verified ? <ShieldCheck size={22} /> : <CircleAlert size={22} />}
          </span>
          <div>
            <h3>{label(result.outcome)}</h3>
            <p>{result.stopping_reason}</p>
          </div>
        </div>
        <dl className="metrics">
          <div>
            <dt>Measured elapsed</dt>
            <dd>{duration(result.elapsed_ms)}</dd>
          </div>
          <div>
            <dt>Synthetic supplier cost</dt>
            <dd>{cost(result.supplier_cost_microrand)}</dd>
          </div>
          <div>
            <dt>Calls started</dt>
            <dd>
              {result.status === 'execution_unconfirmed' ? (
                'Unconfirmed'
              ) : (
                <>
                  {result.checks.filter((c) => c.status !== 'skipped').length}
                  <small> of {result.checks.length}</small>
                </>
              )}
            </dd>
          </div>
        </dl>
      </div>
      <Tabs defaultValue="trace" className="result-tabs">
        <TabsList>
          <TabsTrigger value="trace">Execution trace</TabsTrigger>
          <TabsTrigger value="inputs">Run inputs</TabsTrigger>
        </TabsList>
        <TabsContent value="trace">
          <div className="check-list">
            {result.checks.map((c) => {
              const state =
                c.status === 'available'
                  ? typeof c.value === 'boolean'
                    ? c.value
                      ? 'adverse'
                      : 'valid'
                    : 'unknown'
                  : c.status;
              const tone =
                state === 'valid'
                  ? 'good'
                  : state === 'adverse'
                    ? 'adverse'
                    : ['stale', 'timeout'].includes(state)
                      ? 'warning'
                      : state === 'skipped'
                        ? 'neutral'
                        : 'uncertain';
              const Icon =
                state === 'valid'
                  ? Check
                  : state === 'skipped'
                    ? Minus
                    : state === 'stale'
                      ? Clock
                      : CircleAlert;
              return (
                <details className="trace-item" key={c.capability}>
                  <summary>
                    <span className={'trace-dot ' + tone}>
                      <Icon size={11} />
                    </span>
                    <div className="check-name">
                      <strong>{label(c.capability)}</strong>
                      <small>
                        {c.status === 'skipped'
                          ? 'Not started · No charge'
                          : `${c.elapsed_ms.toFixed(1)} ms · ${cost(c.cost_microrand)} · `}
                        {c.status !== 'skipped' && (
                          <span>{c.compulsory ? 'Compulsory' : 'Optional'}</span>
                        )}
                      </small>
                    </div>
                    <Badge value={state} />
                    <ChevronDown size={13} className="trace-chevron" />
                  </summary>
                  <dl className="evidence-grid">
                    <dt>Observation</dt>
                    <dd>
                      {c.status === 'available'
                        ? typeof c.value !== 'boolean'
                          ? 'Unknown observation'
                          : c.value
                            ? 'Adverse observation'
                            : 'Clear observation'
                        : label(c.status)}
                    </dd>
                    <dt>Source timestamp</dt>
                    <dd>{time(c.source_time)}</dd>
                    {c.event_time && (
                      <>
                        <dt>Change timestamp</dt>
                        <dd>{time(c.event_time)}</dd>
                      </>
                    )}
                    <dt>Started at</dt>
                    <dd>{c.status === 'skipped' ? 'Not started' : time(c.started_at)}</dd>
                    <dt>Configured charge</dt>
                    <dd>{cost(c.cost_microrand)}</dd>
                    {c.subject && (
                      <>
                        <dt>Synthetic subject</dt>
                        <dd>{c.subject}</dd>
                      </>
                    )}
                    {c.scope && (
                      <>
                        <dt>Evidence scope</dt>
                        <dd>{c.scope}</dd>
                      </>
                    )}
                  </dl>
                </details>
              );
            })}
            <p className="stopping-reason">
              <strong>Stopping reason.</strong> {result.stopping_reason}
            </p>
          </div>
        </TabsContent>
        <TabsContent value="inputs">
          <dl className="input-facts">
            <dt>Policy</dt>
            <dd>
              {label(result.profile)} · v{result.policy_version}
            </dd>
            <dt>Scenario</dt>
            <dd>{label(result.scenario)}</dd>
            <dt>Strategy</dt>
            <dd>{label(result.strategy)}</dd>
            <dt>Runtime estimate</dt>
            <dd>{cost(result.runtime_cost_microrand)}</dd>
            <dt>Freshness assumption</dt>
            <dd>60 seconds at evaluation observation time</dd>
            <dt>Change window</dt>
            <dd>24 hours</dd>
            <dt>Trace identifier</dt>
            <dd>{result.id}</dd>
          </dl>
        </TabsContent>
      </Tabs>
      {result.journal?.length ? (
        <details className="report-json">
          <summary>Durable source journal</summary>
          <p>
            A starting event records a possible call; a completed event records its observed result.
            Interrupted executions require reconciliation and are never automatically replayed.
          </p>
          <pre>{JSON.stringify(result.journal, null, 2)}</pre>
        </details>
      ) : null}
      <div className="trace-id">
        Trace {result.id} · Policy v{result.policy_version} · {label(result.strategy)}
      </div>
    </div>
  );
}
export function Traces({ tenant, initialTraceID = '' }: Props & { initialTraceID?: string }) {
  const query = useItems<Evaluation>(tenant.id, 'evaluations');
  const [selected, setSelected] = useState<Evaluation>();
  const [search, setSearch] = useState('');
  const [outcome, setOutcome] = useState('all');
  const [closed, setClosed] = useState(false);
  const current =
    selected ?? (!closed ? query.data?.items.find((r) => r.id === initialTraceID) : undefined);
  const rows = (query.data?.items ?? []).filter(
    (r) =>
      (outcome === 'all' || r.outcome === outcome) &&
      `${r.id} ${r.scenario} ${r.profile} ${r.outcome}`
        .toLowerCase()
        .includes(search.toLowerCase()),
  );
  return (
    <>
      <ErrorMessage error={query.error} />
      {query.isPending && (
        <p role="status" className="subtle">
          Loading records…
        </p>
      )}
      <Panel
        title="Recent evaluations"
        aside={<span className="subtle">{rows.length} recorded</span>}
      >
        <div className="table-toolbar">
          <label className="search-field">
            <Search size={15} />
            <Input
              aria-label="Search execution traces"
              placeholder="Search trace, scenario, or policy…"
              value={search}
              onChange={(e) => setSearch(e.target.value)}
            />
          </label>
          <label>
            <span className="sr-only">Filter by outcome</span>
            <SelectField
              aria-label="Filter by outcome"
              value={outcome}
              onValueChange={(value) => setOutcome(value)}
            >
              <option value="all">All outcomes</option>
              {['verified', 'additional_verification_required', 'indeterminate'].map((v) => (
                <option key={v} value={v}>
                  {label(v)}
                </option>
              ))}
            </SelectField>
          </label>
        </div>
        <Table>
          <TableHeader>
            <TableRow>
              {['Trace', 'Scenario', 'Policy', 'Outcome', 'Elapsed', 'Supplier cost'].map((c) => (
                <TableHead key={c}>{c}</TableHead>
              ))}
            </TableRow>
          </TableHeader>
          <TableBody>
            {rows.map((r) => (
              <TableRow key={r.id}>
                <TableCell>
                  <Button
                    variant="link"
                    className="text-button"
                    onClick={() => {
                      setSelected(r);
                      setClosed(false);
                    }}
                  >
                    {r.id.slice(0, 8)}
                  </Button>
                </TableCell>
                <TableCell>{label(r.scenario)}</TableCell>
                <TableCell>
                  {label(r.profile)} v{r.policy_version}
                </TableCell>
                <TableCell>
                  <Badge value={r.outcome} />
                </TableCell>
                <TableCell>{duration(r.elapsed_ms)}</TableCell>
                <TableCell>{cost(r.supplier_cost_microrand)}</TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
        {query.data && !rows.length && (
          <Empty
            title={search || outcome !== 'all' ? 'No matching evaluations' : 'No evaluations yet'}
            text="Run a verification in the playground or change your filters."
          />
        )}
      </Panel>
      {current && (
        <Panel
          title="Trace details"
          aside={
            <>
              <Button variant="ghost" size="sm" onClick={() => exportTrace(current)}>
                <Download />
                Export
              </Button>
              <Button
                variant="ghost"
                size="sm"
                onClick={() => {
                  setSelected(undefined);
                  setClosed(true);
                }}
              >
                <X size={14} />
                Close
              </Button>
            </>
          }
        >
          <Trace result={current} />
        </Panel>
      )}
    </>
  );
}
