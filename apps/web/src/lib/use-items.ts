'use client';
import { useQuery } from '@tanstack/react-query';
import { api } from './api';

export function useItems<T>(tenant: string, resource: string, enabled = true) {
  return useQuery({
    enabled,
    queryKey: [tenant, resource],
    queryFn: () =>
      api<{
        items: T[];
      }>(`/tenants/${tenant}/${resource}`),
  });
}
