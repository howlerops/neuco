import { createQuery, createMutation, useQueryClient } from '@tanstack/svelte-query';
import { apiClient } from '$lib/api/client';

// ─── Backend response shapes ──────────────────────────────────────────────────
//
// GET  /api/v1/projects/{projectId}/nango/connections  → NangoConnection[]
// POST /api/v1/projects/{projectId}/nango/connections  → NangoConnection
// POST /api/v1/projects/{projectId}/nango/sync/{connectionId}  → { ok: true }
// DELETE /api/v1/projects/{projectId}/nango/connections/{connectionId}  → 204

export interface NangoConnection {
	id: string;
	projectId: string;
	/** Nango integration key, e.g. 'gong', 'intercom' */
	providerConfigKey: string;
	/** Unique per project — format: `{projectId}-{provider}` */
	connectionId: string;
	createdAt: string;
	lastSyncAt?: string | null;
}

export interface CreateNangoConnectionPayload {
	providerConfigKey: string;
	connectionId: string;
}

export interface SyncConnectionPayload {
	connectionId: string;
}

// ─── Query Keys ───────────────────────────────────────────────────────────────

export const nangoKeys = {
	all: (projectId: string) => ['projects', projectId, 'nango'] as const,
	connections: (projectId: string) => [...nangoKeys.all(projectId), 'connections'] as const
};

// ─── Hooks ────────────────────────────────────────────────────────────────────

export function useNangoConnections(projectId: string) {
	return createQuery<NangoConnection[]>(() => ({
		queryKey: nangoKeys.connections(projectId),
		queryFn: () =>
			apiClient.get<NangoConnection[]>(`/api/v1/projects/${projectId}/nango/connections`),
		enabled: !!projectId,
		staleTime: 60 * 1000
	}));
}

export function useCreateNangoConnection(projectId: string) {
	const queryClient = useQueryClient();

	return createMutation<NangoConnection, Error, CreateNangoConnectionPayload>(() => ({
		mutationFn: (payload: CreateNangoConnectionPayload) =>
			apiClient.post<NangoConnection>(
				`/api/v1/projects/${projectId}/nango/connections`,
				payload
			),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: nangoKeys.connections(projectId) });
		}
	}));
}

export function useDeleteNangoConnection(projectId: string) {
	const queryClient = useQueryClient();

	return createMutation<void, Error, string>(() => ({
		mutationFn: (connectionId: string) =>
			apiClient.delete<void>(
				`/api/v1/projects/${projectId}/nango/connections/${connectionId}`
			),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: nangoKeys.connections(projectId) });
		}
	}));
}

export function useCreateNangoSession() {
	return createMutation<{ token: string }, Error, void>(() => ({
		mutationFn: () => apiClient.post<{ token: string }>('/api/v1/auth/nango/connect-session')
	}));
}

export function useSyncConnection(projectId: string) {
	const queryClient = useQueryClient();

	return createMutation<void, Error, string>(() => ({
		mutationFn: (connectionId: string) =>
			apiClient.post<void>(
				`/api/v1/projects/${projectId}/nango/sync/${connectionId}`
			),
		onSuccess: () => {
			// Refresh connections list so lastSyncAt updates
			queryClient.invalidateQueries({ queryKey: nangoKeys.connections(projectId) });
		}
	}));
}
