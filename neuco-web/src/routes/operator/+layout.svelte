<script lang="ts">
	import { page } from '$app/stores';
	import { cn } from '$lib/utils';
	import { QueryClientProvider, QueryClient } from '@tanstack/svelte-query';
	import '../../app.css';

	let { children } = $props();

	const queryClient = new QueryClient({
		defaultOptions: {
			queries: { retry: 1, staleTime: 30_000 }
		}
	});

	const navItems = [
		{ label: 'Organizations', href: '/operator/orgs' },
		{ label: 'Users', href: '/operator/users' },
		{ label: 'Flags', href: '/operator/flags' },
		{ label: 'Health', href: '/operator/health' }
	];

	let currentPath = $derived($page.url.pathname);
</script>

<QueryClientProvider client={queryClient}>
	<div class="min-h-screen bg-background">
		<header class="border-b bg-card">
			<div class="mx-auto max-w-7xl px-6 py-4 flex items-center justify-between">
				<div class="flex items-center gap-6">
					<h1 class="text-lg font-semibold">Neuco Operator</h1>
					<nav class="flex gap-1">
						{#each navItems as item}
							{@const isActive = currentPath.startsWith(item.href)}
							<a
								href={item.href}
								class={cn(
									'px-3 py-1.5 rounded-md text-sm font-medium transition-colors',
									isActive
										? 'bg-primary text-primary-foreground'
										: 'text-muted-foreground hover:text-foreground hover:bg-accent'
								)}
							>
								{item.label}
							</a>
						{/each}
					</nav>
				</div>
				<a href="/" class="text-sm text-muted-foreground hover:text-foreground">
					Back to App
				</a>
			</div>
		</header>

		<main class="mx-auto max-w-7xl px-6 py-8">
			{@render children()}
		</main>
	</div>
</QueryClientProvider>
