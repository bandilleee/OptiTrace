'use client';
import { AlertCircle, Workflow } from 'lucide-react';
import { APIError, label } from '@/lib/api';
import { type ReactNode } from 'react';
import { Card, CardHeader } from './ui/card';
import { Badge as StatusBadge } from './ui/badge';
import { Alert, AlertDescription } from './ui/alert';

export function ErrorMessage({ error }: { error: Error | null }) {
  return error ? (
    <Alert variant="destructive" className="error">
      <AlertCircle size={18} />
      <AlertDescription>
        {error.message}
        {error instanceof APIError && error.status === 401 && (
          <>
            {' '}
            <a href="/api/v1/auth/login">Sign in again</a>
          </>
        )}
      </AlertDescription>
    </Alert>
  ) : null;
}
export function Empty({ title, text }: { title: string; text: string }) {
  return (
    <div className="empty">
      <div className="empty-symbol">
        <Workflow size={25} />
      </div>
      <h3>{title}</h3>
      <p>{text}</p>
    </div>
  );
}
export function Panel({
  title,
  aside,
  children,
  className = '',
}: {
  title: string;
  aside?: ReactNode;
  children: ReactNode;
  className?: string;
}) {
  return (
    <Card className={'panel ' + className}>
      <CardHeader className="panel-heading">
        <h2>{title}</h2>
        {aside && <div className="panel-aside">{aside}</div>}
      </CardHeader>
      {children}
    </Card>
  );
}
export function Badge({ value }: { value: string }) {
  const tone = [
    'verified',
    'available',
    'valid',
    'approved',
    'active',
    'completed',
    'succeeded',
  ].includes(value)
    ? 'good'
    : ['stale', 'timeout', 'pending', 'queued', 'revoked'].includes(value)
      ? 'warning'
      : [
            'additional_verification_required',
            'adverse',
            'permission_denied',
            'failed',
            'source_failure',
            'rate_limited',
          ].includes(value)
        ? 'adverse'
        : ['indeterminate', 'unavailable'].includes(value)
          ? 'uncertain'
          : value === 'submitted' || value === 'running'
            ? 'info'
            : 'neutral';
  return (
    <StatusBadge variant="secondary" className={'badge ' + tone}>
      {label(value)}
    </StatusBadge>
  );
}
