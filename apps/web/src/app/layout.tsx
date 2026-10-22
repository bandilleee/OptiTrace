import type { Metadata } from 'next';
import './globals.css';
export const metadata: Metadata = {
  title: 'OptiTrace | Verification workspace',
  description:
    'Synthetic verification orchestration, policy evidence, and reproducible benchmarks.',
};
export default function Layout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en">
      <body>{children}</body>
    </html>
  );
}
