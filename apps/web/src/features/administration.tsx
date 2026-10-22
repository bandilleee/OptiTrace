'use client';
import { emailError, nameError } from '@/lib/validation';
import { FieldError } from '@/components/field-error';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from '@/components/ui/dialog';
import { Tabs, TabsList, TabsTrigger, TabsContent } from '@/components/ui/tabs';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { SelectField } from '@/components/select-field';
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
  const [errors, setErrors] = useState<{ email?: string; name?: string }>({});
  const [notice, setNotice] = useState('');
  const [revoke, setRevoke] = useState<Credential | null>(null);
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
    onSuccess: (data, variables) => {
      setNotice(
        variables.method === 'DELETE'
          ? 'Credential revoked.'
          : variables.method === 'PATCH'
            ? 'Member roles saved.'
            : variables.path === 'invitations'
              ? 'Invitation created. No email was sent.'
              : 'Credential created. Save the secret before dismissing it.',
      );
      if (variables.method === 'DELETE') setRevoke(null);
      if (data?.secret) setSecret(data.secret);
      if (variables.path === 'invitations') setEmail('');
      if (variables.path === 'credentials') setKeyName('');
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
      {notice && (
        <p role="status" className="success-message">
          {notice}
        </p>
      )}
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
                          onSave={async (roles) => {
                            await change.mutateAsync({
                              path: `memberships/${m.id}`,
                              method: 'PATCH',
                              body: { roles },
                            });
                          }}
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
              noValidate
              onSubmit={(e) => {
                e.preventDefault();
                const error = emailError(email);
                setErrors({ email: error });
                if (!error)
                  change.mutate({
                    path: 'invitations',
                    body: { email: email.trim().toLowerCase(), roles: [inviteRole] },
                  });
              }}
            >
              <Label>
                Google account email
                <Input
                  aria-invalid={!!errors.email}
                  aria-describedby="invite-email-error"
                  type="email"
                  required
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                  placeholder="teammate@example.com"
                />
                <FieldError id="invite-email-error" error={errors.email} />
              </Label>
              <Label>
                Initial role
                <SelectField
                  aria-label="Initial role"
                  value={inviteRole}
                  onValueChange={(value) => setInviteRole(value)}
                >
                  {roles.map((r) => (
                    <option key={r} value={r}>
                      {label(r)}
                    </option>
                  ))}
                </SelectField>
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
              noValidate
              onSubmit={(e) => {
                e.preventDefault();
                const error = nameError(keyName);
                setErrors({ name: error });
                if (!error) {
                  setSecret('');
                  change.mutate({
                    path: 'credentials',
                    body: { name: keyName.trim(), scopes: [keyScope] },
                  });
                }
              }}
            >
              <Label>
                Credential name
                <Input
                  required
                  aria-invalid={!!errors.name}
                  aria-describedby="credential-name-error"
                  value={keyName}
                  onChange={(e) => setKeyName(e.target.value)}
                  placeholder="Local integration"
                />
                <FieldError id="credential-name-error" error={errors.name} />
              </Label>
              <Label>
                Scope
                <SelectField
                  aria-label="Scope"
                  value={keyScope}
                  onValueChange={(value) => setKeyScope(value)}
                >
                  <option value="evaluations:write">Submit evaluations</option>
                  <option value="evaluations:read">Read evaluations</option>
                </SelectField>
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
                            onClick={() => {
                              change.reset();
                              setRevoke(c);
                            }}
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
      <Dialog
        open={!!revoke}
        onOpenChange={(open) => {
          if (!open && !change.isPending) setRevoke(null);
        }}
      >
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Revoke integration credential?</DialogTitle>
            <DialogDescription>
              Revoke {revoke?.name}. Clients using this secret will lose access immediately. This
              cannot be undone; you can create a replacement credential.
            </DialogDescription>
          </DialogHeader>
          <ErrorMessage error={change.error} />
          <DialogFooter>
            <Button variant="outline" disabled={change.isPending} onClick={() => setRevoke(null)}>
              Cancel revocation
            </Button>
            <Button
              variant="destructive"
              disabled={change.isPending}
              onClick={() => {
                if (revoke) change.mutate({ path: `credentials/${revoke.id}`, method: 'DELETE' });
              }}
            >
              Confirm revocation
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}

export function RoleEditor({
  member,
  onSave,
  pending,
}: {
  member: Membership;
  onSave: (roles: string[]) => Promise<void>;
  pending: boolean;
}) {
  const [error, setError] = useState('');
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
      {error && (
        <small role="alert" className="field-error">
          {error}
        </small>
      )}
      <Button
        className="button secondary small"
        disabled={pending}
        onClick={async () => {
          if (!selected.length) {
            setError('Choose at least one role.');
            return;
          }
          setError('');
          try {
            await onSave(selected);
            setEditing(false);
          } catch (failure) {
            setError(
              failure instanceof Error ? failure.message : 'Roles were not saved. Try again.',
            );
          }
        }}
      >
        Save roles
      </Button>
      <Button variant="ghost" className="text-button" onClick={() => setEditing(false)}>
        Cancel
      </Button>
    </div>
  ) : (
    <Button
      variant="ghost"
      className="text-button"
      onClick={() => {
        setSelected(member.roles);
        setError('');
        setEditing(true);
      }}
    >
      Edit roles
    </Button>
  );
}
