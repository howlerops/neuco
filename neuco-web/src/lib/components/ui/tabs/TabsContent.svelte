<script lang="ts">
	import { cn } from '$lib/utils';
	import { Tabs } from 'bits-ui';

	interface Props {
		class?: string;
		value: string;
		children?: import('svelte').Snippet;
	}

	let { class: className = '', value, children, ...restProps }: Props = $props();
</script>

<Tabs.Content
	{value}
	class={cn(
		'mt-2 ring-offset-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2',
		className
	)}
	{...restProps}
>
	{@render children?.()}
</Tabs.Content>
