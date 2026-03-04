<script lang="ts">
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import { useGenerations } from '$lib/api/queries/generations';
	import {
		Table,
		TableBody,
		TableCell,
		TableHead,
		TableHeader,
		TableRow
	} from '$lib/components/ui/table';
	import { Badge } from '$lib/components/ui/badge';
	import { Button } from '$lib/components/ui/button';
	import { Skeleton } from '$lib/components/ui/skeleton';
	import { Alert, AlertTitle, AlertDescription } from '$lib/components/ui/alert';
	import {
		Code2,
		ExternalLink,
		GitBranch,
		Clock,
		AlertCircle,
		RefreshCw,
		GitPullRequest
	} from 'lucide-svelte';
	import type { Generation } from '$lib/api/types';

	const projectId = $derived($page.params.id ?? '');
	const orgSlug = $derived($page.params.orgSlug ?? '');

	const generationsQuery = $derived.by(() => useGenerations(projectId));

	const generations = $derived(generationsQuery.data?.data ?? []);

	function formatDate(dateStr: string): string {
		if (!dateStr) return '—';
		return new Date(dateStr).toLocaleDateString('en-US', {
			month: 'short',
			day: 'numeric',
			year: 'numeric',
			hour: '2-digit',
			minute: '2-digit'
		});
	}

	function formatDuration(createdAt: string, updatedAt: string, status: string): string {
		if (!createdAt) return '—';
		if (status === 'pending' || status === 'running') return 'In progress';
		const start = new Date(createdAt).getTime();
		const end = new Date(updatedAt).getTime();
		const diffMs = end - start;
		if (diffMs < 0) return '—';
		const secs = Math.floor(diffMs / 1000);
		if (secs < 60) return `${secs}s`;
		const mins = Math.floor(secs / 60);
		const remainingSecs = secs % 60;
		return `${mins}m ${remainingSecs}s`;
	}

	function statusVariant(status: Generation['status']): 'default' | 'secondary' | 'destructive' | 'outline' {
		switch (status) {
			case 'completed': return 'default';
			case 'running': return 'secondary';
			case 'failed': return 'destructive';
			default: return 'outline';
		}
	}

	function statusLabel(status: Generation['status']): string {
		switch (status) {
			case 'pending': return 'Pending';
			case 'running': return 'Running';
			case 'completed': return 'Completed';
			case 'failed': return 'Failed';
			default: return status;
		}
	}

	function statusClass(status: Generation['status']): string {
		switch (status) {
			case 'completed': return 'bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400 border-transparent';
			case 'running': return 'bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-400 border-transparent';
			case 'failed': return 'bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400 border-transparent';
			default: return 'bg-gray-100 text-gray-700 dark:bg-gray-800 dark:text-gray-400 border-transparent';
		}
	}

	function navigateToDetail(generationId: string) {
		goto(`/${orgSlug}/projects/${projectId}/generations/${generationId}`);
	}
</script>

<svelte:head>
	<title>Generations — Neuco</title>
</svelte:head>

<div class="p-6 space-y-6">
	<div class="flex items-center justify-between">
		<div>
			<h1 class="text-2xl font-bold tracking-tight">Generations</h1>
			<p class="text-muted-foreground mt-1">
				Code generations produced from approved specs
			</p>
		</div>
	</div>

	{#if generationsQuery.isLoading}
		<div class="rounded-lg border border-border overflow-hidden">
			<div class="bg-muted/30 px-4 py-3 flex gap-6">
				{#each Array(6) as _, i (i)}
					<Skeleton class="h-4 w-24"></Skeleton>
				{/each}
			</div>
			<div class="divide-y divide-border">
				{#each Array(5) as _, i (i)}
					<div class="px-4 py-4 flex gap-6 items-center">
						<Skeleton class="h-4 w-40"></Skeleton>
						<Skeleton class="h-5 w-20 rounded-full"></Skeleton>
						<Skeleton class="h-4 w-32"></Skeleton>
						<Skeleton class="h-4 w-24"></Skeleton>
						<Skeleton class="h-4 w-28"></Skeleton>
						<Skeleton class="h-4 w-16"></Skeleton>
					</div>
				{/each}
			</div>
		</div>
	{:else if generationsQuery.isError}
		<Alert variant="destructive">
			<AlertCircle class="h-4 w-4" />
			<AlertTitle>Failed to load generations</AlertTitle>
			<AlertDescription class="flex items-center justify-between">
				<span>{generationsQuery.error?.message ?? 'An unexpected error occurred.'}</span>
				<Button
					variant="outline"
					size="sm"
					onclick={() => generationsQuery.refetch()}
					class="ml-4 shrink-0"
				>
					<RefreshCw class="mr-1.5 h-3 w-3" />
					Retry
				</Button>
			</AlertDescription>
		</Alert>
	{:else if generations.length === 0}
		<div class="flex flex-col items-center justify-center rounded-xl border-2 border-dashed border-border py-20 text-center">
			<Code2 class="h-12 w-12 text-muted-foreground mb-4" />
			<h3 class="text-lg font-semibold">No generations yet</h3>
			<p class="text-sm text-muted-foreground mt-2 max-w-sm">
				No generations yet. Generate code from a spec to see results here.
			</p>
		</div>
	{:else}
		<div class="rounded-lg border border-border overflow-hidden">
			<Table>
				<TableHeader>
					<TableRow class="bg-muted/30">
						<TableHead class="w-[260px]">Spec</TableHead>
						<TableHead class="w-[130px]">Status</TableHead>
						<TableHead class="w-[180px]">Branch</TableHead>
						<TableHead class="w-[100px]">PR</TableHead>
						<TableHead class="w-[180px]">Created</TableHead>
						<TableHead class="w-[120px]">Duration</TableHead>
					</TableRow>
				</TableHeader>
				<TableBody>
					{#each generations as generation (generation.id)}
						<TableRow
							class="cursor-pointer hover:bg-muted/40 transition-colors"
							onclick={() => navigateToDetail(generation.id)}
						>
							<TableCell class="font-medium">
								<a
									href={`/${orgSlug}/projects/${projectId}/generations/${generation.id}`}
									class="hover:underline text-foreground truncate block max-w-[240px]"
									onclick={(e) => e.stopPropagation()}
								>
									Generation {generation.id.slice(0, 8)}
								</a>
							</TableCell>

							<TableCell>
								<span class="inline-flex items-center rounded-full border px-2.5 py-0.5 text-xs font-semibold {statusClass(generation.status)}">
									{statusLabel(generation.status)}
								</span>
							</TableCell>

							<TableCell>
								{#if generation.prUrl}
									<div class="flex items-center gap-1.5 text-sm text-muted-foreground font-mono">
										<GitBranch class="h-3.5 w-3.5 shrink-0" />
										<span class="truncate max-w-[150px]">branch-{generation.id.slice(0, 6)}</span>
									</div>
								{:else}
									<span class="text-muted-foreground text-sm">—</span>
								{/if}
							</TableCell>

							<TableCell>
								{#if generation.prUrl}
									<a
										href={generation.prUrl}
										target="_blank"
										rel="noopener noreferrer"
										class="inline-flex items-center gap-1 text-sm text-blue-600 hover:text-blue-500 dark:text-blue-400"
										onclick={(e) => e.stopPropagation()}
									>
										<GitPullRequest class="h-3.5 w-3.5" />
										#{generation.prNumber}
										<ExternalLink class="h-3 w-3 opacity-60" />
									</a>
								{:else}
									<span class="text-muted-foreground text-sm">—</span>
								{/if}
							</TableCell>

							<TableCell>
								<div class="flex items-center gap-1.5 text-sm text-muted-foreground">
									<Clock class="h-3.5 w-3.5 shrink-0" />
									{formatDate(generation.createdAt)}
								</div>
							</TableCell>

							<TableCell class="text-sm text-muted-foreground">
								{formatDuration(generation.createdAt, generation.updatedAt, generation.status)}
							</TableCell>
						</TableRow>
					{/each}
				</TableBody>
			</Table>
		</div>

		{#if generationsQuery.data && generationsQuery.data.totalPages > 1}
			<div class="flex items-center justify-between">
				<p class="text-sm text-muted-foreground">
					Showing {generations.length} of {generationsQuery.data.total} generations
				</p>
				<div class="flex gap-2">
					<Button
						variant="outline"
						size="sm"
						disabled={generationsQuery.data.page <= 1}
					>
						Previous
					</Button>
					<Button
						variant="outline"
						size="sm"
						disabled={generationsQuery.data.page >= generationsQuery.data.totalPages}
					>
						Next
					</Button>
				</div>
			</div>
		{/if}
	{/if}
</div>
