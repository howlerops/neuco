<script lang="ts">
	import { cn } from '$lib/utils';
	import { Dialog } from 'bits-ui';

	interface Props {
		class?: string;
		children?: import('svelte').Snippet;
	}

	let { class: className = '', children, ...restProps }: Props = $props();
</script>

<Dialog.Description class={cn('text-sm text-muted-foreground', className)} {...restProps}>
	{@render children?.()}
</Dialog.Description>
