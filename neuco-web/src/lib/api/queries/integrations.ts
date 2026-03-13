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

// ─── Native Intercom ──────────────────────────────────────────────────────────

export interface IntercomAuthorizeResponse {
	authorize_url: string;
	state: string;
}

export function useIntercomAuthorize(projectId: string) {
	return createQuery<IntercomAuthorizeResponse>(() => ({
		queryKey: [...integrationKeys.all(projectId), 'intercom-authorize'] as const,
		queryFn: () =>
			apiClient.get<IntercomAuthorizeResponse>(
				`/api/v1/projects/${projectId}/intercom/authorize`
			),
		enabled: false // Only fetch on demand
	}));
}

export function useIntercomCallback(projectId: string) {
	const queryClient = useQueryClient();

	return createMutation<IntegrationRecord, Error, { code: string; state: string }>(() => ({
		mutationFn: (payload) =>
			apiClient.post<IntegrationRecord>(
				`/api/v1/projects/${projectId}/intercom/callback`,
				payload
			),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: integrationKeys.lists(projectId) });
		}
	}));
}

export function useIntercomDisconnect(projectId: string) {
	const queryClient = useQueryClient();

	return createMutation<void, Error, string>(() => ({
		mutationFn: (integrationId: string) =>
			apiClient.delete<void>(
				`/api/v1/projects/${projectId}/intercom/${integrationId}`
			),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: integrationKeys.lists(projectId) });
		}
	}));
}

export function useIntercomSync(projectId: string) {
	return createMutation<{ run_id: string; status: string }, Error, string>(() => ({
		mutationFn: (integrationId: string) =>
			apiClient.post<{ run_id: string; status: string }>(
				`/api/v1/projects/${projectId}/intercom/${integrationId}/sync`,
				{}
			)
	}));
}

// ─── Native Slack ────────────────────────────────────────────────────────────

export interface SlackAuthorizeResponse {
	authorize_url: string;
	state: string;
}

export function useSlackDisconnect(projectId: string) {
	const queryClient = useQueryClient();

	return createMutation<void, Error, string>(() => ({
		mutationFn: (integrationId: string) =>
			apiClient.delete<void>(
				`/api/v1/projects/${projectId}/slack/${integrationId}`
			),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: integrationKeys.lists(projectId) });
		}
	}));
}

export function useSlackSync(projectId: string) {
	return createMutation<{ run_id: string; status: string }, Error, string>(() => ({
		mutationFn: (integrationId: string) =>
			apiClient.post<{ run_id: string; status: string }>(
				`/api/v1/projects/${projectId}/slack/${integrationId}/sync`,
				{}
			)
	}));
}

// ─── Native Jira ──────────────────────────────────────────────────────────────

export interface JiraAuthorizeResponse {
	authorize_url: string;
	state: string;
}

export function useJiraDisconnect(projectId: string) {
	const queryClient = useQueryClient();

	return createMutation<void, Error, string>(() => ({
		mutationFn: (integrationId: string) =>
			apiClient.delete<void>(
				`/api/v1/projects/${projectId}/jira/${integrationId}`
			),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: integrationKeys.lists(projectId) });
		}
	}));
}

export function useJiraSync(projectId: string) {
	return createMutation<{ run_id: string; status: string }, Error, string>(() => ({
		mutationFn: (integrationId: string) =>
			apiClient.post<{ run_id: string; status: string }>(
				`/api/v1/projects/${projectId}/jira/${integrationId}/sync`,
				{}
			)
	}));
}

// ─── Native Linear ────────────────────────────────────────────────────────────

export interface LinearAuthorizeResponse {
	authorize_url: string;
	state: string;
}

export function useLinearDisconnect(projectId: string) {
	const queryClient = useQueryClient();

	return createMutation<void, Error, string>(() => ({
		mutationFn: (integrationId: string) =>
			apiClient.delete<void>(
				`/api/v1/projects/${projectId}/linear/${integrationId}`
			),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: integrationKeys.lists(projectId) });
		}
	}));
}

export function useLinearSync(projectId: string) {
	return createMutation<{ run_id: string; status: string }, Error, string>(() => ({
		mutationFn: (integrationId: string) =>
			apiClient.post<{ run_id: string; status: string }>(
				`/api/v1/projects/${projectId}/linear/${integrationId}/sync`,
				{}
			)
	}));
}

// ─── Generic Delete ──────────────────────────────────────────────────────────

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
