<script lang="ts">
	import { cn } from '$lib/utils';
	import { Tabs } from 'bits-ui';

	interface Props {
		class?: string;
		value: string;
		disabled?: boolean;
		children?: import('svelte').Snippet;
	}

	let {
		class: className = '',
		value,
		disabled = false,
		children,
		...restProps
	}: Props = $props();
</script>

<Tabs.Trigger
	{value}
	{disabled}
	class={cn(
		'inline-flex items-center justify-center whitespace-nowrap rounded-sm px-3 py-1.5 text-sm font-medium ring-offset-background transition-all focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:pointer-events-none disabled:opacity-50 data-[state=active]:bg-background data-[state=active]:text-foreground data-[state=active]:shadow-sm',
		className
	)}
	{...restProps}
>
	{@render children?.()}
</Tabs.Trigger>
