<script lang="ts">
	import { cn } from '$lib/utils';
	import { Select } from 'bits-ui';
	import { ChevronDown } from 'lucide-svelte';

	interface Props {
		class?: string;
		placeholder?: string;
		disabled?: boolean;
		children?: import('svelte').Snippet;
	}

	let {
		class: className = '',
		placeholder = 'Select...',
		disabled = false,
		children,
		...restProps
	}: Props = $props();
</script>

<Select.Trigger
	{disabled}
	class={cn(
		'flex h-10 w-full items-center justify-between rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50 [&>span]:line-clamp-1',
		className
	)}
	{...restProps}
>
	{@render children?.()}
	<ChevronDown class="h-4 w-4 opacity-50 shrink-0" />
</Select.Trigger>
