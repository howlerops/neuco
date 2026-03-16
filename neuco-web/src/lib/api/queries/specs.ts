import { createQuery, createMutation, useQueryClient } from '@tanstack/svelte-query';
import { apiClient } from '$lib/api/client';
import type { Spec, UpdateSpecPayload } from '$lib/api/types-compat';

// ─── Query Keys ───────────────────────────────────────────────────────────────

export const specKeys = {
	all: (projectId: string) => ['projects', projectId, 'specs'] as const,
	byCandidate: (projectId: string, candidateId: string) =>
		[...specKeys.all(projectId), 'candidate', candidateId] as const,
	detail: (projectId: string, specId: string) =>
		[...specKeys.all(projectId), 'detail', specId] as const
};

// ─── Hooks ────────────────────────────────────────────────────────────────────

export function useSpec(projectId: string, specId: string) {
	return createQuery<Spec>(() => ({
		queryKey: specKeys.detail(projectId, specId),
		queryFn: () => apiClient.get<Spec>(`/api/v1/projects/${projectId}/specs/${specId}`),
		enabled: !!projectId && !!specId,
		staleTime: 60 * 1000
	}));
}

export function useSpecByCandidate(projectId: string, candidateId: string) {
	return createQuery<Spec>(() => ({
		queryKey: specKeys.byCandidate(projectId, candidateId),
		queryFn: () =>
			apiClient.get<Spec>(
				`/api/v1/projects/${projectId}/candidates/${candidateId}/spec`
			),
		enabled: !!projectId && !!candidateId,
		staleTime: 60 * 1000
	}));
}

export function useGenerateSpec(projectId: string, candidateId: string) {
	const queryClient = useQueryClient();
	return createMutation<Spec, Error, void>(() => ({
		mutationFn: () =>
			apiClient.post<Spec>(
				`/api/v1/projects/${projectId}/candidates/${candidateId}/spec/generate`
			),
		onSuccess: (spec: Spec) => {
			queryClient.setQueryData(specKeys.byCandidate(projectId, candidateId), spec);
			queryClient.invalidateQueries({ queryKey: specKeys.all(projectId) });
		}
	}));
}

export function useUpdateSpec(projectId: string, specId: string) {
	const queryClient = useQueryClient();
	return createMutation<Spec, Error, UpdateSpecPayload>(() => ({
		mutationFn: (payload: UpdateSpecPayload) =>
			apiClient.patch<Spec>(`/api/v1/projects/${projectId}/specs/${specId}`, payload),
		onSuccess: (updated: Spec) => {
			queryClient.setQueryData(specKeys.detail(projectId, specId), updated);
		}
	}));
}
