import { createQuery, createMutation, useQueryClient } from '@tanstack/svelte-query';
import { apiClient } from '$lib/api/client';
import type { Notification } from '$lib/api/types-compat';

// ─── Backend response shapes ──────────────────────────────────────────────────

interface NotificationsBackendResponse {
	notifications: Notification[];
	total: number;
}

interface UnreadCountResponse {
	count: number;
}

// ─── Query Keys ───────────────────────────────────────────────────────────────

export const notificationKeys = {
	all: (orgId: string) => ['orgs', orgId, 'notifications'] as const,
	list: (orgId: string) => [...notificationKeys.all(orgId), 'list'] as const,
	unreadCount: (orgId: string) => [...notificationKeys.all(orgId), 'unread-count'] as const
};

// ─── Hooks ────────────────────────────────────────────────────────────────────

export function useNotifications(getOrgId: () => string, limit = 20) {
	return createQuery<Notification[]>(() => {
		const orgId = getOrgId();
		return {
			queryKey: notificationKeys.list(orgId),
			queryFn: async () => {
				const raw = await apiClient.get<NotificationsBackendResponse>(
					`/api/v1/orgs/${orgId}/notifications?limit=${limit}`
				);
				return raw.notifications ?? [];
			},
			enabled: !!orgId,
			staleTime: 15 * 1000,
			refetchInterval: 60 * 1000
		};
	});
}

export function useUnreadCount(getOrgId: () => string) {
	return createQuery<number>(() => {
		const orgId = getOrgId();
		return {
			queryKey: notificationKeys.unreadCount(orgId),
			queryFn: async () => {
				const raw = await apiClient.get<UnreadCountResponse>(
					`/api/v1/orgs/${orgId}/notifications/unread-count`
				);
				return raw.count ?? 0;
			},
			enabled: !!orgId,
			staleTime: 15 * 1000,
			refetchInterval: 30 * 1000
		};
	});
}

export function useMarkNotificationRead(getOrgId: () => string) {
	const queryClient = useQueryClient();
	return createMutation<void, Error, string>(() => ({
		mutationFn: (notificationId: string) =>
			apiClient.patch<void>(`/api/v1/orgs/${getOrgId()}/notifications/${notificationId}/read`),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: notificationKeys.all(getOrgId()) });
		}
	}));
}

export function useMarkAllRead(getOrgId: () => string) {
	const queryClient = useQueryClient();
	return createMutation<void, Error, void>(() => ({
		mutationFn: () => apiClient.post<void>(`/api/v1/orgs/${getOrgId()}/notifications/read-all`),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: notificationKeys.all(getOrgId()) });
		}
	}));
}
