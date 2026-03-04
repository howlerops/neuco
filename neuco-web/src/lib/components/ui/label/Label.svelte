<script lang="ts">
	import { cn } from '$lib/utils';
	import { Label as LabelPrimitive } from 'bits-ui';

	interface Props {
		class?: string;
		for?: string;
		children?: import('svelte').Snippet;
	}

	let { class: className = '', for: htmlFor, children, ...restProps }: Props = $props();
</script>

<LabelPrimitive.Root
	for={htmlFor}
	class={cn(
		'text-sm font-medium leading-none peer-disabled:cursor-not-allowed peer-disabled:opacity-70',
		className
	)}
	{...restProps}
>
	{@render children?.()}
</LabelPrimitive.Root>
