<script lang="ts">
	import { cn } from '$lib/utils';
	import { Avatar } from 'bits-ui';

	interface Props {
		class?: string;
		children?: import('svelte').Snippet;
	}

	let { class: className = '', children, ...restProps }: Props = $props();
</script>

<Avatar.Root
	class={cn('relative flex h-10 w-10 shrink-0 overflow-hidden rounded-full', className)}
	{...restProps}
>
	{@render children?.()}
</Avatar.Root>
