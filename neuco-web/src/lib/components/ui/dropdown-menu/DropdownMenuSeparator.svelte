<script lang="ts">
	import { cn } from '$lib/utils';
	import { DropdownMenu } from 'bits-ui';

	interface Props {
		class?: string;
	}

	let { class: className = '', ...restProps }: Props = $props();
</script>

<DropdownMenu.Separator class={cn('-mx-1 my-1 h-px bg-muted', className)} {...restProps} />
