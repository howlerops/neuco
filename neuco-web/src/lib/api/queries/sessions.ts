import { createQuery } from '@tanstack/svelte-query';
import { apiClient } from '$lib/api/client';
import type { SandboxSession, SandboxSessionPage } from '$lib/api/types';

export const sessionKeys = {
	all: (projectId: string) => ['projects', projectId, 'sessions'] as const,
	lists: (projectId: string) => [...sessionKeys.all(projectId), 'list'] as const,
	detail: (projectId: string, sessionId: string) =>
		[...sessionKeys.all(projectId), sessionId] as const
};

export function useSessions(projectId: string) {
	return createQuery<SandboxSessionPage>(() => ({
		queryKey: sessionKeys.lists(projectId),
		queryFn: () => apiClient.get<SandboxSessionPage>('/api/v1/projects/' + projectId + '/sessions'),
		enabled: !!projectId,
		staleTime: 10_000
	}));
}

export function useSession(projectId: string, sessionId: string) {
	return createQuery<SandboxSession>(() => ({
		queryKey: sessionKeys.detail(projectId, sessionId),
		queryFn: () =>
			apiClient.get<SandboxSession>('/api/v1/projects/' + projectId + '/sessions/' + sessionId),
		enabled: !!projectId && !!sessionId,
		staleTime: 5_000
	}));
}
