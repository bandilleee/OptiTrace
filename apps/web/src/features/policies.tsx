'use client';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { NativeSelect } from '@/components/ui/native-select';
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
import { useItems } from '../lib/use-items';
import { type Policy, api, can, label } from '../lib/api';
import { useQueryClient, useMutation } from '@tanstack/react-query';
import { useState } from 'react';
import { ErrorMessage, Panel, Badge } from '../components/ui';

export function Policies({ tenant, identity }: Props) {
  const query = useItems<Policy>(tenant.id, 'policies');
  const client = useQueryClient();
  const [search, setSearch] = useState('');
  const [name, setName] = useState('');
  const [profile, setProfile] = useState('conditional');
  const change = useMutation({
    mutationFn: ({ id, action }: { id?: string; action: string }) =>
      api(`/tenants/${tenant.id}/policies${id ? `/${id}/${action}` : ''}`, {
        method: 'POST',
        csrf: identity.csrf_token,
        body: id ? undefined : { name, profile },
      }),
    onSuccess: () => {
      setName('');
      client.invalidateQueries({ queryKey: [tenant.id, 'policies'] });
    },
  });
  return (
    <>
      <ErrorMessage error={query.error ?? change.error} />
      {query.isPending && (
        <p role="status" className="subtle">
          Loading policies…
        </p>
      )}
      {can(tenant.roles, 'policy_author') && (
        <Panel title="Create a policy draft">
          <form
            className="inline-form"
            onSubmit={(e) => {
              e.preventDefault();
              change.mutate({ action: 'create' });
            }}
          >
            <Label>
              Policy name
              <Input
                required
                maxLength={120}
                value={name}
                onChange={(e) => setName(e.target.value)}
                placeholder="Account verification"
              />
            </Label>
            <Label>
              Profile
              <NativeSelect value={profile} onChange={(e) => setProfile(e.target.value)}>
                <option value="conditional">Conditional verification</option>
                <option value="complete">Complete verification</option>
              </NativeSelect>
            </Label>
            <Button className="button primary" disabled={change.isPending}>
              Create draft
            </Button>
          </form>
        </Panel>
      )}
      <Panel
        title="Policy versions"
        aside={<span className="subtle">Approved versions are immutable</span>}
      >
        <div className="table-toolbar">
          <Input
            aria-label="Search policies"
            placeholder="Search policy name, profile, or status…"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
          />
        </div>
        <div className="table-scroll">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Policy</TableHead>
                <TableHead>Profile</TableHead>
                <TableHead>Version</TableHead>
                <TableHead>Status</TableHead>
                <TableHead>Next action</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {query.data?.items
                .filter((p) =>
                  (p.name + ' ' + p.profile + ' ' + p.status)
                    .toLowerCase()
                    .includes(search.toLowerCase()),
                )
                .map((p) => (
                  <TableRow key={p.id}>
                    <TableCell>
                      <strong>{p.name}</strong>
                    </TableCell>
                    <TableCell>{label(p.profile)}</TableCell>
                    <TableCell>v{p.version}</TableCell>
                    <TableCell>
                      <Badge value={p.status} />
                    </TableCell>
                    <TableCell>
                      {p.status === 'draft' &&
                      p.author_id === identity.user.id &&
                      can(tenant.roles, 'policy_author') ? (
                        <Button
                          variant="ghost"
                          className="text-button"
                          disabled={change.isPending}
                          onClick={() => change.mutate({ id: p.id, action: 'submit' })}
                        >
                          Submit for review
                        </Button>
                      ) : p.status === 'submitted' &&
                        can(tenant.roles, 'policy_reviewer') &&
                        p.author_id !== identity.user.id ? (
                        <Button
                          variant="ghost"
                          className="text-button"
                          disabled={change.isPending}
                          onClick={() => change.mutate({ id: p.id, action: 'approve' })}
                        >
                          Approve version
                        </Button>
                      ) : p.status === 'approved' &&
                        can(tenant.roles, 'policy_reviewer') &&
                        p.author_id !== identity.user.id ? (
                        <Button
                          variant="ghost"
                          className="text-button"
                          disabled={change.isPending}
                          onClick={() => change.mutate({ id: p.id, action: 'activate' })}
                        >
                          Activate version
                        </Button>
                      ) : (
                        <span className="subtle">
                          {p.status === 'active' ? 'Ready for evaluation' : 'No permitted action'}
                        </span>
                      )}
                    </TableCell>
                  </TableRow>
                ))}
            </TableBody>
          </Table>
        </div>
      </Panel>
      <p className="footnote">
        Demonstration thresholds: recent change within 24 hours; source freshness 60 seconds.
        Authors cannot approve their own versions.
      </p>
    </>
  );
}
