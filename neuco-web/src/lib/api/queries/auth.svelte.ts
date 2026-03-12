import { createQuery, createMutation } from '@tanstack/svelte-query';
import { apiClient } from '$lib/api/client';
import type { User } from '$lib/api/types';
import { authStore } from '$lib/stores/auth.svelte';
import { goto } from '$app/navigation';

// ─── Types ────────────────────────────────────────────────────────────────────

interface LoginResponse {
	accessToken: string;
	expiresIn: number;
}

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
	return createMutation<LoginResponse, Error, { code: string }>(() => ({
		mutationFn: ({ code }: { code: string }) =>
			apiClient.post<LoginResponse>('/api/v1/auth/github/callback', { code }),
		onSuccess: (data: LoginResponse) => {
			authStore.setTokens(data.accessToken, data.expiresIn);
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
