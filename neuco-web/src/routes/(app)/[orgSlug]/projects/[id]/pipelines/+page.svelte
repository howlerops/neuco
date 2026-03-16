<script lang="ts">
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import { usePipelines } from '$lib/api/queries/pipelines';
	import { Card, CardContent } from '$lib/components/ui/card';
	import { Badge } from '$lib/components/ui/badge';
	import { Button } from '$lib/components/ui/button';
	import { Skeleton } from '$lib/components/ui/skeleton';
	import { Alert, AlertTitle, AlertDescription } from '$lib/components/ui/alert';
	import { Separator } from '$lib/components/ui/separator';
	import {
		GitBranch,
		AlertCircle,
		RefreshCw,
		CheckCircle2,
		XCircle,
		Clock,
		Loader2,
		Activity,
		ChevronRight,
		Filter
	} from 'lucide-svelte';
	import type { PipelineRun, PipelineType, PipelineStatus } from '$lib/api/types-compat';

	const projectId = $derived($page.params.id ?? '');
	const orgSlug = $derived($page.params.orgSlug ?? '');

	let filterType = $state<string>('all');
	let filterStatus = $state<string>('all');
	let currentPage = $state(1);
	const PAGE_SIZE = 20;

	const pipelinesQuery = $derived.by(() => usePipelines(projectId, { page: currentPage, pageSize: PAGE_SIZE }));

	const pipelines = $derived(pipelinesQuery.data?.data ?? []);

	const filteredPipelines = $derived(
		pipelines.filter((p) => {
			const typeMatch = filterType === 'all' || p.type === filterType;
			const statusMatch = filterStatus === 'all' || p.status === filterStatus;
			return typeMatch && statusMatch;
		})
	);

	const pipelineTypeLabels: Record<PipelineType, string> = {
		signal_ingestion: 'Signal Ingestion',
		candidate_extraction: 'Candidate Extraction',
		spec_generation: 'Spec Generation',
		code_generation: 'Code Generation',
		pr_creation: 'PR Creation'
	};

	const pipelineTypeColors: Record<PipelineType, string> = {
		signal_ingestion: 'bg-cyan-100 text-cyan-800 dark:bg-cyan-900/30 dark:text-cyan-400 border-transparent',
		candidate_extraction: 'bg-violet-100 text-violet-800 dark:bg-violet-900/30 dark:text-violet-400 border-transparent',
		spec_generation: 'bg-amber-100 text-amber-800 dark:bg-amber-900/30 dark:text-amber-400 border-transparent',
		code_generation: 'bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-400 border-transparent',
		pr_creation: 'bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400 border-transparent'
	};

	function statusClass(status: PipelineStatus): string {
		switch (status) {
			case 'completed': return 'bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400 border-transparent';
			case 'running': return 'bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-400 border-transparent';
			case 'failed': return 'bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400 border-transparent';
			case 'cancelled': return 'bg-gray-100 text-gray-700 dark:bg-gray-800 dark:text-gray-400 border-transparent';
			default: return 'bg-gray-100 text-gray-700 dark:bg-gray-800 dark:text-gray-400 border-transparent';
		}
	}

	function statusIcon(status: PipelineStatus) {
		switch (status) {
			case 'completed': return CheckCircle2;
			case 'failed': return XCircle;
			case 'running': return Loader2;
			default: return Clock;
		}
	}

	function statusIconClass(status: PipelineStatus): string {
		switch (status) {
			case 'completed': return 'text-green-500';
			case 'failed': return 'text-red-500';
			case 'running': return 'text-blue-500 animate-spin';
			default: return 'text-muted-foreground';
		}
	}

	function taskProgress(pipeline: PipelineRun): string {
		if (!pipeline.tasks || pipeline.tasks.length === 0) return '—';
		const completed = pipeline.tasks.filter((t) => t.status === 'completed').length;
		return `${completed}/${pipeline.tasks.length} tasks`;
	}

	function formatRelativeTime(dateStr: string): string {
		if (!dateStr) return '—';
		const date = new Date(dateStr);
		const now = new Date();
		const diffMs = now.getTime() - date.getTime();
		const diffMins = Math.floor(diffMs / 60000);
		const diffHours = Math.floor(diffMins / 60);
		const diffDays = Math.floor(diffHours / 24);
		if (diffMins < 1) return 'Just now';
		if (diffMins < 60) return `${diffMins}m ago`;
		if (diffHours < 24) return `${diffHours}h ago`;
		return `${diffDays}d ago`;
	}

	function formatDate(dateStr: string): string {
		if (!dateStr) return '—';
		return new Date(dateStr).toLocaleDateString('en-US', {
			month: 'short',
			day: 'numeric',
			hour: '2-digit',
			minute: '2-digit'
		});
	}

	function formatDuration(start: string, end: string): string {
		if (!start || !end) return '—';
		const diffMs = new Date(end).getTime() - new Date(start).getTime();
		if (diffMs < 0) return '—';
		const secs = Math.floor(diffMs / 1000);
		if (secs < 60) return `${secs}s`;
		const mins = Math.floor(secs / 60);
		return `${mins}m ${secs % 60}s`;
	}

	function navigateToDetail(runId: string) {
		goto(`/${orgSlug}/projects/${projectId}/pipelines/${runId}`);
	}

	const pipelineTypes: Array<{ value: string; label: string }> = [
		{ value: 'all', label: 'All Types' },
		{ value: 'signal_ingestion', label: 'Signal Ingestion' },
		{ value: 'candidate_extraction', label: 'Candidate Extraction' },
		{ value: 'spec_generation', label: 'Spec Generation' },
		{ value: 'code_generation', label: 'Code Generation' },
		{ value: 'pr_creation', label: 'PR Creation' }
	];

	const pipelineStatuses: Array<{ value: string; label: string }> = [
		{ value: 'all', label: 'All Statuses' },
		{ value: 'pending', label: 'Pending' },
		{ value: 'running', label: 'Running' },
		{ value: 'completed', label: 'Completed' },
		{ value: 'failed', label: 'Failed' },
		{ value: 'cancelled', label: 'Cancelled' }
	];
</script>

<svelte:head>
	<title>Pipelines — Neuco</title>
</svelte:head>

<div class="p-6 space-y-6">
	<!-- Header -->
	<div class="flex items-center justify-between">
		<div>
			<h1 class="text-2xl font-bold tracking-tight">Pipelines</h1>
			<p class="text-muted-foreground mt-1">
				Activity feed for all pipeline runs in this project
			</p>
		</div>
		<Button
			variant="outline"
			size="sm"
			onclick={() => pipelinesQuery.refetch()}
			class="gap-1.5"
			disabled={pipelinesQuery.isFetching}
		>
			<RefreshCw class="h-3.5 w-3.5 {pipelinesQuery.isFetching ? 'animate-spin' : ''}" />
			Refresh
		</Button>
	</div>

	<!-- Filter bar -->
	<div class="flex items-center gap-3 flex-wrap">
		<div class="flex items-center gap-1.5 text-sm text-muted-foreground">
			<Filter class="h-3.5 w-3.5" />
			<span>Filter:</span>
		</div>

		<select
			bind:value={filterType}
			class="h-8 rounded-md border border-input bg-background px-3 py-1 text-sm ring-offset-background focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2"
		>
			{#each pipelineTypes as opt (opt.value)}
				<option value={opt.value}>{opt.label}</option>
			{/each}
		</select>

		<select
			bind:value={filterStatus}
			class="h-8 rounded-md border border-input bg-background px-3 py-1 text-sm ring-offset-background focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2"
		>
			{#each pipelineStatuses as opt (opt.value)}
				<option value={opt.value}>{opt.label}</option>
			{/each}
		</select>

		{#if filterType !== 'all' || filterStatus !== 'all'}
			<Button
				variant="ghost"
				size="sm"
				onclick={() => { filterType = 'all'; filterStatus = 'all'; }}
				class="h-8 text-xs text-muted-foreground"
			>
				Clear filters
			</Button>
		{/if}
	</div>

	<!-- Pipeline list -->
	{#if pipelinesQuery.isLoading}
		<div class="space-y-3">
			{#each Array(6) as _, i (i)}
				<Card>
					<CardContent class="p-4">
						<div class="flex items-center gap-4">
							<Skeleton class="h-9 w-9 rounded-full shrink-0"></Skeleton>
							<div class="flex-1 space-y-2">
								<div class="flex items-center gap-3">
									<Skeleton class="h-5 w-28 rounded-full"></Skeleton>
									<Skeleton class="h-5 w-20 rounded-full"></Skeleton>
								</div>
								<div class="flex items-center gap-3">
									<Skeleton class="h-3.5 w-32"></Skeleton>
									<Skeleton class="h-3.5 w-24"></Skeleton>
									<Skeleton class="h-3.5 w-20"></Skeleton>
								</div>
							</div>
							<Skeleton class="h-5 w-5 rounded shrink-0"></Skeleton>
						</div>
					</CardContent>
				</Card>
			{/each}
		</div>
	{:else if pipelinesQuery.isError}
		<Alert variant="destructive">
			<AlertCircle class="h-4 w-4" />
			<AlertTitle>Failed to load pipelines</AlertTitle>
			<AlertDescription class="flex items-center justify-between">
				<span>{pipelinesQuery.error?.message ?? 'An unexpected error occurred.'}</span>
				<Button
					variant="outline"
					size="sm"
					onclick={() => pipelinesQuery.refetch()}
					class="ml-4 shrink-0"
				>
					<RefreshCw class="mr-1.5 h-3 w-3" />
					Retry
				</Button>
			</AlertDescription>
		</Alert>
	{:else if filteredPipelines.length === 0}
		<div class="flex flex-col items-center justify-center rounded-xl border-2 border-dashed border-border py-20 text-center">
			<Activity class="h-12 w-12 text-muted-foreground mb-4" />
			<h3 class="text-lg font-semibold">
				{pipelines.length === 0 ? 'No pipeline runs yet' : 'No matching pipelines'}
			</h3>
			<p class="text-sm text-muted-foreground mt-2 max-w-sm">
				{#if pipelines.length === 0}
					Pipeline runs will appear here once your project starts processing signals.
				{:else}
					Try adjusting your filters to see more results.
				{/if}
			</p>
			{#if filterType !== 'all' || filterStatus !== 'all'}
				<Button
					variant="outline"
					size="sm"
					onclick={() => { filterType = 'all'; filterStatus = 'all'; }}
					class="mt-4"
				>
					Clear filters
				</Button>
			{/if}
		</div>
	{:else}
		<div class="space-y-2">
			{#each filteredPipelines as pipeline (pipeline.id)}
				{@const StatusIcon = statusIcon(pipeline.status)}
				<Card
					class="cursor-pointer hover:shadow-sm transition-shadow group"
					onclick={() => navigateToDetail(pipeline.id)}
				>
					<CardContent class="p-4">
						<div class="flex items-center gap-4">
							<!-- Status icon -->
							<div class="shrink-0">
								<StatusIcon class="h-5 w-5 {statusIconClass(pipeline.status)}" />
							</div>

							<!-- Main info -->
							<div class="flex-1 min-w-0 space-y-1.5">
								<div class="flex items-center gap-2 flex-wrap">
									<span class="inline-flex items-center rounded-full border px-2.5 py-0.5 text-xs font-semibold {pipelineTypeColors[pipeline.type]}">
										{pipelineTypeLabels[pipeline.type]}
									</span>
									<span class="inline-flex items-center rounded-full border px-2.5 py-0.5 text-xs font-semibold {statusClass(pipeline.status)}">
										{pipeline.status.charAt(0).toUpperCase() + pipeline.status.slice(1)}
									</span>
									{#if pipeline.tasks && pipeline.tasks.length > 0}
										<span class="text-xs text-muted-foreground">
											{taskProgress(pipeline)}
										</span>
									{/if}
								</div>

								<div class="flex items-center gap-4 text-xs text-muted-foreground flex-wrap">
									{#if pipeline.startedAt}
										<span>Started {formatRelativeTime(pipeline.startedAt)}</span>
									{:else}
										<span>Created {formatRelativeTime(pipeline.createdAt)}</span>
									{/if}

									{#if pipeline.completedAt}
										<Separator orientation="vertical" class="h-3" />
										<span>Completed {formatDate(pipeline.completedAt)}</span>
										<Separator orientation="vertical" class="h-3" />
										<span>Duration: {formatDuration(pipeline.startedAt, pipeline.completedAt)}</span>
									{/if}

									{#if pipeline.errorMessage}
										<Separator orientation="vertical" class="h-3" />
										<span class="text-red-600 dark:text-red-400 truncate max-w-[240px]">
											{pipeline.errorMessage.slice(0, 80)}{pipeline.errorMessage.length > 80 ? '...' : ''}
										</span>
									{/if}
								</div>
							</div>

							<!-- Chevron -->
							<ChevronRight class="h-4 w-4 text-muted-foreground shrink-0 opacity-0 group-hover:opacity-100 transition-opacity" />
						</div>
					</CardContent>
				</Card>
			{/each}
		</div>

		<!-- Pagination -->
		{#if pipelinesQuery.data && pipelinesQuery.data.totalPages > 1}
			<div class="flex items-center justify-between pt-2">
				<p class="text-sm text-muted-foreground">
					Page {pipelinesQuery.data.page} of {pipelinesQuery.data.totalPages}
					({pipelinesQuery.data.total} total)
				</p>
				<div class="flex gap-2">
					<Button
						variant="outline"
						size="sm"
						disabled={currentPage <= 1}
						onclick={() => currentPage--}
					>
						Previous
					</Button>
					<Button
						variant="outline"
						size="sm"
						disabled={currentPage >= pipelinesQuery.data.totalPages}
						onclick={() => currentPage++}
					>
						Next
					</Button>
				</div>
			</div>
		{/if}
	{/if}
</div>
