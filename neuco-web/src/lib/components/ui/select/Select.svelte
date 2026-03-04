<script lang="ts">
	import { Select } from 'bits-ui';

	interface Props {
		value?: string;
		onValueChange?: (value: string) => void;
		disabled?: boolean;
		required?: boolean;
		name?: string;
		children?: import('svelte').Snippet;
	}

	let {
		value = $bindable(''),
		onValueChange,
		disabled = false,
		required = false,
		name,
		children
	}: Props = $props();
</script>

<Select.Root
	type="single"
	bind:value
	{onValueChange}
	{disabled}
	{required}
	{name}
>
	{@render children?.()}
</Select.Root>
