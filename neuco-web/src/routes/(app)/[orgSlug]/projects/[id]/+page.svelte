<script lang="ts">
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import { browser } from '$app/environment';

	const orgSlug = $derived($page.params.orgSlug ?? '');
	const projectId = $derived($page.params.id ?? '');

	$effect(() => {
		if (browser && orgSlug && projectId) {
			goto(`/${orgSlug}/projects/${projectId}/signals`, { replaceState: true });
		}
	});
</script>

<div class="flex items-center justify-center p-12">
	<div class="h-6 w-6 animate-spin rounded-full border-2 border-muted border-t-primary"></div>
</div>
