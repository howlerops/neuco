<script lang="ts">
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import { onMount } from 'svelte';
	import { apiClient } from '$lib/api/client';
	import { Loader2, CheckCircle2, XCircle } from 'lucide-svelte';

	let status = $state<'loading' | 'success' | 'error'>('loading');
	let errorMessage = $state('');

	onMount(async () => {
		const code = $page.url.searchParams.get('code');
		const state = $page.url.searchParams.get('state');

		if (!code || !state) {
			status = 'error';
			errorMessage = 'Missing authorization code or state parameter.';
			return;
		}

		// Extract projectId from state (format: "projectId:randomHex")
		const projectId = state.split(':')[0];
		if (!projectId) {
			status = 'error';
			errorMessage = 'Invalid state parameter.';
			return;
		}

		try {
			await apiClient.post(`/api/v1/projects/${projectId}/intercom/callback`, {
				code,
				state
			});
			status = 'success';

			// Redirect back to integrations page after a brief delay.
			setTimeout(() => {
				// We need the org slug to navigate. For now, go to the root and let
				// the app resolve.
				window.close();
			}, 2000);
		} catch (err) {
			status = 'error';
			errorMessage = err instanceof Error ? err.message : 'Failed to connect Intercom.';
		}
	});
</script>

<svelte:head>
	<title>Connecting Intercom — Neuco</title>
</svelte:head>

<div class="flex min-h-screen items-center justify-center">
	<div class="text-center space-y-4">
		{#if status === 'loading'}
			<Loader2 class="h-8 w-8 animate-spin mx-auto text-muted-foreground" />
			<p class="text-muted-foreground">Connecting your Intercom workspace...</p>
		{:else if status === 'success'}
			<CheckCircle2 class="h-8 w-8 mx-auto text-green-500" />
			<p class="font-medium">Intercom connected successfully!</p>
			<p class="text-sm text-muted-foreground">This window will close automatically.</p>
		{:else}
			<XCircle class="h-8 w-8 mx-auto text-red-500" />
			<p class="font-medium">Failed to connect Intercom</p>
			<p class="text-sm text-muted-foreground">{errorMessage}</p>
		{/if}
	</div>
</div>
