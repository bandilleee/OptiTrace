'use client';
import { type Props } from '../lib/workspace-types';
import { useItems } from '../lib/use-items';
import { type Policy, api, can, label } from '../lib/api';
import { useQueryClient, useMutation } from '@tanstack/react-query';
import { useState } from 'react';
import { ErrorMessage, Panel, Badge } from '../components/ui';

export function Policies({ tenant, identity }: Props) {
  const query = useItems<Policy>(tenant.id, 'policies');
  const client = useQueryClient();
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
            <label>
              Policy name
              <input
                required
                maxLength={120}
                value={name}
                onChange={(e) => setName(e.target.value)}
                placeholder="Account verification"
              />
            </label>
            <label>
              Profile
              <select value={profile} onChange={(e) => setProfile(e.target.value)}>
                <option value="conditional">Conditional verification</option>
                <option value="complete">Complete verification</option>
              </select>
            </label>
            <button className="button primary" disabled={change.isPending}>
              Create draft
            </button>
          </form>
        </Panel>
      )}
      <Panel
        title="Policy versions"
        aside={<span className="subtle">Approved versions are immutable</span>}
      >
        <div className="table-scroll">
          <table>
            <thead>
              <tr>
                <th>Policy</th>
                <th>Profile</th>
                <th>Version</th>
                <th>Status</th>
                <th>Next action</th>
              </tr>
            </thead>
            <tbody>
              {query.data?.items.map((p) => (
                <tr key={p.id}>
                  <td>
                    <strong>{p.name}</strong>
                  </td>
                  <td>{label(p.profile)}</td>
                  <td>v{p.version}</td>
                  <td>
                    <Badge value={p.status} />
                  </td>
                  <td>
                    {p.status === 'draft' &&
                    p.author_id === identity.user.id &&
                    can(tenant.roles, 'policy_author') ? (
                      <button
                        className="text-button"
                        disabled={change.isPending}
                        onClick={() => change.mutate({ id: p.id, action: 'submit' })}
                      >
                        Submit for review
                      </button>
                    ) : p.status === 'submitted' &&
                      can(tenant.roles, 'policy_reviewer') &&
                      p.author_id !== identity.user.id ? (
                      <button
                        className="text-button"
                        disabled={change.isPending}
                        onClick={() => change.mutate({ id: p.id, action: 'approve' })}
                      >
                        Approve version
                      </button>
                    ) : p.status === 'approved' &&
                      can(tenant.roles, 'policy_reviewer') &&
                      p.author_id !== identity.user.id ? (
                      <button
                        className="text-button"
                        disabled={change.isPending}
                        onClick={() => change.mutate({ id: p.id, action: 'activate' })}
                      >
                        Activate version
                      </button>
                    ) : (
                      <span className="subtle">
                        {p.status === 'active' ? 'Ready for evaluation' : 'No permitted action'}
                      </span>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </Panel>
      <p className="footnote">
        Demonstration thresholds: recent change within 24 hours; source freshness 60 seconds.
        Authors cannot approve their own versions.
      </p>
    </>
  );
}
