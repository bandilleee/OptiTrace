'use client';
import { useQuery } from '@tanstack/react-query';
import { api } from './api';

export function useItems<T>(
  tenant: string,
  resource: string,
  enabled = true,
  refetchInterval: number | false = false,
) {
  return useQuery({
    enabled,
    refetchInterval,
    queryKey: [tenant, resource],
    queryFn: () =>
      api<{
        items: T[];
      }>(`/tenants/${tenant}/${resource}`),
  });
}
