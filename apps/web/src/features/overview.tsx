'use client';
import { ArrowUpRight, ShieldCheck } from 'lucide-react';
import { Panel } from '@/components/ui';
import { Button } from '@/components/ui/button';
import { roleWorkflows } from '@/lib/access';
import type { Props } from '@/lib/workspace-types';
export function Overview({
  tenant,
  identity,
  onNavigate,
}: Props & { onNavigate: (view: string) => void }) {
  return (
    <>
      <Panel title="Your workspace" aside={<ShieldCheck size={18} />}>
        <p className="panel-copy">
          Signed in as <strong>{identity.user.email}</strong> in <strong>{tenant.name}</strong>.
          Your assigned roles determine the workflows below. Multiple roles combine their
          permissions.
        </p>
      </Panel>
      <div className="workflow-grid">
        {roleWorkflows
          .filter((item) => tenant.roles.includes(item.role))
          .map((item) => (
            <Panel key={item.role} title={item.title}>
              <p className="panel-copy">{item.description}</p>
              <Button variant="outline" onClick={() => onNavigate(item.view)}>
                {item.action}
                <ArrowUpRight size={16} />
              </Button>
            </Panel>
          ))}
      </div>
      <Panel title="How access works">
        <p className="panel-copy">
          Administrators assign workspace roles. Service principals use scoped API credentials and
          do not sign into this dashboard. Platform operators provision tenants separately and have
          no tenant evidence access by default.
        </p>
      </Panel>
    </>
  );
}
