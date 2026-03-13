import { createQuery, createMutation, useQueryClient } from '@tanstack/svelte-query';
import { apiClient, transformKeys } from '$lib/api/client';
import type { Signal, PaginatedResponse, SignalFilterParams } from '$lib/api/types';

// ─── Backend response shapes ──────────────────────────────────────────────────
// GET /api/v1/projects/{projectId}/signals returns:
//   { signals: Signal[], total: number }
// The query hook normalises this into PaginatedResponse<Signal> so pages can
// use the standard .data / .total / .totalPages shape.

interface SignalsBackendResponse {
	signals: Signal[];
	total: number;
}

// ─── Query Keys ───────────────────────────────────────────────────────────────

export const signalKeys = {
	all: (projectId: string) => ['projects', projectId, 'signals'] as const,
	lists: (projectId: string, params?: SignalFilterParams) =>
		[...signalKeys.all(projectId), 'list', params] as const,
	detail: (projectId: string, signalId: string) =>
		[...signalKeys.all(projectId), 'detail', signalId] as const
};

// ─── Hooks ────────────────────────────────────────────────────────────────────

export function useSignals(projectId: string, params?: SignalFilterParams) {
	return createQuery<PaginatedResponse<Signal>>(() => ({
		queryKey: signalKeys.lists(projectId, params),
		queryFn: async () => {
			const searchParams = new URLSearchParams();
			// Backend uses limit/offset, not page/pageSize
			const page = params?.page ?? 1;
			const pageSize = params?.pageSize ?? 20;
			const limit = pageSize;
			const offset = (page - 1) * pageSize;
			searchParams.set('limit', String(limit));
			searchParams.set('offset', String(offset));
			if (params?.source) searchParams.set('source', params.source);
			if (params?.type) searchParams.set('type', params.type);
			if (params?.search) searchParams.set('search', params.search);
			if (params?.excludeDuplicates) searchParams.set('exclude_duplicates', 'true');
			const qs = searchParams.toString();
			const raw = await apiClient.get<SignalsBackendResponse>(
				`/api/v1/projects/${projectId}/signals${qs ? `?${qs}` : ''}`
			);
			// Normalise to PaginatedResponse shape
			const items = raw.signals ?? [];
			const total = raw.total ?? 0;
			const totalPages = Math.max(1, Math.ceil(total / pageSize));
			return {
				data: items,
				total,
				page,
				pageSize,
				totalPages
			} satisfies PaginatedResponse<Signal>;
		},
		enabled: !!projectId,
		staleTime: 30 * 1000
	}));
}

export function useUploadSignals(projectId: string) {
	const queryClient = useQueryClient();
	return createMutation<Signal[], Error, FormData>(() => ({
		mutationFn: async (formData: FormData) => {
			// Use raw fetch (not apiClient) because multipart/form-data must NOT have
			// a manually-set Content-Type header — the browser sets it with the boundary.
			// We manually apply the snake→camel transformer to the response to match
			// what apiClient would do automatically.
			const res = await fetch(
				`${import.meta.env.VITE_API_BASE_URL ?? 'http://localhost:8080'}/api/v1/projects/${projectId}/signals/upload`,
				{
					method: 'POST',
					headers: {
						Authorization: `Bearer ${localStorage.getItem('access_token') ?? ''}`
					},
					body: formData
				}
			);
			if (!res.ok) throw new Error(await res.text());
			const raw = await res.json();
			return transformKeys(raw) as Signal[];
		},
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: signalKeys.all(projectId) });
		}
	}));
}

export function useDeleteSignal(projectId: string) {
	const queryClient = useQueryClient();
	return createMutation<void, Error, string>(() => ({
		mutationFn: (signalId: string) =>
			apiClient.delete<void>(`/api/v1/projects/${projectId}/signals/${signalId}`),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: signalKeys.all(projectId) });
		}
	}));
}
