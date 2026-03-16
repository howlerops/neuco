import { createQuery, createMutation, useQueryClient } from '@tanstack/svelte-query';
import { apiClient } from '$lib/api/client';
import type {
	FeatureCandidate,
	UpdateCandidateStatusPayload,
	PageParams,
	PaginatedResponse
} from '$lib/api/types-compat';

// ─── Backend response shapes ──────────────────────────────────────────────────
// GET /api/v1/projects/{projectId}/candidates returns:
//   { candidates: FeatureCandidate[], total: number }
// POST …/candidates/refresh returns:
//   { pipelineRunId: string }  (a job was enqueued, not a list of candidates)

interface CandidatesBackendResponse {
	candidates: FeatureCandidate[];
	total: number;
}

interface RefreshCandidatesResponse {
	pipelineRunId: string;
}

// ─── Query Keys ───────────────────────────────────────────────────────────────

export const candidateKeys = {
	all: (projectId: string) => ['projects', projectId, 'candidates'] as const,
	lists: (projectId: string, params?: PageParams) =>
		[...candidateKeys.all(projectId), 'list', params] as const,
	detail: (projectId: string, candidateId: string) =>
		[...candidateKeys.all(projectId), 'detail', candidateId] as const
};

// ─── Hooks ────────────────────────────────────────────────────────────────────

export function useCandidates(projectId: string, params?: PageParams) {
	return createQuery<PaginatedResponse<FeatureCandidate>>(() => ({
		queryKey: candidateKeys.lists(projectId, params),
		queryFn: async () => {
			const page = params?.page ?? 1;
			const pageSize = params?.pageSize ?? 50;
			const searchParams = new URLSearchParams();
			searchParams.set('limit', String(pageSize));
			searchParams.set('offset', String((page - 1) * pageSize));
			const qs = searchParams.toString();
			const raw = await apiClient.get<CandidatesBackendResponse>(
				`/api/v1/projects/${projectId}/candidates${qs ? `?${qs}` : ''}`
			);
			const items = raw.candidates ?? [];
			const total = raw.total ?? 0;
			const totalPages = Math.max(1, Math.ceil(total / pageSize));
			return {
				data: items,
				total,
				page,
				pageSize,
				totalPages
			} satisfies PaginatedResponse<FeatureCandidate>;
		},
		enabled: !!projectId,
		staleTime: 30 * 1000
	}));
}

export function useRefreshCandidates(projectId: string) {
	const queryClient = useQueryClient();
	// The backend enqueues a job and returns { pipeline_run_id }, not the
	// candidates themselves. Return the response as-is.
	return createMutation<RefreshCandidatesResponse, Error, void>(() => ({
		mutationFn: () =>
			apiClient.post<RefreshCandidatesResponse>(
				`/api/v1/projects/${projectId}/candidates/refresh`
			),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: candidateKeys.all(projectId) });
		}
	}));
}

export function useUpdateCandidateStatus(projectId: string, candidateId: string) {
	const queryClient = useQueryClient();
	return createMutation<FeatureCandidate, Error, UpdateCandidateStatusPayload>(() => ({
		mutationFn: (payload: UpdateCandidateStatusPayload) =>
			apiClient.patch<FeatureCandidate>(
				`/api/v1/projects/${projectId}/candidates/${candidateId}/status`,
				payload
			),
		onSuccess: (updated: FeatureCandidate) => {
			queryClient.setQueryData(candidateKeys.detail(projectId, candidateId), updated);
			queryClient.invalidateQueries({ queryKey: candidateKeys.lists(projectId) });
		}
	}));
}
