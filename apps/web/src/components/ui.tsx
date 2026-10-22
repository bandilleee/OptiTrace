'use client';
import { AlertCircle, Workflow } from 'lucide-react';
import { APIError, label } from '../lib/api';
import { type ReactNode } from 'react';

export function ErrorMessage({ error }: { error: Error | null }) {
  return error ? (
    <div className="error" role="alert">
      <AlertCircle size={18} />
      <span>
        {error.message}
        {error instanceof APIError && error.status === 401 && (
          <>
            {' '}
            <a href="/api/v1/auth/login">Sign in again</a>
          </>
        )}
      </span>
    </div>
  ) : null;
}

export function Empty({ title, text }: { title: string; text: string }) {
  return (
    <div className="empty">
      <div className="empty-symbol">
        <Workflow size={26} />
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
    <section className={'panel ' + className}>
      <div className="panel-heading">
        <h2>{title}</h2>
        {aside}
      </div>
      {children}
    </section>
  );
}

export function Badge({ value }: { value: string }) {
  return (
    <span
      className={
        'badge ' +
        (['verified', 'available', 'approved', 'active', 'completed', 'succeeded'].includes(value)
          ? 'good'
          : ['indeterminate', 'stale', 'timeout', 'pending', 'running', 'queued'].includes(value)
            ? 'warning'
            : [
                  'additional_verification_required',
                  'permission_denied',
                  'failed',
                  'unavailable',
                  'rate_limited',
                ].includes(value)
              ? 'adverse'
              : 'neutral')
      }
    >
      {label(value)}
    </span>
  );
}
