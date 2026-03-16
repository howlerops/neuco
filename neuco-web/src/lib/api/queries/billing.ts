import { createQuery, createMutation } from '@tanstack/svelte-query';
import { apiClient } from '$lib/api/client';
import type { UsageSummary, SubscriptionResponse } from '$lib/api/types-compat';

// ─── Query Keys ───────────────────────────────────────────────────────────────

export const billingKeys = {
	all: () => ['billing'] as const,
	subscription: (orgId: string) => [...billingKeys.all(), 'subscription', orgId] as const,
	usage: (orgId: string) => [...billingKeys.all(), 'usage', orgId] as const
};

// ─── Hooks ────────────────────────────────────────────────────────────────────

export function useSubscription(orgId: string) {
	return createQuery<SubscriptionResponse>(() => ({
		queryKey: billingKeys.subscription(orgId),
		queryFn: () => apiClient.get<SubscriptionResponse>(`/api/v1/orgs/${orgId}/billing/subscription`),
		enabled: !!orgId,
		staleTime: 60 * 1000
	}));
}

export function useUsage(orgId: string) {
	return createQuery<UsageSummary>(() => ({
		queryKey: billingKeys.usage(orgId),
		queryFn: () => apiClient.get<UsageSummary>(`/api/v1/orgs/${orgId}/billing/usage`),
		enabled: !!orgId,
		staleTime: 30 * 1000
	}));
}

export function useCreateCheckout(orgId: string) {
	return createMutation<{ url: string }, Error, { planTier: string }>(() => ({
		mutationFn: (payload) =>
			apiClient.post<{ url: string }>(`/api/v1/orgs/${orgId}/billing/checkout`, payload)
	}));
}

export function useCreatePortalSession(orgId: string) {
	return createMutation<{ url: string }, Error, void>(() => ({
		mutationFn: () =>
			apiClient.post<{ url: string }>(`/api/v1/orgs/${orgId}/billing/portal`, {})
	}));
}
