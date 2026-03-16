import { createQuery, createMutation, useQueryClient } from '@tanstack/svelte-query';
import { apiClient } from '$lib/api/client';
import type {
	ProjectContext,
	CreateProjectContextPayload,
	UpdateProjectContextPayload,
	PageParams,
	PaginatedResponse
} from '$lib/api/types-compat';

// ─── Backend response shapes ──────────────────────────────────────────────────

interface ContextsBackendResponse {
	contexts: ProjectContext[];
	total: number;
}

// ─── Query Keys ───────────────────────────────────────────────────────────────

export const contextKeys = {
	all: (projectId: string) => ['projects', projectId, 'contexts'] as const,
	lists: (projectId: string, params?: PageParams & { category?: string }) =>
		[...contextKeys.all(projectId), 'list', params] as const,
	detail: (projectId: string, contextId: string) =>
		[...contextKeys.all(projectId), 'detail', contextId] as const
};

// ─── Hooks ────────────────────────────────────────────────────────────────────

export function useProjectContexts(
	projectId: string,
	params?: PageParams & { category?: string }
) {
	return createQuery<PaginatedResponse<ProjectContext>>(() => ({
		queryKey: contextKeys.lists(projectId, params),
		queryFn: async () => {
			const page = params?.page ?? 1;
			const pageSize = params?.pageSize ?? 50;
			const searchParams = new URLSearchParams();
			searchParams.set('limit', String(pageSize));
			searchParams.set('offset', String((page - 1) * pageSize));
			if (params?.category) {
				searchParams.set('category', params.category);
			}
			const qs = searchParams.toString();
			const raw = await apiClient.get<ContextsBackendResponse>(
				`/api/v1/projects/${projectId}/contexts${qs ? `?${qs}` : ''}`
			);
			const items = raw.contexts ?? [];
			const total = raw.total ?? 0;
			const totalPages = Math.max(1, Math.ceil(total / pageSize));
			return {
				data: items,
				total,
				page,
				pageSize,
				totalPages
			} satisfies PaginatedResponse<ProjectContext>;
		},
		enabled: !!projectId,
		staleTime: 30 * 1000
	}));
}

export function useCreateProjectContext(projectId: string) {
	const queryClient = useQueryClient();
	return createMutation<ProjectContext, Error, CreateProjectContextPayload>(() => ({
		mutationFn: (payload) =>
			apiClient.post<ProjectContext>(
				`/api/v1/projects/${projectId}/contexts`,
				payload
			),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: contextKeys.all(projectId) });
		}
	}));
}

export function useUpdateProjectContext(projectId: string, contextId: string) {
	const queryClient = useQueryClient();
	return createMutation<ProjectContext, Error, UpdateProjectContextPayload>(() => ({
		mutationFn: (payload) =>
			apiClient.patch<ProjectContext>(
				`/api/v1/projects/${projectId}/contexts/${contextId}`,
				payload
			),
		onSuccess: (updated) => {
			queryClient.setQueryData(contextKeys.detail(projectId, contextId), updated);
			queryClient.invalidateQueries({ queryKey: contextKeys.lists(projectId) });
		}
	}));
}

export function useDeleteProjectContext(projectId: string, contextId: string) {
	const queryClient = useQueryClient();
	return createMutation<void, Error, void>(() => ({
		mutationFn: () =>
			apiClient.delete(`/api/v1/projects/${projectId}/contexts/${contextId}`),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: contextKeys.all(projectId) });
		}
	}));
}
