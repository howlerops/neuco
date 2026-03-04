<script lang="ts">
	import { cn } from '$lib/utils';
	import { DropdownMenu } from 'bits-ui';

	interface Props {
		class?: string;
		inset?: boolean;
		disabled?: boolean;
		onSelect?: () => void;
		children?: import('svelte').Snippet;
	}

	let {
		class: className = '',
		inset = false,
		disabled = false,
		onSelect,
		children,
		...restProps
	}: Props = $props();
</script>

<DropdownMenu.Item
	{disabled}
	{onSelect}
	class={cn(
		'relative flex cursor-default select-none items-center rounded-sm px-2 py-1.5 text-sm outline-none transition-colors focus:bg-accent focus:text-accent-foreground data-[disabled]:pointer-events-none data-[disabled]:opacity-50',
		inset && 'pl-8',
		className
	)}
	{...restProps}
>
	{@render children?.()}
</DropdownMenu.Item>
