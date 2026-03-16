import { createQuery, createMutation, useQueryClient } from '@tanstack/svelte-query';
import { apiClient } from '$lib/api/client';
import type { CopilotNote } from '$lib/api/types-compat';

// ─── Query Keys ───────────────────────────────────────────────────────────────

export const copilotKeys = {
	all: (projectId: string) => ['projects', projectId, 'copilot-notes'] as const,
	active: (projectId: string) => [...copilotKeys.all(projectId), 'active'] as const
};

// ─── Hooks ────────────────────────────────────────────────────────────────────

export function useCopilotNotes(projectId: string) {
	return createQuery<CopilotNote[]>(() => ({
		queryKey: copilotKeys.active(projectId),
		queryFn: () =>
			apiClient.get<CopilotNote[]>(`/api/v1/projects/${projectId}/copilot/notes`),
		enabled: !!projectId,
		staleTime: 30 * 1000,
		refetchInterval: 60 * 1000
	}));
}

export function useDismissNote(projectId: string) {
	const queryClient = useQueryClient();
	return createMutation<void, Error, string>(() => ({
		mutationFn: (noteId: string) =>
			apiClient.patch<void>(
				`/api/v1/projects/${projectId}/copilot/notes/${noteId}/dismiss`
			),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: copilotKeys.active(projectId) });
		}
	}));
}
