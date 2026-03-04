<script lang="ts">
	import { cn } from '$lib/utils';

	type Variant = 'default' | 'destructive' | 'outline' | 'secondary' | 'ghost' | 'link';
	type Size = 'default' | 'sm' | 'lg' | 'icon';

	interface BaseProps {
		variant?: Variant;
		size?: Size;
		class?: string;
		disabled?: boolean;
		children?: import('svelte').Snippet;
	}

	interface ButtonProps extends BaseProps {
		href?: undefined;
		onclick?: (e: MouseEvent) => void;
		type?: 'button' | 'submit' | 'reset';
		form?: string;
	}

	interface AnchorProps extends BaseProps {
		href: string;
		onclick?: (e: MouseEvent) => void;
	}

	type Props = ButtonProps | AnchorProps;

	let {
		variant = 'default',
		size = 'default',
		class: className = '',
		href,
		disabled = false,
		onclick,
		children,
		...restProps
	}: Props = $props();

	const base =
		'inline-flex items-center justify-center gap-2 whitespace-nowrap rounded-md text-sm font-medium transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:pointer-events-none disabled:opacity-50 [&_svg]:pointer-events-none [&_svg]:size-4 [&_svg]:shrink-0';

	const variants: Record<Variant, string> = {
		default: 'bg-primary text-primary-foreground hover:bg-primary/90',
		destructive: 'bg-destructive text-destructive-foreground hover:bg-destructive/90',
		outline:
			'border border-input bg-background hover:bg-accent hover:text-accent-foreground',
		secondary: 'bg-secondary text-secondary-foreground hover:bg-secondary/80',
		ghost: 'hover:bg-accent hover:text-accent-foreground',
		link: 'text-primary underline-offset-4 hover:underline'
	};

	const sizes: Record<Size, string> = {
		default: 'h-10 px-4 py-2',
		sm: 'h-9 rounded-md px-3',
		lg: 'h-11 rounded-md px-8',
		icon: 'h-10 w-10'
	};

	const classes = $derived(cn(base, variants[variant], sizes[size], className));
</script>

{#if href}
	<a
		{href}
		class={classes}
		aria-disabled={disabled}
		tabindex={disabled ? -1 : undefined}
		{...restProps as Record<string, unknown>}
	>
		{@render children?.()}
	</a>
{:else}
	<button
		class={classes}
		{disabled}
		{onclick}
		{...restProps as Record<string, unknown>}
	>
		{@render children?.()}
	</button>
{/if}
