import { describe, it, expect } from 'vitest';
import { nameError, emailError, integerError } from '../src/lib/validation';
import { canView } from '../src/lib/access';
describe('form boundaries', () => {
  it('rejects whitespace, controls and names exceeding the API byte budget', () => {
    expect(nameError('  ')).toBeTruthy();
    expect(nameError('a\nb')).toBeTruthy();
    expect(nameError('é'.repeat(60))).toBeUndefined();
    expect(nameError('é'.repeat(61))).toBeTruthy();
  });
  it('matches conventional invitation email syntax', () => {
    for (const email of ['person+demo@example.com', ' person@optitrace.test '])
      expect(emailError(email)).toBeUndefined();
    for (const email of ['a@@b.com', 'a@b', 'Name <a@b.com>', 'a..b@example.com', 'a@-example.com'])
      expect(emailError(email)).toBeTruthy();
  });
  it('rejects blanks, fractions, unsafe seeds and out-of-range counts', () => {
    for (const value of ['', '2.5', '1001', '1']) expect(integerError(value, 2, 1000)).toBeTruthy();
    expect(integerError('2', 2, 1000)).toBeUndefined();
    expect(
      integerError('9007199254740992', Number.MIN_SAFE_INTEGER, Number.MAX_SAFE_INTEGER),
    ).toBeTruthy();
  });
  it('combines roles without elevating tenant administrators to evidence access', () => {
    expect(canView('playground', ['tenant_admin'])).toBe(false);
    expect(canView('audit', ['integration_developer'])).toBe(false);
    expect(canView('policies', ['policy_author'])).toBe(true);
    expect(canView('workspace', ['auditor', 'tenant_admin'])).toBe(true);
    expect(canView('overview', ['platform_operator'])).toBe(false);
  });
});
