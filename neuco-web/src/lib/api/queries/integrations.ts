import { createQuery, createMutation, useQueryClient } from '@tanstack/svelte-query';
import { apiClient } from '$lib/api/client';

// ─── Backend response shapes ──────────────────────────────────────────────────
//
// The integrations backend uses `kind` (not `provider`) as the field name.
// `webhookSecret` is only present on the create (POST) response — the GET list
// strips it for security.
//
// GET  /api/v1/projects/{projectId}/integrations  → IntegrationRecord[]
// POST /api/v1/projects/{projectId}/integrations  → IntegrationRecord (with webhookSecret)
// DELETE /api/v1/projects/{projectId}/integrations/{integrationId}  → 204 No Content

export interface IntegrationRecord {
	id: string;
	projectId: string;
	provider: string;
	webhookSecret?: string;
	config?: Record<string, unknown>;
	lastSyncAt?: string;
	isActive: boolean;
	createdAt: string;
}

// ─── Extended Types ────────────────────────────────────────────────────────────

export type ExtendedIntegrationProvider =
	| 'gong'
	| 'intercom'
	| 'linear'
	| 'jira'
	| 'hubspot'
	| 'notion'
	| 'slack'
	| 'webhook'
	| 'github';

export interface CreateIntegrationPayload {
	provider: ExtendedIntegrationProvider;
	config?: Record<string, unknown>;
}

/**
 * Returned by `useCreateIntegration.mutateAsync`.
 * Contains the full integration record plus the one-time webhook secret.
 */
export interface CreateIntegrationResult {
	integration: IntegrationRecord;
	/** One-time secret, visible only on creation. Used to build the webhook URL. */
	webhookSecret: string;
}

// ─── Query Keys ───────────────────────────────────────────────────────────────

export const integrationKeys = {
	all: (projectId: string) => ['projects', projectId, 'integrations'] as const,
	lists: (projectId: string) => [...integrationKeys.all(projectId), 'list'] as const,
	detail: (projectId: string, integrationId: string) =>
		[...integrationKeys.all(projectId), 'detail', integrationId] as const
};

// ─── Hooks ────────────────────────────────────────────────────────────────────

export function useIntegrations(projectId: string) {
	return createQuery<IntegrationRecord[]>(() => ({
		queryKey: integrationKeys.lists(projectId),
		queryFn: () =>
			apiClient.get<IntegrationRecord[]>(`/api/v1/projects/${projectId}/integrations`),
		enabled: !!projectId,
		staleTime: 60 * 1000
	}));
}

export function useCreateIntegration(projectId: string) {
	const queryClient = useQueryClient();

	return createMutation<CreateIntegrationResult, Error, CreateIntegrationPayload>(() => ({
		mutationFn: async (payload: CreateIntegrationPayload) => {
			const record = await apiClient.post<IntegrationRecord>(
				`/api/v1/projects/${projectId}/integrations`,
				{ provider: payload.provider, config: payload.config ?? {} }
			);

			// `webhookSecret` is only present on this create response.
			// Extract it here so the caller receives it and can construct the URL.
			const webhookSecret = record.webhookSecret ?? '';

			return {
				integration: record,
				webhookSecret
			};
		},
		onSuccess: () => {
			// Invalidate the list so the new integration appears immediately.
			queryClient.invalidateQueries({ queryKey: integrationKeys.lists(projectId) });
		}
	}));
}

export function useDeleteIntegration(projectId: string) {
	const queryClient = useQueryClient();

	return createMutation<void, Error, string>(() => ({
		mutationFn: (integrationId: string) =>
			apiClient.delete<void>(
				`/api/v1/projects/${projectId}/integrations/${integrationId}`
			),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: integrationKeys.lists(projectId) });
		}
	}));
}
