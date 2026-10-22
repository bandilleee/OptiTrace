'use client';
import { canView } from '@/lib/access';
import { Overview } from '@/features/overview';
import { LayoutDashboard } from 'lucide-react';
import { label, APIError } from '@/lib/api';
import {
  SlidersHorizontal,
  Route,
  Layers,
  ChartNoAxesCombined,
  Users,
  ClipboardList,
  ChevronRight,
  LogOut,
  FlaskConical,
  Menu,
  BookOpen,
  ShieldCheck,
} from 'lucide-react';
import { useState } from 'react';
import { QueryClient, QueryClientProvider, useQuery, useMutation } from '@tanstack/react-query';
import { getSession, logout as endSession } from '@/lib/sdk';
import { Logo, Login, BrandLoader } from './brand';
import { PreferenceControls } from './preferences';
import { ErrorMessage, Empty } from './ui';
import { Button } from './ui/button';
import { SelectField } from './select-field';
import { Sheet, SheetContent, SheetTitle, SheetTrigger } from './ui/sheet';
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogTrigger } from './ui/dialog';
import { Tooltip, TooltipContent, TooltipTrigger } from './ui/tooltip';
import { type Props } from '@/lib/workspace-types';
import { Playground } from '@/features/playground';
import { Traces } from '@/features/traces';
import { Policies } from '@/features/policies';
import { Benchmarks } from '@/features/benchmarks';
import { Administration } from '@/features/administration';
import { Audit } from '@/features/audit';

export const tabs = [
  { id: 'overview', label: 'Overview', icon: LayoutDashboard },
  { id: 'playground', label: 'Playground', icon: SlidersHorizontal },
  { id: 'traces', label: 'Execution traces', icon: Route },
  { id: 'policies', label: 'Policy library', icon: Layers },
  { id: 'benchmarks', label: 'Benchmarks', icon: ChartNoAxesCombined },
  { id: 'workspace', label: 'Administration', icon: Users },
  { id: 'audit', label: 'Audit history', icon: ClipboardList },
];
const headings: Record<string, string> = {
  overview: 'Workspace overview',
  playground: 'Verification playground',
  traces: 'Execution traces',
  policies: 'Policy library',
  benchmarks: 'Benchmarks',
  workspace: 'Workspace administration',
  audit: 'Audit history',
};
export const descriptions: Record<string, string> = {
  overview: 'Your roles. Your workflows. One accountable workspace.',
  playground: 'Follow the evidence. Understand every decision.',
  traces: 'A lasting record of every decision and the evidence behind it.',
  policies: 'Versioned decisions. Explicit obligations. Independent approval.',
  benchmarks: 'Compare execution strategies on the same synthetic observations.',
  workspace: 'Manage who can work here and how they connect.',
  audit: 'Changes, decisions, and the people behind them.',
};

export function AssumptionsDialog() {
  return (
    <Dialog>
      <DialogTrigger asChild>
        <Button variant="link" className="assumptions-link">
          View assumptions
        </Button>
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Versioned demonstration assumptions</DialogTitle>
        </DialogHeader>
        <div className="guide-copy">
          <p>
            Scenarios use synthetic South African data. No customer numbers or live network adapters
            are used.
          </p>
          <ul>
            <li>Evidence freshness: 60 seconds at the fixed evaluation observation time.</li>
            <li>
              A number mismatch or SIM/device change within 24 hours requires additional
              verification.
            </li>
            <li>
              Complete verification requires all three capabilities. Conditional verification always
              requires Number Verification.
            </li>
            <li>
              Every started billable call is charged, including failed or cancelled calls. Skipped
              calls are not started.
            </li>
          </ul>
          <p>
            Trace charges come from the engine configuration. They are synthetic assumptions, not
            supplier quotes or actual cloud expenditure. Runtime cost is an estimate. No commercial
            savings are established.
          </p>
        </div>
      </DialogContent>
    </Dialog>
  );
}
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
  const [requestedTab, setTab] = useState('overview');
  const [tenantID, setTenantID] = useState('');
  const [menuOpen, setMenuOpen] = useState(false);
  const [traceID, setTraceID] = useState('');
  const identity = useQuery({ queryKey: ['identity'], queryFn: () => getSession() });
  const tenant = identity.data?.tenants.find((t) => t.id === tenantID) ?? identity.data?.tenants[0];
  const logout = useMutation({
    mutationFn: () => endSession({ csrf: identity.data?.csrf_token }),
    onSuccess: () => window.location.assign('/'),
  });
  if (identity.isPending)
    return (
      <div className="loading-page">
        <Logo />
        <BrandLoader />
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
  const tab = canView(requestedTab, tenant?.roles ?? []) ? requestedTab : 'overview';
  const active = tabs.find((t) => t.id === tab)!;
  function navigate(id: string) {
    setTab(id);
    setMenuOpen(false);
    if (id !== 'traces') setTraceID('');
    window.scrollTo({ top: 0, behavior: 'instant' });
  }
  const navigation = (
    <>
      <Logo />
      <div className="workspace-picker">
        <span className="workmark">{tenant?.name.slice(0, 1) ?? 'O'}</span>
        <label>
          <span className="sr-only">Select workspace</span>
          <SelectField
            aria-label="Select workspace"
            value={tenant?.id ?? ''}
            onValueChange={(value) => {
              setTenantID(value);
              setTraceID('');
            }}
          >
            {identity.data.tenants.map((t) => (
              <option key={t.id} value={t.id}>
                {t.name}
              </option>
            ))}
          </SelectField>
          <small>South Africa · Synthetic</small>
        </label>
      </div>
      <div className="nav-label">Verification</div>
      <nav aria-label="Verification navigation">
        {tabs
          .slice(0, 5)
          .filter((item) => canView(item.id, tenant?.roles ?? []))
          .map((item) => (
            <Button
              key={item.id}
              variant="ghost"
              aria-current={tab === item.id ? 'page' : undefined}
              className={'nav-item ' + (tab === item.id ? 'active' : '')}
              onClick={() => navigate(item.id)}
            >
              <item.icon size={18} />
              <span>{item.label}</span>
            </Button>
          ))}
      </nav>
      <div className="nav-label workspace-nav-label">Workspace</div>
      <nav aria-label="Workspace navigation">
        {tabs
          .slice(5)
          .filter((item) => canView(item.id, tenant?.roles ?? []))
          .map((item) => (
            <Button
              key={item.id}
              variant="ghost"
              aria-current={tab === item.id ? 'page' : undefined}
              className={'nav-item ' + (tab === item.id ? 'active' : '')}
              onClick={() => navigate(item.id)}
            >
              <item.icon size={18} />
              <span>{item.label}</span>
            </Button>
          ))}
      </nav>
      <div className="sidebar-bottom">
        <div className="sandbox">
          <FlaskConical size={18} />
          <strong>A safe place to explore</strong>
          <p>Synthetic scenarios. Persisted decisions. No live network verification calls.</p>
        </div>
        <div className="profile">
          <span className="avatar">{identity.data.user.email.slice(0, 2).toUpperCase()}</span>
          <div>
            <strong title={identity.data.user.email}>{identity.data.user.email}</strong>
            <small>{tenant?.roles.map(label).join(' · ') || 'No workspace role'}</small>
          </div>
        </div>
      </div>
    </>
  );
  return (
    <div className="workspace-shell">
      <a href="#main-content" className="skip-link">
        Skip to content
      </a>
      <aside className="sidebar">{navigation}</aside>
      <div className="main-shell">
        <header className="topbar">
          <div className="breadcrumb">
            <Sheet open={menuOpen} onOpenChange={setMenuOpen}>
              <SheetTrigger asChild>
                <Button
                  variant="outline"
                  size="icon"
                  className="mobile-menu"
                  aria-label="Open navigation"
                >
                  <Menu />
                </Button>
              </SheetTrigger>
              <SheetContent side="left" className="mobile-sidebar">
                <SheetTitle className="sr-only">Workspace navigation</SheetTitle>
                {navigation}
              </SheetContent>
            </Sheet>
            <span className="breadcrumb-section">
              {['workspace', 'audit'].includes(tab) ? 'Workspace' : 'Verification'}
            </span>
            <ChevronRight size={13} />
            <strong>{active.label}</strong>
          </div>
          <div className="topbar-actions">
            <PreferenceControls />
            <Dialog>
              <DialogTrigger asChild>
                <Button variant="ghost" size="sm" className="quick-guide">
                  <BookOpen />
                  Quick guide
                </Button>
              </DialogTrigger>
              <DialogContent>
                <DialogHeader>
                  <DialogTitle>Your verification workspace</DialogTitle>
                </DialogHeader>
                <div className="guide-copy">
                  <p>
                    Choose an approved policy, a synthetic scenario, and an execution strategy. Run
                    a verification to inspect the actual engine decision and its persisted evidence.
                  </p>
                  <p>
                    Authors submit policy drafts; a different reviewer approves and activates them.
                    Integration developers run evaluations and benchmarks. Administrators manage
                    access. Auditors read evidence.
                  </p>
                  <p>
                    Use Execution traces to revisit earlier runs and Benchmarks to compare
                    strategies. Missing or stale evidence never counts as clear.
                  </p>
                </div>
              </DialogContent>
            </Dialog>
            <Tooltip>
              <TooltipTrigger asChild>
                <Button
                  variant="ghost"
                  size="icon"
                  aria-label="Sign out"
                  disabled={logout.isPending}
                  onClick={() => logout.mutate()}
                >
                  <LogOut size={17} />
                </Button>
              </TooltipTrigger>
              <TooltipContent>Sign out</TooltipContent>
            </Tooltip>
            <span className="avatar account-avatar" title={identity.data.user.email}>
              {identity.data.user.email.slice(0, 2).toUpperCase()}
            </span>
          </div>
        </header>
        <main id="main-content" tabIndex={-1}>
          <div className="page-heading">
            <div>
              <h1>{headings[tab]}</h1>
              <p>{descriptions[tab]}</p>
            </div>
            <span className="workspace-status">
              <ShieldCheck size={14} />
              Invitation-only workspace
            </span>
          </div>
          <div className="synthetic-banner">
            <span>
              <FlaskConical size={16} />
              <span>
                <strong>Synthetic sandbox.</strong> Scenarios and configured prices are
                demonstration assumptions.
              </span>
            </span>
            <AssumptionsDialog />
          </div>
          <ErrorMessage error={logout.error} />
          {!tenant ? (
            <Empty
              title="No workspace access"
              text="Ask your administrator for an invitation to a workspace."
            />
          ) : (
            <TenantContent
              key={tenant.id}
              tab={tab}
              onNavigate={navigate}
              tenant={tenant}
              identity={identity.data}
              traceID={traceID}
              onOpenTrace={(id) => {
                setTraceID(id);
                setTab('traces');
                window.scrollTo({ top: 0 });
              }}
            />
          )}
          <footer>
            <span>OptiTrace · Evidence before optimisation</span>
            <span>Synthetic data only. No live network services.</span>
          </footer>
        </main>
      </div>
    </div>
  );
}
export function TenantContent({
  tab,
  traceID,
  onOpenTrace,
  onNavigate,
  ...props
}: Props & {
  tab: string;
  traceID: string;
  onOpenTrace: (id: string) => void;
  onNavigate: (view: string) => void;
}) {
  if (!canView(tab, props.tenant.roles))
    return (
      <Empty
        title="No workflow access"
        text="Ask your workspace administrator to assign an appropriate role."
      />
    );
  if (tab === 'overview') return <Overview {...props} onNavigate={onNavigate} />;
  if (tab === 'playground') return <Playground {...props} onOpenTrace={onOpenTrace} />;
  if (tab === 'traces') return <Traces {...props} initialTraceID={traceID} />;
  if (tab === 'policies') return <Policies {...props} />;
  if (tab === 'benchmarks') return <Benchmarks {...props} />;
  if (tab === 'workspace') return <Administration {...props} />;
  return <Audit {...props} />;
}
