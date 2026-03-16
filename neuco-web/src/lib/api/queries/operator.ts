import { createQuery, createMutation, useQueryClient } from '@tanstack/svelte-query';
import { apiClient as api } from '$lib/api/client';
import type { Organization, User } from '$lib/api/types-compat';

interface OperatorOrg extends Organization {
	memberCount: number;
	projectCount: number;
}

interface HealthStatus {
	status: string;
	checks: {
		database: string;
		queue: string;
	};
	timestamp: string;
}

export function useOperatorOrgs() {
	return createQuery(() => ({
		queryKey: ['operator', 'orgs'],
		queryFn: () => api.get<OperatorOrg[]>('/operator/orgs')
	}));
}

export function useOperatorOrg(orgId: string) {
	return createQuery(() => ({
		queryKey: ['operator', 'orgs', orgId],
		queryFn: () => api.get<OperatorOrg>(`/operator/orgs/${orgId}`),
		enabled: !!orgId
	}));
}

export function useOperatorUsers() {
	return createQuery(() => ({
		queryKey: ['operator', 'users'],
		queryFn: () => api.get<User[]>('/operator/users')
	}));
}

export function useOperatorHealth() {
	return createQuery(() => ({
		queryKey: ['operator', 'health'],
		queryFn: () => api.get<HealthStatus>('/operator/health'),
		refetchInterval: 30000
	}));
}

// ─── Feature Flags ───────────────────────────────────────────────────────────

export interface FeatureFlag {
	key: string;
	enabled: boolean;
	description: string;
	metadata: Record<string, unknown>;
	updatedAt: string;
	updatedBy: string | null;
}

export function useOperatorFlags() {
	return createQuery(() => ({
		queryKey: ['operator', 'flags'],
		queryFn: () => api.get<FeatureFlag[]>('/operator/flags')
	}));
}

export function useToggleFlag() {
	const queryClient = useQueryClient();
	return createMutation<FeatureFlag, Error, { key: string; enabled: boolean }>(() => ({
		mutationFn: ({ key, enabled }: { key: string; enabled: boolean }) =>
			api.patch<FeatureFlag>(`/operator/flags/${key}`, { enabled }),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['operator', 'flags'] });
		}
	}));
}
