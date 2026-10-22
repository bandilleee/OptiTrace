'use client';
import { type Props } from '../lib/workspace-types';
import { useItems } from '../lib/use-items';
import { ErrorMessage, Panel, Empty } from '../components/ui';
import { label } from '../lib/api';

export function Audit({ tenant }: Props) {
  const query = useItems<Record<string, unknown>>(tenant.id, 'audit');
  return (
    <>
      <ErrorMessage error={query.error} />
      {query.isPending && (
        <p role="status" className="subtle">
          Loading records…
        </p>
      )}
      <Panel
        title="Recorded events"
        aside={<span className="subtle">Append-only application history</span>}
      >
        <div className="table-scroll">
          <table>
            <thead>
              <tr>
                <th>Time</th>
                <th>Actor</th>
                <th>Action</th>
                <th>Resource / details</th>
              </tr>
            </thead>
            <tbody>
              {query.data?.items.map((event, i) => (
                <tr key={String(event.id ?? i)}>
                  <td>
                    {event.created_at ? new Date(String(event.created_at)).toLocaleString() : '—'}
                  </td>
                  <td>{String(event.actor_id ?? event.actor ?? 'System')}</td>
                  <td>{label(String(event.action ?? event.event_type ?? 'Event'))}</td>
                  <td>
                    <details>
                      <summary>Inspect event</summary>
                      <pre>{JSON.stringify(event, null, 2)}</pre>
                    </details>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
        {query.data && !query.data.items.length && (
          <Empty
            title="No recorded events"
            text="Policy and workspace changes will appear here as they occur."
          />
        )}
      </Panel>
    </>
  );
}
