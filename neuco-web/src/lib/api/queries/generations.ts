import { createQuery, createMutation, useQueryClient } from '@tanstack/svelte-query';
import { apiClient } from '$lib/api/client';
import type { Generation, PageParams, PaginatedResponse } from '$lib/api/types';

// ─── Backend response shapes ──────────────────────────────────────────────────
// GET /api/v1/projects/{projectId}/generations returns:
//   { generations: Generation[], total: number }
// POST …/generations returns { generationId, pipelineRunId } (a job enqueue
// response), not a Generation object.

interface GenerationsBackendResponse {
	generations: Generation[];
	total: number;
}

// ─── Query Keys ───────────────────────────────────────────────────────────────

export const generationKeys = {
	all: (projectId: string) => ['projects', projectId, 'generations'] as const,
	lists: (projectId: string, params?: PageParams) =>
		[...generationKeys.all(projectId), 'list', params] as const,
	detail: (projectId: string, generationId: string) =>
		[...generationKeys.all(projectId), 'detail', generationId] as const
};

// ─── Hooks ────────────────────────────────────────────────────────────────────

export function useGenerations(projectId: string, params?: PageParams) {
	return createQuery<PaginatedResponse<Generation>>(() => ({
		queryKey: generationKeys.lists(projectId, params),
		queryFn: async () => {
			const page = params?.page ?? 1;
			const pageSize = params?.pageSize ?? 20;
			const searchParams = new URLSearchParams();
			searchParams.set('limit', String(pageSize));
			searchParams.set('offset', String((page - 1) * pageSize));
			const qs = searchParams.toString();
			const raw = await apiClient.get<GenerationsBackendResponse>(
				`/api/v1/projects/${projectId}/generations${qs ? `?${qs}` : ''}`
			);
			const items = raw.generations ?? [];
			const total = raw.total ?? 0;
			const totalPages = Math.max(1, Math.ceil(total / pageSize));
			return {
				data: items,
				total,
				page,
				pageSize,
				totalPages
			} satisfies PaginatedResponse<Generation>;
		},
		enabled: !!projectId,
		staleTime: 30 * 1000
	}));
}

export function useGeneration(projectId: string, generationId: string) {
	return createQuery<Generation>(() => ({
		queryKey: generationKeys.detail(projectId, generationId),
		queryFn: () =>
			apiClient.get<Generation>(
				`/api/v1/projects/${projectId}/generations/${generationId}`
			),
		enabled: !!projectId && !!generationId,
		staleTime: 30 * 1000
	}));
}

export function useGenerateCode(projectId: string) {
	const queryClient = useQueryClient();
	return createMutation<Generation, Error, { specId: string }>(() => ({
		mutationFn: ({ specId }: { specId: string }) =>
			apiClient.post<Generation>(`/api/v1/projects/${projectId}/generations`, {
				specId
			}),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: generationKeys.all(projectId) });
		}
	}));
}
