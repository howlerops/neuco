import { createQuery, createMutation, useQueryClient } from '@tanstack/svelte-query';
import { apiClient } from '$lib/api/client';
import type { Organization, CreateOrgPayload, UpdateOrgPayload } from '$lib/api/types-compat';

// ─── Query Keys ───────────────────────────────────────────────────────────────

export const orgKeys = {
	all: () => ['organizations'] as const,
	lists: () => [...orgKeys.all(), 'list'] as const,
	detail: (id: string) => [...orgKeys.all(), 'detail', id] as const,
	bySlug: (slug: string) => [...orgKeys.all(), 'slug', slug] as const
};

// ─── Hooks ────────────────────────────────────────────────────────────────────

export function useOrgs() {
	return createQuery<Organization[]>(() => ({
		queryKey: orgKeys.lists(),
		queryFn: () => apiClient.get<Organization[]>('/api/v1/orgs'),
		staleTime: 2 * 60 * 1000
	}));
}

export function useOrg(idOrSlug: string) {
	return createQuery<Organization>(() => ({
		queryKey: orgKeys.bySlug(idOrSlug),
		queryFn: () => apiClient.get<Organization>(`/api/v1/orgs/${idOrSlug}`),
		enabled: !!idOrSlug,
		staleTime: 2 * 60 * 1000
	}));
}

export function useCreateOrg() {
	const queryClient = useQueryClient();
	return createMutation<Organization, Error, CreateOrgPayload>(() => ({
		mutationFn: (payload: CreateOrgPayload) =>
			apiClient.post<Organization>('/api/v1/orgs', payload),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: orgKeys.lists() });
		}
	}));
}

export function useUpdateOrg(orgId: string) {
	const queryClient = useQueryClient();
	return createMutation<Organization, Error, UpdateOrgPayload>(() => ({
		mutationFn: (payload: UpdateOrgPayload) =>
			apiClient.patch<Organization>(`/api/v1/orgs/${orgId}`, payload),
		onSuccess: (updated: Organization) => {
			queryClient.setQueryData(orgKeys.detail(orgId), updated);
			queryClient.invalidateQueries({ queryKey: orgKeys.lists() });
		}
	}));
}
