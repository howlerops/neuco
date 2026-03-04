<script lang="ts">
	import { cn } from '$lib/utils';
	import { Avatar } from 'bits-ui';

	interface Props {
		class?: string;
		children?: import('svelte').Snippet;
	}

	let { class: className = '', children, ...restProps }: Props = $props();
</script>

<Avatar.Fallback
	class={cn(
		'flex h-full w-full items-center justify-center rounded-full bg-muted text-sm font-medium',
		className
	)}
	{...restProps}
>
	{@render children?.()}
</Avatar.Fallback>
