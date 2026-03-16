import { createQuery, createMutation, useQueryClient } from '@tanstack/svelte-query';
import { apiClient } from '$lib/api/client';
import type { OnboardingStatus, OnboardingStep } from '$lib/api/types-compat';

// ─── Query Keys ───────────────────────────────────────────────────────────────

export const onboardingKeys = {
	all: () => ['onboarding'] as const,
	status: () => [...onboardingKeys.all(), 'status'] as const
};

// ─── Hooks ────────────────────────────────────────────────────────────────────

export function useOnboardingStatus() {
	return createQuery<OnboardingStatus>(() => ({
		queryKey: onboardingKeys.status(),
		queryFn: () => apiClient.get<OnboardingStatus>('/api/v1/onboarding/status'),
		staleTime: 5 * 60 * 1000
	}));
}

export function useCompleteStep() {
	const queryClient = useQueryClient();
	return createMutation<OnboardingStatus, Error, { step: OnboardingStep }>(() => ({
		mutationFn: (payload) =>
			apiClient.post<OnboardingStatus>('/api/v1/onboarding/step', payload),
		onSuccess: (data) => {
			queryClient.setQueryData(onboardingKeys.status(), data);
		}
	}));
}

export function useSkipOnboarding() {
	const queryClient = useQueryClient();
	return createMutation<OnboardingStatus, Error, void>(() => ({
		mutationFn: () => apiClient.post<OnboardingStatus>('/api/v1/onboarding/skip', {}),
		onSuccess: (data) => {
			queryClient.setQueryData(onboardingKeys.status(), data);
		}
	}));
}
