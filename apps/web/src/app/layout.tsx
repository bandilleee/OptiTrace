import type { Metadata } from 'next';
import './globals.css';
import { PreferencesProvider } from '@/components/preferences';
import { TooltipProvider } from '@/components/ui/tooltip';
export const metadata: Metadata = {
  title: 'OptiTrace | Verification workspace',
  description:
    'Synthetic verification orchestration, policy evidence, and reproducible benchmarks.',
};
export default function Layout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en" className="dark" data-theme="dark" suppressHydrationWarning>
      <head>
        <script
          dangerouslySetInnerHTML={{
            __html: `try{const d=localStorage.getItem('optitrace.theme')!=='light';document.documentElement.classList.toggle('dark',d);document.documentElement.dataset.theme=d?'dark':'light'}catch{}`,
          }}
        />
      </head>
      <body>
        <PreferencesProvider>
          <TooltipProvider>{children}</TooltipProvider>
        </PreferencesProvider>
      </body>
    </html>
  );
}
