<script lang="ts">
	import { browser } from '$app/environment';
	import { goto } from '$app/navigation';
	import { authStore } from '$lib/stores/auth.svelte';
	import { apiClient } from '$lib/api/client';

	interface MeResponse {
		user: { id: string; githubLogin: string; email: string; avatarUrl: string };
		orgs: { id: string; name: string; slug: string; plan: string }[];
	}

	$effect(() => {
		if (!browser) return;

		if (!authStore.isAuthenticated) {
			goto('/login');
			return;
		}

		// We have tokens but need to fetch user + orgs to know where to redirect
		apiClient.get<MeResponse>('/api/v1/auth/me')
			.then((data) => {
				if (data.orgs && data.orgs.length > 0) {
					const org = data.orgs[0];
					authStore.setOrg(org as any);
					goto(`/${org.slug}/dashboard`);
				} else {
					// No orgs — shouldn't happen since we auto-create on login
					goto('/login');
				}
			})
			.catch(() => {
				// Token might be invalid
				authStore.clearAuth();
				goto('/login');
			});
	});
</script>

<div class="flex min-h-screen items-center justify-center">
	<div class="h-8 w-8 animate-spin rounded-full border-4 border-muted border-t-primary"></div>
</div>
