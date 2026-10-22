'use client';
import { Tabs, TabsList, TabsTrigger, TabsContent } from '@/components/ui/tabs';
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
      <Tabs defaultValue="members" className="administration-tabs">
        <TabsList aria-label="Administration sections">
          <TabsTrigger value="members">Members</TabsTrigger>
          <TabsTrigger value="invitations">Invitations</TabsTrigger>
          <TabsTrigger value="credentials">Integration credentials</TabsTrigger>
        </TabsList>
        <TabsContent value="members">
          {' '}
          <Panel title="Workspace members">
            <div className="table-scroll">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Member</TableHead>
                    <TableHead>Roles</TableHead>
                    <TableHead>Manage access</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {members.data?.items.map((m) => (
                    <TableRow key={m.id}>
                      <TableCell>{m.email}</TableCell>
                      <TableCell>{m.roles.map(label).join(', ') || 'No roles'}</TableCell>
                      <TableCell>
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
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>
          </Panel>
        </TabsContent>
        <TabsContent value="invitations">
          <Panel title="Invite a teammate">
            <form
              className="inline-form"
              onSubmit={(e) => {
                e.preventDefault();
                change.mutate({ path: 'invitations', body: { email, roles: [inviteRole] } });
              }}
            >
              <Label>
                Google account email
                <Input
                  type="email"
                  required
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                  placeholder="teammate@example.com"
                />
              </Label>
              <Label>
                Initial role
                <NativeSelect value={inviteRole} onChange={(e) => setInviteRole(e.target.value)}>
                  {roles.map((r) => (
                    <option key={r} value={r}>
                      {label(r)}
                    </option>
                  ))}
                </NativeSelect>
              </Label>
              <Button className="button primary" disabled={change.isPending}>
                Create invitation
              </Button>
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
        </TabsContent>
        <TabsContent value="credentials">
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
              <Label>
                Credential name
                <Input
                  required
                  value={keyName}
                  onChange={(e) => setKeyName(e.target.value)}
                  placeholder="Local integration"
                />
              </Label>
              <Label>
                Scope
                <NativeSelect value={keyScope} onChange={(e) => setKeyScope(e.target.value)}>
                  <option value="evaluations:write">Submit evaluations</option>
                  <option value="evaluations:read">Read evaluations</option>
                </NativeSelect>
              </Label>
              <Button className="button primary" disabled={change.isPending}>
                Create credential
              </Button>
            </form>
            {secret && (
              <div className="secret-box" role="status">
                <strong>Copy this secret now. It will not be shown again.</strong>
                <code>{secret}</code>
                <Button variant="ghost" className="text-button" onClick={() => setSecret('')}>
                  Dismiss secret
                </Button>
              </div>
            )}
            <div className="table-scroll">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Name</TableHead>
                    <TableHead>Scopes</TableHead>
                    <TableHead>Status</TableHead>
                    <TableHead>Action</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {credentials.data?.items.map((c) => (
                    <TableRow key={c.id}>
                      <TableCell>{c.name}</TableCell>
                      <TableCell>{c.scopes.join(', ')}</TableCell>
                      <TableCell>{c.revoked_at ? 'Revoked' : 'Active'}</TableCell>
                      <TableCell>
                        {!c.revoked_at && (
                          <Button
                            variant="ghost"
                            className="text-button danger"
                            disabled={change.isPending}
                            onClick={() =>
                              change.mutate({ path: `credentials/${c.id}`, method: 'DELETE' })
                            }
                          >
                            Revoke credential
                          </Button>
                        )}
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>
          </Panel>
        </TabsContent>
      </Tabs>
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
        <Label key={r}>
          <Input
            type="checkbox"
            checked={selected.includes(r)}
            onChange={(e) =>
              setSelected(e.target.checked ? [...selected, r] : selected.filter((s) => s !== r))
            }
          />
          {label(r)}
        </Label>
      ))}
      <Button
        className="button secondary small"
        disabled={pending}
        onClick={() => {
          onSave(selected);
          setEditing(false);
        }}
      >
        Save roles
      </Button>
      <Button variant="ghost" className="text-button" onClick={() => setEditing(false)}>
        Cancel
      </Button>
    </div>
  ) : (
    <Button variant="ghost" className="text-button" onClick={() => setEditing(true)}>
      Edit roles
    </Button>
  );
}
