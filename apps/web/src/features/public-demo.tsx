'use client';

import { useRef, useState } from 'react';
import { Github, Play, Download, ShieldCheck, ArrowLeft, FlaskConical } from 'lucide-react';
import { Logo } from '@/components/brand';
import { PreferenceControls } from '@/components/preferences';
import { Panel, ErrorMessage } from '@/components/ui';
import { Button } from '@/components/ui/button';
import { SelectField } from '@/components/select-field';
import { type Evaluation, label, scenarios, strategies } from '@/lib/api';
import { runPublicDemo } from '@/lib/sdk';
import { Trace, TraceRail } from './traces';

export function PublicDemo() {
  const [profile, setProfile] = useState('conditional');
  const [scenario, setScenario] = useState<(typeof scenarios)[number]>('number_mismatch');
  const [strategy, setStrategy] = useState<(typeof strategies)[number]>('staged');
  const [pending, setPending] = useState(false);
  const running = useRef(false);
  const [error, setError] = useState<Error | null>(null);
  const [result, setResult] = useState<
    Omit<Evaluation, 'tenant_id'> & { plan: string; persisted: false }
  >();

  async function run() {
    if (running.current) return;
    running.current = true;
    setPending(true);
    setError(null);
    setResult(undefined);
    try {
      const response = await runPublicDemo({
        body: { profile: profile as 'complete' | 'conditional', scenario, strategy },
      });
      setResult(response);
    } catch (e) {
      setError(e instanceof Error ? e : new Error('The demo could not finish. Try again shortly.'));
    } finally {
      running.current = false;
      setPending(false);
    }
  }
  function download() {
    if (!result) return;
    const url = URL.createObjectURL(
      new Blob([JSON.stringify(result, null, 2)], { type: 'application/json' }),
    );
    const link = document.createElement('a');
    link.href = url;
    link.download = `optitrace-${result.id}.json`;
    link.click();
    setTimeout(() => URL.revokeObjectURL(url), 1000);
  }
  return (
    <div className="public-demo">
      <a className="sr-only focus:not-sr-only" href="#demo-main">
        Skip to sandbox
      </a>
      <header className="demo-header">
        <a href="/" aria-label="OptiTrace workspace">
          <Logo />
        </a>
        <nav aria-label="Sandbox navigation">
          <Button asChild variant="ghost">
            <a href="https://github.com/bandilleee/OptiTrace">
              <Github size={16} />
              Source code
            </a>
          </Button>
          <Button asChild variant="outline">
            <a href="/">
              <ArrowLeft size={16} />
              Workspace
            </a>
          </Button>
          <PreferenceControls />
        </nav>
      </header>
      <main id="demo-main">
        <section className="demo-intro">
          <div>
            <span className="demo-caption">
              <FlaskConical size={16} />
              Public synthetic sandbox
            </span>
            <h1>
              Follow a decision.
              <br />
              See why every call ran.
            </h1>
            <p>
              Change the evidence, run a policy, and inspect the path to an outcome. This runs the
              same Go evaluator used by the workspace, including OPA-backed policy evaluation.
            </p>
          </div>
          <aside>
            <ShieldCheck size={22} />
            <strong>No account needed</strong>
            <p>
              Fixed synthetic scenarios only. Results stay in this page and are not saved to a
              workspace. No real phone numbers or supplier calls.
            </p>
          </aside>
        </section>
        <div className="demo-grid">
          <div>
            <Panel title="Configure a verification">
              <form
                onSubmit={(e) => {
                  e.preventDefault();
                  void run();
                }}
              >
                <div className="form-body demo-fields">
                  <label>
                    Policy profile
                    <SelectField
                      aria-label="Policy profile"
                      value={profile}
                      onValueChange={(v) => {
                        setProfile(v);
                        setResult(undefined);
                        setError(null);
                      }}
                      disabled={pending}
                    >
                      <option value="conditional">Conditional verification</option>
                      <option value="complete">Complete verification</option>
                    </SelectField>
                  </label>
                  <p>
                    {profile === 'complete'
                      ? 'All three capabilities are compulsory, including their explicit failure paths.'
                      : 'Number Verification is compulsory. Optional work can stop when the outcome is established.'}
                  </p>
                  <label>
                    Observation scenario
                    <SelectField
                      aria-label="Observation scenario"
                      value={scenario}
                      onValueChange={(v) => {
                        setScenario(v as typeof scenario);
                        setResult(undefined);
                        setError(null);
                      }}
                      disabled={pending}
                    >
                      {scenarios.map((s) => (
                        <option key={s} value={s}>
                          {label(s)}
                        </option>
                      ))}
                    </SelectField>
                  </label>
                  <label>
                    Execution strategy
                    <SelectField
                      aria-label="Execution strategy"
                      value={strategy}
                      onValueChange={(v) => {
                        setStrategy(v as typeof strategy);
                        setResult(undefined);
                        setError(null);
                      }}
                      disabled={pending}
                    >
                      {strategies.map((s) => (
                        <option key={s} value={s}>
                          {label(s)}
                        </option>
                      ))}
                    </SelectField>
                  </label>
                  <p>
                    Start with number mismatch + staged, then compare full parallel. Both must
                    return the same decision; started calls may differ.
                  </p>
                  <Button type="submit" disabled={pending}>
                    <Play size={16} />
                    {pending ? 'Running verification…' : 'Run verification'}
                  </Button>
                  <p className="demo-small">
                    The free API may take time to wake up. Shared limit: 30 requests/minute and two
                    concurrent evaluations. If busy, wait and try again.
                  </p>
                  <ErrorMessage error={error} />
                </div>
              </form>
            </Panel>
            <Panel title="What to try next">
              <div className="form-body demo-notes">
                <p>
                  <strong>Clear + complete:</strong> all three checks are compulsory.
                </p>
                <p>
                  <strong>Stale or timeout:</strong> unresolved evidence must remain indeterminate.
                </p>
                <p>
                  <strong>Change strategies:</strong> compare calls and synthetic charges for the
                  same inputs. One request is not a benchmark.
                </p>
                <a href="https://github.com/bandilleee/OptiTrace#run-locally-on-windows">
                  Run locally for policies, approvals and reproducible benchmarks
                </a>
              </div>
            </Panel>
          </div>
          <section aria-label="Sandbox execution result" aria-busy={pending}>
            <Panel
              title="Execution result"
              aside={
                result && (
                  <Button variant="ghost" size="sm" onClick={download}>
                    <Download size={14} />
                    Export JSON
                  </Button>
                )
              }
            >
              <div role="status" className="sr-only">
                {pending
                  ? 'Running synthetic verification'
                  : result
                    ? label(result.outcome)
                    : 'Ready to run'}
              </div>
              {result ? (
                <>
                  <div className="demo-plan">
                    <strong>Selected plan</strong>
                    <p>{result.plan}</p>
                    <span>Ephemeral result · not persisted</span>
                  </div>
                  <Trace result={result} />
                </>
              ) : (
                <div className="form-body">
                  <TraceRail />
                  <p>
                    {pending
                      ? 'Fetching synthetic evidence. Please wait…'
                      : 'Run a verification to inspect evidence states, source timestamps, compulsory work and every configured charge.'}
                  </p>
                </div>
              )}
            </Panel>
          </section>
        </div>
        <footer className="demo-footer">
          Synthetic South African scenarios. Freshness: 60 seconds. Change window: 24 hours. Charges
          are demonstration assumptions, not supplier prices or measured savings. Engine elapsed
          time excludes authentication and persistence; it is not an API performance claim.
        </footer>
      </main>
    </div>
  );
}
