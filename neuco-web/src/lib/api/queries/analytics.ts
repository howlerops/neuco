import { createQuery } from '@tanstack/svelte-query';
import { apiClient } from '$lib/api/client';
import type { OrgAnalytics } from '$lib/api/types';

export const analyticsKeys = {
	all: (orgId: string) => ['orgs', orgId, 'analytics'] as const,
	byRange: (orgId: string, days: number) =>
		[...analyticsKeys.all(orgId), days] as const
};

export function useOrgAnalytics(orgId: string, days: number = 30) {
	return createQuery<OrgAnalytics>(() => ({
		queryKey: analyticsKeys.byRange(orgId, days),
		queryFn: () =>
			apiClient.get<OrgAnalytics>(
				`/api/v1/orgs/${orgId}/analytics?days=${days}`
			),
		enabled: !!orgId,
		staleTime: 2 * 60 * 1000,
		refetchInterval: 2 * 60 * 1000
	}));
}
