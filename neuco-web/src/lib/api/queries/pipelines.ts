import { createQuery, createMutation, useQueryClient } from '@tanstack/svelte-query';
import { apiClient } from '$lib/api/client';
import type { PipelineRun, ProjectStats, PageParams, PaginatedResponse } from '$lib/api/types';

// ─── Backend response shapes ──────────────────────────────────────────────────
// GET /api/v1/projects/{projectId}/pipelines returns:
//   { runs: PipelineRun[], total: number }
// The query hook normalises this into PaginatedResponse<PipelineRun>.

interface PipelinesBackendResponse {
	runs: PipelineRun[];
	total: number;
}

// ─── Query Keys ───────────────────────────────────────────────────────────────

export const pipelineKeys = {
	all: (projectId: string) => ['projects', projectId, 'pipelines'] as const,
	lists: (projectId: string, params?: PageParams) =>
		[...pipelineKeys.all(projectId), 'list', params] as const,
	detail: (projectId: string, pipelineId: string) =>
		[...pipelineKeys.all(projectId), 'detail', pipelineId] as const,
	stats: (projectId: string) => ['projects', projectId, 'stats'] as const
};

// ─── Hooks ────────────────────────────────────────────────────────────────────

export function usePipelines(projectId: string, params?: PageParams) {
	return createQuery<PaginatedResponse<PipelineRun>>(() => ({
		queryKey: pipelineKeys.lists(projectId, params),
		queryFn: async () => {
			const page = params?.page ?? 1;
			const pageSize = params?.pageSize ?? 20;
			const searchParams = new URLSearchParams();
			searchParams.set('limit', String(pageSize));
			searchParams.set('offset', String((page - 1) * pageSize));
			const qs = searchParams.toString();
			const raw = await apiClient.get<PipelinesBackendResponse>(
				`/api/v1/projects/${projectId}/pipelines${qs ? `?${qs}` : ''}`
			);
			const items = raw.runs ?? [];
			const total = raw.total ?? 0;
			const totalPages = Math.max(1, Math.ceil(total / pageSize));
			return {
				data: items,
				total,
				page,
				pageSize,
				totalPages
			} satisfies PaginatedResponse<PipelineRun>;
		},
		enabled: !!projectId,
		staleTime: 30 * 1000,
		refetchInterval: 15 * 1000
	}));
}

export function usePipeline(projectId: string, pipelineId: string) {
	return createQuery<PipelineRun>(() => ({
		queryKey: pipelineKeys.detail(projectId, pipelineId),
		queryFn: () =>
			apiClient.get<PipelineRun>(
				`/api/v1/projects/${projectId}/pipelines/${pipelineId}`
			),
		enabled: !!projectId && !!pipelineId,
		staleTime: 10 * 1000,
		refetchInterval: 10 * 1000
	}));
}

export function useRetryPipeline(projectId: string) {
	const queryClient = useQueryClient();
	return createMutation<PipelineRun, Error, string>(() => ({
		mutationFn: (pipelineId: string) =>
			apiClient.post<PipelineRun>(
				`/api/v1/projects/${projectId}/pipelines/${pipelineId}/retry`
			),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: pipelineKeys.all(projectId) });
		}
	}));
}

export function useProjectStats(projectId: string) {
	return createQuery<ProjectStats>(() => ({
		queryKey: pipelineKeys.stats(projectId),
		queryFn: () =>
			apiClient.get<ProjectStats>(`/api/v1/projects/${projectId}/stats`),
		enabled: !!projectId,
		staleTime: 60 * 1000,
		refetchInterval: 60 * 1000
	}));
}
