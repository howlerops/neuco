<script lang="ts">
	import { browser } from '$app/environment';
	import { goto } from '$app/navigation';
	import { page } from '$app/stores';
	import { onMount } from 'svelte';
	import { authStore } from '$lib/stores/auth.svelte';
	import { apiClient } from '$lib/api/client';

	// After the API client's snake→camel transformer, keys are camelCase
	interface CallbackResponse {
		accessToken: string;
		refreshToken: string;
		expiresIn: number;
	}

	let status = $state<'loading' | 'error'>('loading');
	let errorMessage = $state('');

	onMount(async () => {
		if (!browser) return;

		const code = $page.url.searchParams.get('code');
		const errorParam = $page.url.searchParams.get('error');

		if (errorParam) {
			status = 'error';
			errorMessage = $page.url.searchParams.get('error_description') ?? 'GitHub authorization failed.';
			return;
		}

		if (!code) {
			status = 'error';
			errorMessage = 'No authorization code received from GitHub.';
			return;
		}

		try {
			const data = await apiClient.post<CallbackResponse>('/api/v1/auth/github/callback', {
				code
			});

			authStore.setTokens(data.accessToken, data.refreshToken);

			// Redirect to root — the app layout will fetch /me and resolve the user + org
			goto('/');
		} catch (err) {
			status = 'error';
			errorMessage =
				err instanceof Error
					? err.message
					: 'Authentication failed. Please try again.';
		}
	});
</script>

<svelte:head>
	<title>Signing in — Neuco</title>
</svelte:head>

<div class="flex min-h-screen flex-col items-center justify-center gap-4 bg-background">
	{#if status === 'loading'}
		<div class="flex flex-col items-center gap-3">
			<div class="h-10 w-10 animate-spin rounded-full border-4 border-muted border-t-primary"></div>
			<p class="text-sm text-muted-foreground">Completing sign in...</p>
		</div>
	{:else}
		<div class="flex max-w-sm flex-col items-center gap-4 text-center">
			<div class="flex h-12 w-12 items-center justify-center rounded-full bg-destructive/10 text-destructive">
				<svg class="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
					<path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" />
				</svg>
			</div>
			<div>
				<h2 class="text-lg font-semibold">Authentication failed</h2>
				<p class="mt-1 text-sm text-muted-foreground">{errorMessage}</p>
			</div>
			<a
				href="/login"
				class="inline-flex h-10 items-center justify-center rounded-md bg-primary px-4 text-sm font-medium text-primary-foreground hover:bg-primary/90"
			>
				Back to sign in
			</a>
		</div>
	{/if}
</div>
