'use client';
import { loginUrl } from '@/lib/sdk';
import { ShieldCheck, LockKeyhole, Phone, Smartphone, ScanLine } from 'lucide-react';
import { ErrorMessage } from './ui';
import { usePreferences, PreferenceControls } from './preferences';
import { Button } from './ui/button';

export function Logo() {
  const { dark, paused } = usePreferences();
  return (
    <div className="logo">
      {paused ? (
        <>
          <img src="/brand/static.svg" width="38" height="38" alt="" />
          <img
            className="static-wordmark"
            src="/brand/wordmark.svg"
            width="126"
            height="35"
            alt="OptiTrace"
          />
        </>
      ) : (
        <img
          src={`/brand/sidebar-lockup${dark ? '-light' : ''}.svg`}
          width="178"
          height="57"
          alt="OptiTrace"
        />
      )}
    </div>
  );
}
export function BrandLoader({
  label = 'Opening your verification workspace…',
}: {
  label?: string;
}) {
  const { paused } = usePreferences();
  return (
    <div className="brand-loader" role="status">
      <img src={`/brand/${paused ? 'static' : 'loader'}.svg`} width="64" height="64" alt="" />
      <span>{label}</span>
    </div>
  );
}
export function Login({ error }: { error?: string }) {
  return (
    <div className="login-page">
      <section className="login-intro">
        <div className="login-brandbar">
          <Logo />
          <PreferenceControls />
        </div>
        <div className="login-story">
          <h1>
            Every call.
            <br />A clear reason.
          </h1>
          <p>
            Follow the evidence. Understand every decision. A shared workspace for verification
            policies, execution traces, and reproducible comparisons.
          </p>
          <div className="login-evidence">
            {[
              { icon: Phone, name: 'Number Verification', note: 'Establish a number match' },
              { icon: ScanLine, name: 'SIM Swap', note: 'Inspect recent SIM changes' },
              { icon: Smartphone, name: 'Device Swap', note: 'Understand device evidence' },
            ].map((x) => (
              <div key={x.name}>
                <x.icon />
                <span>
                  <strong>{x.name}</strong>
                  <small>{x.note}</small>
                </span>
                <ShieldCheck />
              </div>
            ))}
          </div>
        </div>
        <small>Synthetic South African scenarios. No customer data.</small>
      </section>
      <section className="login-form">
        <div className="login-card">
          <span className="login-symbol">
            <LockKeyhole size={23} />
          </span>
          <h2>Your verification workspace</h2>
          <p>
            Sign in with your invited Google account. Your workspace administrator manages access
            and permissions.
          </p>
          {error && <ErrorMessage error={new Error(error)} />}
          <Button asChild className="google-button" variant="outline">
            <a href={loginUrl()}>
              <svg viewBox="0 0 48 48" aria-hidden="true">
                <path
                  fill="#EA4335"
                  d="M24 9.5c3.54 0 6.71 1.22 9.21 3.6l6.85-6.85C35.9 2.38 30.47 0 24 0 14.62 0 6.51 5.38 2.56 13.22l7.98 6.19C12.43 13.72 17.74 9.5 24 9.5z"
                />
                <path
                  fill="#4285F4"
                  d="M46.98 24.55c0-1.57-.15-3.09-.38-4.55H24v9.02h12.94c-.58 2.96-2.26 5.48-4.78 7.18l7.73 6C44.4 38.06 46.98 31.88 46.98 24.55z"
                />
                <path
                  fill="#FBBC05"
                  d="M10.53 28.59A14.41 14.41 0 0 1 9.75 24c0-1.59.27-3.13.76-4.59l-7.98-6.19A23.87 23.87 0 0 0 0 24c0 3.87.93 7.53 2.56 10.78l7.97-6.19z"
                />
                <path
                  fill="#34A853"
                  d="M24 48c6.48 0 11.93-2.13 15.91-5.8l-7.73-6c-2.15 1.45-4.92 2.3-8.18 2.3-6.26 0-11.57-4.22-13.47-9.91l-7.98 6.19C6.51 42.62 14.62 48 24 48z"
                />
              </svg>
              Continue with Google
            </a>
          </Button>
          <div className="login-note">
            <ShieldCheck size={17} />
            <span>
              Invitation-only access. Sign in with the Google account invited to your workspace.
            </span>
          </div>
          <div className="login-facts">
            <span>Google sign-in</span>
            <span>Workspace permissions enforced on every request</span>
          </div>
        </div>
      </section>
    </div>
  );
}
