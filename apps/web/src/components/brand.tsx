'use client';
import { Layers3, ChevronRight, ShieldCheck, ArrowRight, KeyRound } from 'lucide-react';
import { ErrorMessage } from './ui';

export function Logo() {
  return (
    <div className="logo">
      <div className="logo-mark">
        <Layers3 size={24} />
      </div>
      <span>
        OptiTrace<span className="logo-dot">.</span>
      </span>
    </div>
  );
}

export function Login({ error }: { error?: string }) {
  return (
    <div className="login-page">
      <section className="login-intro">
        <Logo />
        <div>
          <span className="login-kicker">Verification, made inspectable</span>
          <h1>
            Every call.
            <br />A clear reason.
          </h1>
          <p>
            Explore a safer path through network verification. Define the policy, trace the
            evidence, and measure what actually changes.
          </p>
          <div className="login-rail">
            <span>Number</span>
            <ChevronRight />
            <span>SIM</span>
            <ChevronRight />
            <span>Device</span>
          </div>
        </div>
        <small>Synthetic South African scenarios · Portfolio demonstration</small>
      </section>
      <section className="login-form">
        <div className="login-card">
          <ShieldCheck size={36} />
          <h2>Your verification workspace</h2>
          <p>
            Sign in with your invited Google account. Workspace permissions are managed by your
            administrator.
          </p>
          {error && <ErrorMessage error={new Error(error)} />}
          <a className="button primary google-button" href="/api/v1/auth/login">
            <span className="google-g">G</span>Continue with Google
            <ArrowRight size={18} />
          </a>
          <div className="login-note">
            <KeyRound size={17} />
            <span>
              Invitation-only access. Local development uses a selectable test identity provider.
            </span>
          </div>
        </div>
      </section>
    </div>
  );
}
