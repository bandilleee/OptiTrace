'use client';
import {
  Table,
  TableHeader,
  TableBody,
  TableRow,
  TableHead,
  TableCell,
} from '@/components/ui/table';
import { useState } from 'react';
import { Input } from '@/components/ui/input';
import { type Props } from '../lib/workspace-types';
import { useItems } from '../lib/use-items';
import { ErrorMessage, Panel, Empty } from '../components/ui';
import { label } from '../lib/api';

export function Audit({ tenant }: Props) {
  const [search, setSearch] = useState('');
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
        <div className="table-toolbar">
          <Input
            aria-label="Search audit history"
            placeholder="Search action, actor, or resource…"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
          />
        </div>
        <div className="table-scroll">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Time</TableHead>
                <TableHead>Actor</TableHead>
                <TableHead>Action</TableHead>
                <TableHead>Resource / details</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {query.data?.items
                .filter((e) => JSON.stringify(e).toLowerCase().includes(search.toLowerCase()))
                .map((event, i) => (
                  <TableRow key={String(event.id ?? i)}>
                    <TableCell>
                      {event.created_at ? new Date(String(event.created_at)).toLocaleString() : '—'}
                    </TableCell>
                    <TableCell>{String(event.actor_id ?? event.actor ?? 'System')}</TableCell>
                    <TableCell>
                      {label(String(event.action ?? event.event_type ?? 'Event'))}
                    </TableCell>
                    <TableCell>
                      <details>
                        <summary>Inspect event</summary>
                        <pre>{JSON.stringify(event, null, 2)}</pre>
                      </details>
                    </TableCell>
                  </TableRow>
                ))}
            </TableBody>
          </Table>
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
