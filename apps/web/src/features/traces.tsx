'use client';
import { type Evaluation, label, duration, cost } from '../lib/api';
import { Badge, ErrorMessage, Panel, Empty } from '../components/ui';
import { CheckCircle2, ChevronRight } from 'lucide-react';
import { type Props } from '../lib/workspace-types';
import { useItems } from '../lib/use-items';
import { useState } from 'react';

export function TraceRail() {
  return (
    <div className="trace-rail" aria-label="Three verification capabilities">
      {['Number verification', 'SIM swap', 'Device swap'].map((c, i) => (
        <div className="trace-node" key={c}>
          <div className="node-circle">{i + 1}</div>
          <span>{c}</span>
          <small>Awaiting execution</small>
        </div>
      ))}
    </div>
  );
}

export function Trace({ result }: { result: Evaluation }) {
  return (
    <div className="trace-detail">
      <div className="outcome-line">
        <Badge value={result.outcome} />
        <span className="subtle">
          Policy v{result.policy_version} · {label(result.strategy)}
        </span>
      </div>
      <div className="trace-rail">
        {result.checks.map((c, i) => (
          <div
            className={'trace-node ' + (c.status === 'skipped' ? 'is-skipped' : '')}
            key={c.capability}
          >
            <div
              className={
                'node-circle ' +
                (c.status === 'available'
                  ? c.value
                    ? 'node-adverse'
                    : 'node-good'
                  : c.status === 'skipped'
                    ? ''
                    : 'node-warning')
              }
            >
              {c.status === 'available' && !c.value ? <CheckCircle2 size={20} /> : i + 1}
            </div>
            <strong>{label(c.capability)}</strong>
            <small>{c.compulsory ? 'Compulsory' : 'Conditional'}</small>
            <Badge value={c.status} />
          </div>
        ))}
      </div>
      <p className="stopping-reason">{result.stopping_reason}</p>
      <div className="metrics">
        <div>
          <span>Elapsed</span>
          <strong>{duration(result.elapsed_ms)}</strong>
        </div>
        <div>
          <span>Supplier cost</span>
          <strong>{cost(result.supplier_cost_microrand)}</strong>
        </div>
        <div>
          <span>Runtime estimate</span>
          <strong>{cost(result.runtime_cost_microrand)}</strong>
        </div>
      </div>
      <div className="check-list">
        {result.checks.map((c) => (
          <details key={c.capability}>
            <summary>
              <span>{label(c.capability)}</span>
              <span>
                {c.status === 'available'
                  ? c.value
                    ? 'Adverse observation'
                    : 'Clear observation'
                  : label(c.status)}{' '}
                <ChevronRight size={14} />
              </span>
            </summary>
            <dl>
              <dt>Source timestamp</dt>
              <dd>
                {c.source_time && c.source_time !== '0001-01-01T00:00:00Z'
                  ? new Date(c.source_time).toLocaleString()
                  : 'No source observation'}
              </dd>
              <dt>Started at</dt>
              <dd>
                {c.started_at && c.status !== 'skipped'
                  ? new Date(c.started_at).toLocaleTimeString()
                  : 'Not started'}
              </dd>
              <dt>Latency</dt>
              <dd>{c.elapsed_ms.toFixed(1)} ms</dd>
              <dt>Configured charge</dt>
              <dd>{cost(c.cost_microrand)}</dd>
            </dl>
          </details>
        ))}
      </div>
      <>
        {result.journal?.length ? (
          <details className="report-json">
            <summary>Durable source journal</summary>
            <p>
              A starting event records a possible call. A completed event records its observed
              result. Interrupted executions require reconciliation and are never automatically
              replayed.
            </p>
            <pre>{JSON.stringify(result.journal, null, 2)}</pre>
          </details>
        ) : null}
      </>
      <div className="trace-id">Trace {result.id}</div>
    </div>
  );
}

export function Traces({ tenant }: Props) {
  const query = useItems<Evaluation>(tenant.id, 'evaluations');
  const [selected, setSelected] = useState<Evaluation>();
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
        aside={<span className="subtle">{query.data?.items.length ?? 0} recorded</span>}
      >
        <div className="table-scroll">
          <table>
            <thead>
              <tr>
                <th>Trace</th>
                <th>Scenario</th>
                <th>Policy</th>
                <th>Outcome</th>
                <th>Elapsed</th>
                <th>Supplier cost</th>
              </tr>
            </thead>
            <tbody>
              {query.data?.items.map((r) => (
                <tr key={r.id}>
                  <td>
                    <button className="text-button" onClick={() => setSelected(r)}>
                      {r.id.slice(0, 8)}
                    </button>
                  </td>
                  <td>{label(r.scenario)}</td>
                  <td>
                    {label(r.profile)} v{r.policy_version}
                  </td>
                  <td>
                    <Badge value={r.outcome} />
                  </td>
                  <td>{duration(r.elapsed_ms)}</td>
                  <td>{cost(r.supplier_cost_microrand)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
        {query.data && !query.data.items.length && (
          <Empty
            title="No evaluations yet"
            text="Run a verification in the playground to record its decision and evidence."
          />
        )}
      </Panel>
      {selected && (
        <Panel
          title="Trace details"
          aside={
            <button className="text-button" onClick={() => setSelected(undefined)}>
              Close
            </button>
          }
        >
          <Trace result={selected} />
        </Panel>
      )}
    </>
  );
}
