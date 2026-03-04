<script lang="ts">
	import { Toaster } from 'svelte-sonner';
	import { ModeWatcher } from 'mode-watcher';

	interface Props {
		position?:
			| 'top-left'
			| 'top-right'
			| 'bottom-left'
			| 'bottom-right'
			| 'top-center'
			| 'bottom-center';
		richColors?: boolean;
		closeButton?: boolean;
		duration?: number;
	}

	let {
		position = 'bottom-right',
		richColors = true,
		closeButton = false,
		duration = 4000
	}: Props = $props();
</script>

<Toaster
	{position}
	{richColors}
	{closeButton}
	{duration}
	theme="system"
/>
