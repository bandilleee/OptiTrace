'use client';
import { BenchmarkChart } from './benchmark-chart';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import {
  Table,
  TableHeader,
  TableBody,
  TableRow,
  TableHead,
  TableCell,
} from '@/components/ui/table';
import { type Props } from '../lib/workspace-types';
import { useQueryClient, useQuery, useMutation } from '@tanstack/react-query';
import { createBenchmark, getBenchmark } from '../lib/sdk';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from '@/components/ui/dialog';
import { FieldError } from '@/components/field-error';
import { integerError } from '@/lib/validation';
import { useRef, useState } from 'react';
import { useItems } from '../lib/use-items';
import { type Benchmark, label, can, cost } from '../lib/api';
import { Panel, ErrorMessage, Badge, Empty } from '../components/ui';
import { FlaskConical, ChevronRight, Download } from 'lucide-react';

export function Benchmarks({ tenant, identity }: Props) {
  const client = useQueryClient();
  const [seed, setSeed] = useState('42');
  const [count, setCount] = useState('40');
  const [open, setOpen] = useState(false);
  const [errors, setErrors] = useState<{ seed?: string; count?: string }>({});
  const returnFocus = useRef<HTMLButtonElement | null>(null);
  const launch = useRef<HTMLButtonElement | null>(null);
  const [selected, setSelected] = useState('');
  const runs = useItems<Benchmark>(tenant.id, 'benchmarks', true, 5000);
  const run = useQuery({
    queryKey: [tenant.id, 'benchmark', selected],
    enabled: open && !!selected,
    queryFn: () => getBenchmark({ path: { tenant: tenant.id, id: selected } }),
    refetchInterval: (q) =>
      q.state.data && ['completed', 'succeeded', 'failed'].includes(q.state.data.status)
        ? false
        : 2000,
  });
  const start = useMutation({
    mutationFn: () =>
      createBenchmark({
        path: { tenant: tenant.id },
        csrf: identity.csrf_token,
        body: { seed: Number(seed), count: Number(count) },
      }),
    onSuccess: (r) => {
      setSelected(r.id);
      returnFocus.current = launch.current;
      setOpen(true);
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
          noValidate
          onSubmit={(e) => {
            e.preventDefault();
            const next = {
              seed: integerError(seed, Number.MIN_SAFE_INTEGER, Number.MAX_SAFE_INTEGER),
              count: integerError(count, 2, 1000),
            };
            setErrors(next);
            if (!next.seed && !next.count) start.mutate();
          }}
        >
          <Label>
            Random seed
            <Input
              type="number"
              aria-invalid={!!errors.seed}
              aria-describedby="seed-error"
              value={seed}
              onChange={(e) => setSeed(e.target.value)}
              required
            />
            <FieldError id="seed-error" error={errors.seed} />
          </Label>
          <Label>
            Observation count
            <Input
              type="number"
              min={2}
              max={1000}
              required
              aria-invalid={!!errors.count}
              aria-describedby="count-error"
              value={count}
              onChange={(e) => setCount(e.target.value)}
            />
            <FieldError id="count-error" error={errors.count} />
          </Label>
          <Button
            ref={launch}
            className="button primary"
            disabled={start.isPending || !can(tenant.roles, 'integration_developer')}
          >
            <FlaskConical size={17} />
            Launch benchmark
          </Button>
        </form>
        <ErrorMessage error={start.error ?? runs.error} />
      </Panel>
      <Panel title="Benchmark runs">
        {runs.isPending && (
          <p role="status" className="panel-copy">
            Loading benchmark runs…
          </p>
        )}
        <div className="run-list">
          {runs.data?.items.map((r) => (
            <Button
              className={selected === r.id ? 'run-row selected' : 'run-row'}
              key={r.id}
              onClick={(event) => {
                returnFocus.current = event.currentTarget;
                setSelected(r.id);
                setOpen(true);
              }}
            >
              <span>
                <strong>{r.id.slice(0, 8)}</strong>
                <small>
                  Seed {r.seed} · {r.count} requests
                </small>
              </span>
              <Badge value={open && selected === r.id && current ? current.status : r.status} />
              <ChevronRight size={17} />
            </Button>
          ))}
          {runs.data && !runs.data.items.length && (
            <Empty
              title="Ready to measure"
              text="Launch a seeded run to compare correctness, configured cost, and latency."
            />
          )}
        </div>
      </Panel>
      <Dialog open={open} onOpenChange={setOpen}>
        <DialogContent
          className="benchmark-dialog"
          onCloseAutoFocus={(event) => {
            event.preventDefault();
            returnFocus.current?.focus();
          }}
        >
          <DialogHeader>
            <DialogTitle>Comparison report</DialogTitle>
            <DialogDescription>
              Run {selected.slice(0, 8)}. Synthetic charges and measured runtime are reported
              separately.
            </DialogDescription>
          </DialogHeader>
          {run.isPending && <p role="status">Loading benchmark report…</p>}
          <ErrorMessage error={run.error} />
          {run.isError && (
            <Button variant="outline" onClick={() => run.refetch()}>
              Retry report
            </Button>
          )}
          {current && (
            <Panel
              title="Strategy results"
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
              <ErrorMessage
                error={
                  rows.some(
                    (row) =>
                      (typeof row.agreement_rate === 'number' && row.agreement_rate < 1) ||
                      (typeof row.compulsory_violations === 'number' &&
                        row.compulsory_violations > 0),
                  )
                    ? new Error(
                        'This run contains outcome disagreements or compulsory-work violations. Review the comparison table and complete report before using these results.',
                      )
                    : null
                }
              />
              <BenchmarkChart rows={rows} />
              <div className="table-scroll">
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>Strategy</TableHead>
                      <TableHead>Profile</TableHead>
                      <TableHead>Measurement</TableHead>
                      <TableHead>Agreement</TableHead>
                      <TableHead>Compulsory violations</TableHead>
                      <TableHead>Mean cost</TableHead>
                      <TableHead>p95</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {rows.map((r, i) => (
                      <TableRow key={i}>
                        <TableCell>{label(String(r.strategy ?? ''))}</TableCell>
                        <TableCell>{label(String(r.profile ?? ''))}</TableCell>
                        <TableCell>{label(String(r.measurement ?? 'Unspecified'))}</TableCell>
                        <TableCell>
                          {typeof r.agreement_rate === 'number'
                            ? `${(r.agreement_rate * 100).toFixed(1)}%`
                            : 'Unavailable'}
                        </TableCell>
                        <TableCell>{String(r.compulsory_violations ?? '—')}</TableCell>
                        <TableCell>
                          {cost(
                            typeof r.mean_cost_microrand === 'number'
                              ? r.mean_cost_microrand
                              : null,
                          )}
                        </TableCell>
                        <TableCell>
                          {typeof r.p95_ms === 'number'
                            ? `${r.p95_ms.toFixed(1)} ms`
                            : 'Unavailable'}
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </div>
              {!rows.length && (
                <p className="panel-copy">
                  {['failed'].includes(current.status)
                    ? 'The worker reported a failure. Inspect the error and start a new run.'
                    : 'Waiting for the benchmark worker. This report refreshes automatically.'}
                </p>
              )}
              <details className="report-json">
                <summary>Inspect complete report and gate evidence</summary>
                <pre>{JSON.stringify(current, null, 2)}</pre>
              </details>
            </Panel>
          )}
        </DialogContent>
      </Dialog>
    </>
  );
}
