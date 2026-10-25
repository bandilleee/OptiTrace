import type { Metadata } from 'next';
import { PublicDemo } from '@/features/public-demo';

export const metadata: Metadata = {
  title: 'Try OptiTrace | Public synthetic sandbox',
  description:
    'Run the real policy evaluator on synthetic verification scenarios. No sign-in needed.',
};

export default function DemoPage() {
  return <PublicDemo />;
}
