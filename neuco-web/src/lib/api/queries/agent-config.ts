import { createQuery, createMutation, useQueryClient } from '@tanstack/svelte-query';
import { apiClient } from '$lib/api/client';
import type {
	AgentConfig,
	UpsertAgentConfigPayload,
	ValidateAgentConfigPayload,
	ValidateAgentConfigResponse,
	AgentProviderInfo
} from '$lib/api/types';

export const agentConfigKeys = {
	all: (projectId: string) => ['projects', projectId, 'agent-config'] as const,
	detail: (projectId: string) => [...agentConfigKeys.all(projectId), 'detail'] as const,
	providers: () => ['agent-providers'] as const
};

export function useAgentConfig(projectId: string) {
	return createQuery<AgentConfig>(() => ({
		queryKey: agentConfigKeys.detail(projectId),
		queryFn: () => apiClient.get<AgentConfig>('/api/v1/projects/' + projectId + '/agent-config'),
		enabled: !!projectId,
		staleTime: 60_000,
		retry: false // 404 is expected when no config exists
	}));
}

export function useUpsertAgentConfig(projectId: string) {
	const qc = useQueryClient();
	return createMutation<AgentConfig, Error, UpsertAgentConfigPayload>(() => ({
		mutationFn: (payload) =>
			apiClient.put<AgentConfig>('/api/v1/projects/' + projectId + '/agent-config', payload),
		onSuccess: () => {
			qc.invalidateQueries({ queryKey: agentConfigKeys.all(projectId) });
		}
	}));
}

export function useDeleteAgentConfig(projectId: string) {
	const qc = useQueryClient();
	return createMutation<void, Error, { provider: string }>(() => ({
		mutationFn: ({ provider }) =>
			apiClient.delete<void>('/api/v1/projects/' + projectId + '/agent-config?provider=' + provider),
		onSuccess: () => {
			qc.invalidateQueries({ queryKey: agentConfigKeys.all(projectId) });
		}
	}));
}

export function useValidateAgentConfig(projectId: string) {
	return createMutation<ValidateAgentConfigResponse, Error, ValidateAgentConfigPayload>(() => ({
		mutationFn: (payload) =>
			apiClient.post<ValidateAgentConfigResponse>(
				'/api/v1/projects/' + projectId + '/agent-config/validate',
				payload
			)
	}));
}

export function useAgentProviders() {
	return createQuery<AgentProviderInfo[]>(() => ({
		queryKey: agentConfigKeys.providers(),
		queryFn: () => apiClient.get<AgentProviderInfo[]>('/api/v1/agent-providers'),
		staleTime: 5 * 60_000
	}));
}
