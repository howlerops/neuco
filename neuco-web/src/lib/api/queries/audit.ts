import { createQuery } from '@tanstack/svelte-query';
import { apiClient } from '$lib/api/client';
import type { AuditEntry, PageParams, PaginatedResponse } from '$lib/api/types-compat';

// ─── Filter Params ────────────────────────────────────────────────────────────

export interface AuditFilterParams extends PageParams {
	action?: string;
	entityType?: string;
}

// ─── Query Keys ───────────────────────────────────────────────────────────────

export const auditKeys = {
	all: (orgId: string) => ['organizations', orgId, 'audit'] as const,
	lists: (orgId: string, params?: AuditFilterParams) =>
		[...auditKeys.all(orgId), 'list', params] as const
};

// ─── Hooks ────────────────────────────────────────────────────────────────────

export function useAuditLog(orgId: string, params?: AuditFilterParams) {
	return createQuery<PaginatedResponse<AuditEntry>>(() => ({
		queryKey: auditKeys.lists(orgId, params),
		queryFn: () => {
			const searchParams = new URLSearchParams();
			if (params?.page) searchParams.set('page', String(params.page));
			if (params?.pageSize) searchParams.set('pageSize', String(params.pageSize));
			if (params?.action) searchParams.set('action', params.action);
			if (params?.entityType) searchParams.set('entityType', params.entityType);
			const qs = searchParams.toString();
			return apiClient.get<PaginatedResponse<AuditEntry>>(
				`/api/v1/orgs/${orgId}/audit-log${qs ? `?${qs}` : ''}`
			);
		},
		enabled: !!orgId,
		staleTime: 30 * 1000
	}));
}
