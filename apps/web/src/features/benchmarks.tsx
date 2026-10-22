'use client';
import { type Props } from '../lib/workspace-types';
import { useQueryClient, useQuery, useMutation } from '@tanstack/react-query';
import { useState } from 'react';
import { useItems } from '../lib/use-items';
import { type Benchmark, api, label, can, cost } from '../lib/api';
import { Panel, ErrorMessage, Badge, Empty } from '../components/ui';
import { FlaskConical, ChevronRight, Download } from 'lucide-react';

export function Benchmarks({ tenant, identity }: Props) {
  const client = useQueryClient();
  const [seed, setSeed] = useState(42);
  const [count, setCount] = useState(40);
  const [selected, setSelected] = useState('');
  const runs = useItems<Benchmark>(tenant.id, 'benchmarks');
  const run = useQuery({
    queryKey: [tenant.id, 'benchmark', selected],
    enabled: !!selected,
    queryFn: () => api<Benchmark>(`/tenants/${tenant.id}/benchmarks/${selected}`),
    refetchInterval: (q) =>
      q.state.data && ['completed', 'succeeded', 'failed'].includes(q.state.data.status)
        ? false
        : 2000,
  });
  const start = useMutation({
    mutationFn: () =>
      api<Benchmark>(`/tenants/${tenant.id}/benchmarks`, {
        method: 'POST',
        csrf: identity.csrf_token,
        body: { seed, count },
      }),
    onSuccess: (r) => {
      setSelected(r.id);
      client.invalidateQueries({ queryKey: [tenant.id, 'benchmarks'] });
    },
  });
  const current = run.data;
  const rows = current?.results ?? [];
  return (
    <>
      <Panel
        title="Reproducible comparison"
        aside={<span className="subtle">Held-out synthetic observations</span>}
      >
        <p className="panel-copy">
          Replay the same observations across all five strategies. Correctness comes first; a
          negative savings result remains a valid finding.
        </p>
        <form
          className="inline-form"
          onSubmit={(e) => {
            e.preventDefault();
            start.mutate();
          }}
        >
          <label>
            Random seed
            <input
              type="number"
              value={seed}
              onChange={(e) => setSeed(Number(e.target.value))}
              min={0}
              required
            />
          </label>
          <label>
            Observation count
            <input
              type="number"
              min={2}
              max={1000}
              required
              value={count}
              onChange={(e) => setCount(Number(e.target.value))}
            />
          </label>
          <button
            className="button primary"
            disabled={start.isPending || !can(tenant.roles, 'integration_developer')}
          >
            <FlaskConical size={17} />
            Launch benchmark
          </button>
        </form>
        <ErrorMessage error={start.error ?? runs.error ?? run.error} />
      </Panel>
      <Panel title="Benchmark runs">
        <div className="run-list">
          {runs.data?.items.map((r) => (
            <button
              className={selected === r.id ? 'run-row selected' : 'run-row'}
              key={r.id}
              onClick={() => setSelected(r.id)}
            >
              <span>
                <strong>{r.id.slice(0, 8)}</strong>
                <small>
                  Seed {r.seed} · {r.count} requests
                </small>
              </span>
              <Badge value={selected === r.id && current ? current.status : r.status} />
              <ChevronRight size={17} />
            </button>
          ))}
          {runs.data && !runs.data.items.length && (
            <Empty
              title="Ready to measure"
              text="Launch a seeded run to compare correctness, configured cost, and latency."
            />
          )}
        </div>
      </Panel>
      {current && (
        <Panel
          title="Comparison report"
          aside={
            <>
              <Badge value={current.status} />
              {['completed', 'succeeded'].includes(current.status) && (
                <a
                  className="button secondary small"
                  href={`/api/v1/tenants/${tenant.id}/benchmarks/${current.id}/export`}
                  download
                >
                  <Download size={15} />
                  Export JSON
                </a>
              )}
            </>
          }
        >
          <p className="panel-copy">
            Virtual simulation and measured HTTP execution must be read separately. Configured
            supplier charges are synthetic.
          </p>
          {current.error && <ErrorMessage error={new Error(current.error)} />}
          <div className="table-scroll">
            <table>
              <thead>
                <tr>
                  <th>Strategy</th>
                  <th>Profile</th>
                  <th>Measurement</th>
                  <th>Agreement</th>
                  <th>Compulsory violations</th>
                  <th>Mean cost</th>
                  <th>p95</th>
                </tr>
              </thead>
              <tbody>
                {rows.map((r, i) => (
                  <tr key={i}>
                    <td>{label(String(r.strategy ?? ''))}</td>
                    <td>{label(String(r.profile ?? ''))}</td>
                    <td>{label(String(r.measurement ?? 'Unspecified'))}</td>
                    <td>{(Number(r.agreement_rate ?? 0) * 100).toFixed(1)}%</td>
                    <td>{String(r.compulsory_violations ?? '—')}</td>
                    <td>{cost(Number(r.mean_cost_microrand ?? 0))}</td>
                    <td>{Number(r.p95_ms ?? 0).toFixed(1)} ms</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          {!rows.length && (
            <p className="panel-copy">
              {['failed'].includes(current.status)
                ? 'The worker reported a failure. Inspect the error and start a new run.'
                : 'Waiting for the benchmark worker. This page refreshes automatically.'}
            </p>
          )}
          <details className="report-json">
            <summary>Inspect complete report and gate evidence</summary>
            <pre>{JSON.stringify(current, null, 2)}</pre>
          </details>
        </Panel>
      )}
    </>
  );
}
