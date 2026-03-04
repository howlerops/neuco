<script lang="ts">
	import { cn } from '$lib/utils';
	import { Select } from 'bits-ui';
	import { Check } from 'lucide-svelte';

	interface Props {
		class?: string;
		value: string;
		label?: string;
		disabled?: boolean;
		children?: import('svelte').Snippet;
	}

	let {
		class: className = '',
		value,
		label,
		disabled = false,
		children,
		...restProps
	}: Props = $props();
</script>

<Select.Item
	{value}
	{label}
	{disabled}
	class={cn(
		'relative flex w-full cursor-default select-none items-center rounded-sm py-1.5 pl-8 pr-2 text-sm outline-none focus:bg-accent focus:text-accent-foreground data-[disabled]:pointer-events-none data-[disabled]:opacity-50',
		className
	)}
	{...restProps}
>
	{#snippet children({ selected })}
		<span class="absolute left-2 flex h-3.5 w-3.5 items-center justify-center">
			{#if selected}
				<Check class="h-4 w-4" />
			{/if}
		</span>
		{label ?? value}
	{/snippet}
</Select.Item>
