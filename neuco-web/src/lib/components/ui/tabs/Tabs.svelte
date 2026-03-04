<script lang="ts">
	import { Tabs } from 'bits-ui';

	interface Props {
		value?: string;
		onValueChange?: (value: string) => void;
		class?: string;
		orientation?: 'horizontal' | 'vertical';
		children?: import('svelte').Snippet;
	}

	let {
		value = $bindable(''),
		onValueChange,
		class: className = '',
		orientation = 'horizontal',
		children
	}: Props = $props();
</script>

<Tabs.Root bind:value {onValueChange} {orientation} class={className}>
	{@render children?.()}
</Tabs.Root>
