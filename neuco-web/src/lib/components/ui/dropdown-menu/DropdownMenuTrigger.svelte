<script lang="ts">
	import { DropdownMenu } from 'bits-ui';

	interface Props {
		class?: string;
		children?: import('svelte').Snippet;
	}

	let { class: className = '', children, ...restProps }: Props = $props();
</script>

<DropdownMenu.Trigger class={className} {...restProps}>
	{@render children?.()}
</DropdownMenu.Trigger>
