import { createQuery, createMutation } from '@tanstack/svelte-query';
import { apiClient } from '$lib/api/client';
import type { AuthResponse, User } from '$lib/api/types';
import { authStore } from '$lib/stores/auth.svelte';
import { goto } from '$app/navigation';

// ─── Query Keys ───────────────────────────────────────────────────────────────

export const authKeys = {
	me: () => ['auth', 'me'] as const
};

// ─── Hooks ────────────────────────────────────────────────────────────────────

export function useMe() {
	return createQuery<User>(() => ({
		queryKey: authKeys.me(),
		queryFn: () => apiClient.get<User>('/api/v1/auth/me'),
		enabled: authStore.isAuthenticated,
		retry: false,
		staleTime: 5 * 60 * 1000
	}));
}

export function useGitHubLogin() {
	return createMutation<AuthResponse, Error, { code: string }>(() => ({
		mutationFn: ({ code }: { code: string }) =>
			apiClient.post<AuthResponse>('/api/v1/auth/github/callback', { code }),
		onSuccess: (data: AuthResponse) => {
			authStore.setTokens(data.accessToken, data.refreshToken);
			authStore.setUser(data.user);
			goto('/');
		}
	}));
}

export function useLogout() {
	return createMutation<void, Error, void>(() => ({
		mutationFn: () => apiClient.post<void>('/api/v1/auth/logout'),
		onSettled: () => {
			authStore.clearAuth();
			goto('/login');
		}
	}));
}
