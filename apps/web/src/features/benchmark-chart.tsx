'use client';
import { useState } from 'react';
import { NativeSelect } from '@/components/ui/native-select';
import { label, cost } from '@/lib/api';

export function BenchmarkChart({ rows }: { rows: Record<string, unknown>[] }) {
  const [cohort, setCohort] = useState('');
  const [metric, setMetric] = useState('mean_cost_microrand');
  const cohorts = [...new Set(rows.map((r) => `${r.profile}|${r.measurement}`))];
  const selected = cohorts.includes(cohort) ? cohort : cohorts[0];
  const samples = rows.filter((r) => `${r.profile}|${r.measurement}` === selected);
  const valid = samples.filter(
    (r) => typeof r[metric] === 'number' && Number.isFinite(r[metric]) && Number(r[metric]) >= 0,
  );
  const maximum = Math.max(...valid.map((r) => Number(r[metric])), 0);
  if (!rows.length) return null;
  return (
    <section className="benchmark-chart" aria-label="Strategy comparison chart">
      <div className="chart-controls">
        <div>
          <h3>Compare like for like</h3>
          <p>Each chart uses one policy profile and measurement type.</p>
        </div>
        <div>
          <NativeSelect
            aria-label="Chart cohort"
            value={selected}
            onChange={(e) => setCohort(e.target.value)}
          >
            {cohorts.map((c) => (
              <option key={c} value={c}>
                {c.split('|').map(label).join(' · ')}
              </option>
            ))}
          </NativeSelect>
          <NativeSelect
            aria-label="Chart metric"
            value={metric}
            onChange={(e) => setMetric(e.target.value)}
          >
            <option value="mean_cost_microrand">Mean synthetic cost</option>
            <option value="p95_ms">p95 latency</option>
          </NativeSelect>
        </div>
      </div>
      <div className="chart-bars">
        {valid.map((r) => (
          <div className="chart-row" key={String(r.strategy)}>
            <span>{label(String(r.strategy))}</span>
            <div className="chart-track">
              <div
                className={'chart-fill ' + (r.strategy === 'optitrace' ? 'is-optitrace' : '')}
                style={{ width: maximum ? `${(Number(r[metric]) / maximum) * 100}%` : '0%' }}
              />
            </div>
            <strong>
              {metric === 'p95_ms' ? `${Number(r[metric]).toFixed(1)} ms` : cost(Number(r[metric]))}
            </strong>
          </div>
        ))}
      </div>
      <p className="chart-note">
        Configured supplier charges and estimated runtime cost are synthetic. The full results and
        correctness evidence remain in the table and export.
      </p>
    </section>
  );
}
