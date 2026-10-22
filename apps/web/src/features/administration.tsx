'use client';
import { type Props } from '../lib/workspace-types';
import { can, api, label } from '../lib/api';
import { useItems } from '../lib/use-items';
import { useQueryClient, useMutation } from '@tanstack/react-query';
import { useState } from 'react';
import { Empty, ErrorMessage, Panel } from '../components/ui';

export const roles = [
  'tenant_admin',
  'policy_author',
  'policy_reviewer',
  'integration_developer',
  'auditor',
];

export type Membership = {
  id: string;
  email: string;
  roles: string[];
};

export type Invitation = {
  id: string;
  email: string;
  roles: string[];
  status?: string;
};

export type Credential = {
  id: string;
  name: string;
  scopes: string[];
  revoked_at?: string;
};

export function Administration({ tenant, identity }: Props) {
  const allowed = can(tenant.roles, 'tenant_admin');
  const members = useItems<Membership>(tenant.id, 'memberships', allowed);
  const invitations = useItems<Invitation>(tenant.id, 'invitations', allowed);
  const credentials = useItems<Credential>(tenant.id, 'credentials', allowed);
  const client = useQueryClient();
  const [email, setEmail] = useState('');
  const [inviteRole, setInviteRole] = useState('integration_developer');
  const [keyName, setKeyName] = useState('');
  const [secret, setSecret] = useState('');
  const [keyScope, setKeyScope] = useState('evaluations:write');
  const change = useMutation({
    mutationFn: ({
      path,
      method = 'POST',
      body,
    }: {
      path: string;
      method?: string;
      body?: unknown;
    }) =>
      api<{
        secret?: string;
      }>(`/tenants/${tenant.id}/${path}`, { method, body, csrf: identity.csrf_token }),
    onSuccess: (data) => {
      if (data?.secret) setSecret(data.secret);
      setEmail('');
      setKeyName('');
      client.invalidateQueries({ queryKey: ['identity'] });
      for (const resource of ['memberships', 'invitations', 'credentials'])
        client.invalidateQueries({ queryKey: [tenant.id, resource] });
    },
  });
  if (!allowed)
    return (
      <Empty
        title="Administration is restricted"
        text="Your account needs the tenant administrator role to manage memberships, invitations, and credentials."
      />
    );
  return (
    <>
      <ErrorMessage
        error={change.error ?? members.error ?? invitations.error ?? credentials.error}
      />
      <Panel title="Workspace members">
        <div className="table-scroll">
          <table>
            <thead>
              <tr>
                <th>Member</th>
                <th>Roles</th>
                <th>Manage access</th>
              </tr>
            </thead>
            <tbody>
              {members.data?.items.map((m) => (
                <tr key={m.id}>
                  <td>{m.email}</td>
                  <td>{m.roles.map(label).join(', ') || 'No roles'}</td>
                  <td>
                    <RoleEditor
                      member={m}
                      pending={change.isPending}
                      onSave={(roles) =>
                        change.mutate({
                          path: `memberships/${m.id}`,
                          method: 'PATCH',
                          body: { roles },
                        })
                      }
                    />
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </Panel>
      <Panel title="Invite a teammate">
        <form
          className="inline-form"
          onSubmit={(e) => {
            e.preventDefault();
            change.mutate({ path: 'invitations', body: { email, roles: [inviteRole] } });
          }}
        >
          <label>
            Google account email
            <input
              type="email"
              required
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              placeholder="teammate@example.com"
            />
          </label>
          <label>
            Initial role
            <select value={inviteRole} onChange={(e) => setInviteRole(e.target.value)}>
              {roles.map((r) => (
                <option key={r} value={r}>
                  {label(r)}
                </option>
              ))}
            </select>
          </label>
          <button className="button primary" disabled={change.isPending}>
            Create invitation
          </button>
        </form>
        <div className="compact-list">
          {invitations.data?.items.map((i) => (
            <div key={i.id}>
              <span>{i.email}</span>
              <span className="subtle">
                {i.roles.map(label).join(', ')} · {label(i.status ?? 'pending')}
              </span>
            </div>
          ))}
        </div>
        <p className="panel-copy">
          Invitations authorize sign-in; this action does not send an email.
        </p>
      </Panel>
      <Panel
        title="Integration credentials"
        aside={<span className="subtle">Secrets are shown once</span>}
      >
        <form
          className="inline-form"
          onSubmit={(e) => {
            e.preventDefault();
            setSecret('');
            change.mutate({ path: 'credentials', body: { name: keyName, scopes: [keyScope] } });
          }}
        >
          <label>
            Credential name
            <input
              required
              value={keyName}
              onChange={(e) => setKeyName(e.target.value)}
              placeholder="Local integration"
            />
          </label>
          <label>
            Scope
            <select value={keyScope} onChange={(e) => setKeyScope(e.target.value)}>
              <option value="evaluations:write">Submit evaluations</option>
              <option value="evaluations:read">Read evaluations</option>
            </select>
          </label>
          <button className="button primary" disabled={change.isPending}>
            Create credential
          </button>
        </form>
        {secret && (
          <div className="secret-box" role="status">
            <strong>Copy this secret now. It will not be shown again.</strong>
            <code>{secret}</code>
            <button className="text-button" onClick={() => setSecret('')}>
              Dismiss secret
            </button>
          </div>
        )}
        <div className="table-scroll">
          <table>
            <thead>
              <tr>
                <th>Name</th>
                <th>Scopes</th>
                <th>Status</th>
                <th>Action</th>
              </tr>
            </thead>
            <tbody>
              {credentials.data?.items.map((c) => (
                <tr key={c.id}>
                  <td>{c.name}</td>
                  <td>{c.scopes.join(', ')}</td>
                  <td>{c.revoked_at ? 'Revoked' : 'Active'}</td>
                  <td>
                    {!c.revoked_at && (
                      <button
                        className="text-button danger"
                        disabled={change.isPending}
                        onClick={() =>
                          change.mutate({ path: `credentials/${c.id}`, method: 'DELETE' })
                        }
                      >
                        Revoke credential
                      </button>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </Panel>
    </>
  );
}

export function RoleEditor({
  member,
  onSave,
  pending,
}: {
  member: Membership;
  onSave: (roles: string[]) => void;
  pending: boolean;
}) {
  const [editing, setEditing] = useState(false);
  const [selected, setSelected] = useState(member.roles);
  return editing ? (
    <div className="role-editor">
      {roles.map((r) => (
        <label key={r}>
          <input
            type="checkbox"
            checked={selected.includes(r)}
            onChange={(e) =>
              setSelected(e.target.checked ? [...selected, r] : selected.filter((s) => s !== r))
            }
          />
          {label(r)}
        </label>
      ))}
      <button
        className="button secondary small"
        disabled={pending}
        onClick={() => {
          onSave(selected);
          setEditing(false);
        }}
      >
        Save roles
      </button>
      <button className="text-button" onClick={() => setEditing(false)}>
        Cancel
      </button>
    </div>
  ) : (
    <button className="text-button" onClick={() => setEditing(true)}>
      Edit roles
    </button>
  );
}
