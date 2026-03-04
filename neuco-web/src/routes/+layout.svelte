<script lang="ts">
	import '../app.css';
	import favicon from '$lib/assets/favicon.svg';
	import { QueryClient, QueryClientProvider } from '@tanstack/svelte-query';
	import { ModeWatcher } from 'mode-watcher';
	import { Sonner } from '$lib/components/ui/sonner';

	let { children } = $props();

	const queryClient = new QueryClient({
		defaultOptions: {
			queries: {
				staleTime: 30 * 1000,
				gcTime: 5 * 60 * 1000,
				retry: (failureCount, error) => {
					if (error && typeof error === 'object' && 'status' in error) {
						const status = (error as { status: number }).status;
						if (status === 401 || status === 403 || status === 404) {
							return false;
						}
					}
					return failureCount < 2;
				}
			}
		}
	});
</script>

<svelte:head>
	<link rel="icon" href={favicon} />
</svelte:head>

<ModeWatcher />
<QueryClientProvider client={queryClient}>
	{@render children()}
</QueryClientProvider>
<Sonner />
