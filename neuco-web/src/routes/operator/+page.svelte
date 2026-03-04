<script lang="ts">
	import { goto } from '$app/navigation';
	import { onMount } from 'svelte';

	onMount(() => {
		goto('/operator/orgs');
	});
</script>

<p class="text-muted-foreground">Redirecting to organizations...</p>
