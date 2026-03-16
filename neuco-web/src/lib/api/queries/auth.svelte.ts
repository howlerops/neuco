import { createQuery, createMutation } from '@tanstack/svelte-query';
import { apiClient } from '$lib/api/client';
import type { User, Organization } from '$lib/api/types-compat';
import { authStore } from '$lib/stores/auth.svelte';
import { resetUser } from '$lib/analytics';
import { goto } from '$app/navigation';

// ─── Types ────────────────────────────────────────────────────────────────────

interface LoginResponse {
	accessToken: string;
	expiresIn: number;
}

export interface MeResponse {
	user: User;
	orgs: Organization[];
}

// ─── Query Keys ───────────────────────────────────────────────────────────────

export const authKeys = {
	me: () => ['auth', 'me'] as const
};

// ─── Hooks ────────────────────────────────────────────────────────────────────

export function useMe() {
	return createQuery<MeResponse>(() => ({
		queryKey: authKeys.me(),
		queryFn: () => apiClient.get<MeResponse>('/api/v1/auth/me'),
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
			resetUser();
			authStore.clearAuth();
			goto('/login');
		}
	}));
}
