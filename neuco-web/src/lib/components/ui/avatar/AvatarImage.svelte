<script lang="ts">
	import { cn } from '$lib/utils';
	import { Avatar } from 'bits-ui';

	interface Props {
		class?: string;
		src?: string;
		alt?: string;
	}

	let { class: className = '', src = '', alt = '', ...restProps }: Props = $props();
</script>

<Avatar.Image
	{src}
	{alt}
	class={cn('aspect-square h-full w-full', className)}
	{...restProps}
/>
