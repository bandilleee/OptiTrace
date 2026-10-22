'use client';
import { label, api, type Identity, APIError } from '../lib/api';
import {
  Workflow,
  Activity,
  GitBranch,
  FlaskConical,
  Users,
  FileClock,
  ChevronRight,
  ArrowRight,
  LogOut,
  ShieldCheck,
} from 'lucide-react';
import { useState } from 'react';
import { QueryClient, QueryClientProvider, useQuery, useMutation } from '@tanstack/react-query';
import { Logo, Login } from './brand';
import { ErrorMessage, Empty } from './ui';
import { type Props } from '../lib/workspace-types';
import { Playground } from '../features/playground';
import { Traces } from '../features/traces';
import { Policies } from '../features/policies';
import { Benchmarks } from '../features/benchmarks';
import { Administration } from '../features/administration';
import { Audit } from '../features/audit';

export const tabs = [
  { id: 'playground', label: 'Verification playground', icon: Workflow },
  { id: 'traces', label: 'Execution traces', icon: Activity },
  { id: 'policies', label: 'Policy library', icon: GitBranch },
  { id: 'benchmarks', label: 'Benchmarks', icon: FlaskConical },
  { id: 'workspace', label: 'Workspace', icon: Users },
  { id: 'audit', label: 'Audit history', icon: FileClock },
];

export const descriptions: Record<string, string> = {
  playground: 'Follow a decision, one API call at a time.',
  traces: 'Inspect the evidence behind every outcome.',
  policies: 'Versioned decisions. Explicit obligations.',
  benchmarks: 'Measure the optimiser against a fair baseline.',
  workspace: 'Manage who can work here and how they connect.',
  audit: 'A record of changes and the people behind them.',
};

export function Console() {
  const [client] = useState(
    () =>
      new QueryClient({
        defaultOptions: { queries: { retry: false, refetchOnWindowFocus: false } },
      }),
  );
  return (
    <QueryClientProvider client={client}>
      <Workspace />
    </QueryClientProvider>
  );
}

export function Workspace() {
  const [tab, setTab] = useState('playground');
  const [tenantID, setTenantID] = useState('');
  const identity = useQuery({ queryKey: ['identity'], queryFn: () => api<Identity>('/auth/me') });
  const tenant = identity.data?.tenants.find((t) => t.id === tenantID) ?? identity.data?.tenants[0];
  const logout = useMutation({
    mutationFn: () => api('/auth/logout', { method: 'POST', csrf: identity.data?.csrf_token }),
    onSuccess: () => window.location.assign('/'),
  });
  if (identity.isPending)
    return (
      <div className="loading-page">
        <Logo />
        <p>Opening your verification workspace…</p>
      </div>
    );
  if (!identity.data)
    return (
      <Login
        error={
          identity.error instanceof APIError && identity.error.status !== 401
            ? identity.error.message
            : undefined
        }
      />
    );
  const active = tabs.find((t) => t.id === tab)!;
  return (
    <div className="workspace">
      <aside className="sidebar">
        <Logo />
        <div className="workspace-label">Verification operations</div>
        <nav aria-label="Main navigation">
          {tabs.map((item) => (
            <button
              key={item.id}
              aria-current={tab === item.id ? 'page' : undefined}
              className={tab === item.id ? 'nav-item active' : 'nav-item'}
              onClick={() => setTab(item.id)}
            >
              <item.icon size={19} />
              <span>{item.label}</span>
              {tab === item.id && <ChevronRight size={15} />}
            </button>
          ))}
        </nav>
        <div className="sidebar-bottom">
          <div className="environment">
            <span />
            Synthetic environment
          </div>
          <p>South Africa · Demonstration</p>
          <a
            href="https://cloud.google.com/identity-platform/docs"
            target="_blank"
            rel="noreferrer"
          >
            Google Cloud identity <ArrowRight size={14} />
          </a>
        </div>
      </aside>
      <div className="main-shell">
        <header className="topbar">
          <div className="breadcrumb">
            Workspace <ChevronRight size={14} />
            <strong>{active.label}</strong>
          </div>
          <div className="account">
            <span className="avatar">{identity.data.user.email.slice(0, 1).toUpperCase()}</span>
            <span className="account-email">{identity.data.user.email}</span>
            <button className="icon-button" aria-label="Sign out" onClick={() => logout.mutate()}>
              <LogOut size={17} />
            </button>
          </div>
        </header>
        <main>
          <div className="page-heading">
            <div>
              <h1>{active.label}</h1>
              <p>{descriptions[tab]}</p>
            </div>
            <label className="tenant-label">
              Workspace
              <select
                aria-label="Select workspace"
                value={tenant?.id ?? ''}
                onChange={(e) => setTenantID(e.target.value)}
              >
                {identity.data.tenants.map((t) => (
                  <option key={t.id} value={t.id}>
                    {t.name}
                  </option>
                ))}
              </select>
            </label>
          </div>
          <div className="synthetic-banner">
            <ShieldCheck size={17} />
            <span>
              <strong>Synthetic data only.</strong> Prices are configured assumptions. Results do
              not represent observed network savings.
            </span>
          </div>
          {logout.error && <ErrorMessage error={logout.error} />}{' '}
          {!tenant ? (
            <Empty
              title="No workspace access"
              text="Ask your administrator for an invitation to a workspace."
            />
          ) : (
            <TenantContent key={tenant.id} tab={tab} tenant={tenant} identity={identity.data} />
          )}
        </main>
        <footer>
          <span>OptiTrace · Evidence before optimisation</span>
          <span>Google Cloud Identity Platform</span>
        </footer>
      </div>
    </div>
  );
}

export function TenantContent({
  tab,
  ...props
}: Props & {
  tab: string;
}) {
  if (tab === 'playground') return <Playground {...props} />;
  if (tab === 'traces') return <Traces {...props} />;
  if (tab === 'policies') return <Policies {...props} />;
  if (tab === 'benchmarks') return <Benchmarks {...props} />;
  if (tab === 'workspace') return <Administration {...props} />;
  return <Audit {...props} />;
}
