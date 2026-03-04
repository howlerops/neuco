<script lang="ts">
	import { Dialog } from 'bits-ui';

	interface Props {
		class?: string;
		children?: import('svelte').Snippet;
	}

	let { class: className = '', children, ...restProps }: Props = $props();
</script>

<Dialog.Trigger class={className} {...restProps}>
	{@render children?.()}
</Dialog.Trigger>
