import { createQuery, createMutation, useQueryClient } from '@tanstack/svelte-query';
import { apiClient } from '$lib/api/client';
import type { Project, CreateProjectPayload, UpdateProjectPayload } from '$lib/api/types';

// ─── Query Keys ───────────────────────────────────────────────────────────────

export const projectKeys = {
	all: (orgId: string) => ['organizations', orgId, 'projects'] as const,
	lists: (orgId: string) => [...projectKeys.all(orgId), 'list'] as const,
	detail: (orgId: string, projectId: string) =>
		[...projectKeys.all(orgId), 'detail', projectId] as const,
	bySlug: (orgSlug: string, projectSlug: string) =>
		['organizations', orgSlug, 'projects', 'slug', projectSlug] as const
};

// ─── Hooks ────────────────────────────────────────────────────────────────────

export function useProjects(orgId: string) {
	return createQuery<Project[]>(() => ({
		queryKey: projectKeys.lists(orgId),
		queryFn: () => apiClient.get<Project[]>(`/api/v1/orgs/${orgId}/projects`),
		enabled: !!orgId,
		staleTime: 60 * 1000
	}));
}

export function useProject(orgId: string, projectId: string) {
	return createQuery<Project>(() => ({
		queryKey: projectKeys.detail(orgId, projectId),
		queryFn: () =>
			apiClient.get<Project>(`/api/v1/projects/${projectId}`),
		enabled: !!orgId && !!projectId,
		staleTime: 60 * 1000
	}));
}

export function useCreateProject(orgId: string) {
	const queryClient = useQueryClient();
	return createMutation<Project, Error, CreateProjectPayload>(() => ({
		mutationFn: (payload: CreateProjectPayload) =>
			apiClient.post<Project>(`/api/v1/orgs/${orgId}/projects`, payload),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: projectKeys.lists(orgId) });
		}
	}));
}

export function useUpdateProject(orgId: string, projectId: string) {
	const queryClient = useQueryClient();
	return createMutation<Project, Error, UpdateProjectPayload>(() => ({
		mutationFn: (payload: UpdateProjectPayload) =>
			apiClient.patch<Project>(
				`/api/v1/projects/${projectId}`,
				payload
			),
		onSuccess: (updated: Project) => {
			queryClient.setQueryData(projectKeys.detail(orgId, projectId), updated);
			queryClient.invalidateQueries({ queryKey: projectKeys.lists(orgId) });
		}
	}));
}
