import { createQuery } from '@tanstack/svelte-query';
import { apiClient } from '$lib/api/client';
import type { LLMUsageAgg } from '$lib/api/types';

// ─── Query Keys ───────────────────────────────────────────────────────────────

export const llmUsageKeys = {
	all: () => ['llm-usage'] as const,
	org: (orgId: string) => [...llmUsageKeys.all(), 'org', orgId] as const,
	project: (projectId: string) => [...llmUsageKeys.all(), 'project', projectId] as const,
	pipeline: (projectId: string, runId: string) =>
		[...llmUsageKeys.all(), 'pipeline', projectId, runId] as const
};

// ─── Hooks ────────────────────────────────────────────────────────────────────

export function useOrgLLMUsage(orgId: string) {
	return createQuery<LLMUsageAgg>(() => ({
		queryKey: llmUsageKeys.org(orgId),
		queryFn: () => apiClient.get<LLMUsageAgg>(`/api/v1/orgs/${orgId}/llm-usage`),
		enabled: !!orgId,
		staleTime: 30 * 1000
	}));
}

export function useProjectLLMUsage(projectId: string) {
	return createQuery<LLMUsageAgg>(() => ({
		queryKey: llmUsageKeys.project(projectId),
		queryFn: () => apiClient.get<LLMUsageAgg>(`/api/v1/projects/${projectId}/llm-usage`),
		enabled: !!projectId,
		staleTime: 30 * 1000
	}));
}

export function usePipelineLLMUsage(projectId: string, runId: string) {
	return createQuery<LLMUsageAgg>(() => ({
		queryKey: llmUsageKeys.pipeline(projectId, runId),
		queryFn: () =>
			apiClient.get<LLMUsageAgg>(
				`/api/v1/projects/${projectId}/pipelines/${runId}/llm-usage`
			),
		enabled: !!projectId && !!runId,
		staleTime: 30 * 1000
	}));
}
