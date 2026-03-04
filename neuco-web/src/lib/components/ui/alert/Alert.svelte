<script lang="ts">
	import { cn } from '$lib/utils';
	import type { HTMLAttributes } from 'svelte/elements';

	type Variant = 'default' | 'destructive';

	interface Props extends HTMLAttributes<HTMLDivElement> {
		variant?: Variant;
		class?: string;
	}

	let { variant = 'default', class: className = '', children, ...restProps }: Props = $props();

	const variants: Record<Variant, string> = {
		default: 'bg-background text-foreground',
		destructive:
			'border-destructive/50 text-destructive dark:border-destructive [&>svg]:text-destructive'
	};

	const classes = $derived(
		cn(
			'relative w-full rounded-lg border border-border p-4 [&>svg~*]:pl-7 [&>svg+div]:translate-y-[-3px] [&>svg]:absolute [&>svg]:left-4 [&>svg]:top-4 [&>svg]:text-foreground',
			variants[variant],
			className
		)
	);
</script>

<div class={classes} role="alert" {...restProps}>
	{@render children?.()}
</div>
