export const roleWorkflows = [
  {
    role: 'tenant_admin',
    title: 'Tenant administrator',
    description:
      'Manage invitations, memberships and scoped integration credentials. This role alone does not grant access to verification evidence.',
    view: 'workspace',
    action: 'Manage workspace access',
  },
  {
    role: 'policy_author',
    title: 'Policy author',
    description:
      'Create policy drafts and submit them for independent review. You cannot approve your own changes.',
    view: 'policies',
    action: 'Write and submit policies',
  },
  {
    role: 'policy_reviewer',
    title: 'Policy reviewer',
    description: 'Review, approve and activate policy versions authored by someone else.',
    view: 'policies',
    action: 'Review policy versions',
  },
  {
    role: 'integration_developer',
    title: 'Integration developer',
    description:
      'Run synthetic verifications, inspect execution traces and launch bounded, reproducible benchmarks.',
    view: 'playground',
    action: 'Open verification playground',
  },
  {
    role: 'auditor',
    title: 'Auditor',
    description:
      'Inspect policy versions, traces, benchmark reports and audit history. Evidence access is read-only.',
    view: 'audit',
    action: 'Inspect audit history',
  },
] as const;
const permissions: Record<string, readonly string[]> = {
  overview: roleWorkflows.map((r) => r.role),
  playground: ['integration_developer'],
  traces: ['integration_developer', 'auditor'],
  policies: ['policy_author', 'policy_reviewer', 'integration_developer', 'auditor'],
  benchmarks: ['integration_developer', 'auditor'],
  workspace: ['tenant_admin'],
  audit: ['auditor'],
};
export function canView(view: string, roles: readonly string[]): boolean {
  return (permissions[view] ?? []).some((role) => roles.includes(role));
}
