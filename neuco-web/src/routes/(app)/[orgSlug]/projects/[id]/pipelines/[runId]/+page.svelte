<script lang="ts">
	import { page } from '$app/stores';
	import { usePipeline, useRetryPipeline } from '$lib/api/queries/pipelines';
	import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '$lib/components/ui/card';
	import { Button } from '$lib/components/ui/button';
	import { Skeleton } from '$lib/components/ui/skeleton';
	import { Alert, AlertTitle, AlertDescription } from '$lib/components/ui/alert';
	import { Separator } from '$lib/components/ui/separator';
	import { toast } from 'svelte-sonner';
	import {
		ArrowLeft,
		AlertCircle,
		RefreshCw,
		CheckCircle2,
		XCircle,
		Clock,
		Loader2,
		RotateCcw,
		Activity
	} from 'lucide-svelte';
	import type { PipelineStatus, PipelineType } from '$lib/api/types';

	const projectId = $derived($page.params.id ?? '');
	const runId = $derived($page.params.runId ?? '');
	const orgSlug = $derived($page.params.orgSlug ?? '');

	const pipelineQuery = $derived.by(() => usePipeline(projectId, runId));
	const retryMutation = $derived.by(() => useRetryPipeline(projectId));

	// Auto-refresh every 2 seconds when pipeline is running
	$effect(() => {
		const pipeline = pipelineQuery.data;
		if (!pipeline) return;

		const isActive = pipeline.status === 'running' || pipeline.status === 'pending';
		if (!isActive) return;

		const interval = setInterval(() => {
			pipelineQuery.refetch();
		}, 2000);

		return () => clearInterval(interval);
	});

	const hasFailedTasks = $derived(
		(pipelineQuery.data?.tasks ?? []).some((t) => t.status === 'failed')
	);

	const isRetryable = $derived(
		pipelineQuery.data?.status === 'failed' || hasFailedTasks
	);

	async function handleRetry() {
		try {
			await retryMutation.mutateAsync(runId);
			toast.success('Pipeline retry initiated');
		} catch (err) {
			toast.error('Failed to retry pipeline', {
				description: err instanceof Error ? err.message : 'An unexpected error occurred.'
			});
		}
	}

	const pipelineTypeLabels: Record<PipelineType, string> = {
		signal_ingestion: 'Signal Ingestion',
		candidate_extraction: 'Candidate Extraction',
		spec_generation: 'Spec Generation',
		code_generation: 'Code Generation',
		pr_creation: 'PR Creation'
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

	function taskStatusIcon(status: PipelineStatus) {
		switch (status) {
			case 'completed': return CheckCircle2;
			case 'failed': return XCircle;
			case 'running': return Loader2;
			default: return Clock;
		}
	}

	function taskStatusIconClass(status: PipelineStatus): string {
		switch (status) {
			case 'completed': return 'text-green-600 dark:text-green-400';
			case 'failed': return 'text-red-600 dark:text-red-400';
			case 'running': return 'text-blue-600 dark:text-blue-400 animate-spin';
			default: return 'text-muted-foreground';
		}
	}

	function taskBorderClass(status: PipelineStatus): string {
		switch (status) {
			case 'completed': return 'border-green-400 dark:border-green-600 bg-green-50 dark:bg-green-900/20';
			case 'failed': return 'border-red-400 dark:border-red-600 bg-red-50 dark:bg-red-900/20';
			case 'running': return 'border-blue-400 dark:border-blue-600 bg-blue-50 dark:bg-blue-900/20';
			default: return 'border-border bg-muted/30';
		}
	}

	function connectorClass(status: PipelineStatus): string {
		return status === 'completed'
			? 'bg-green-300 dark:bg-green-700'
			: 'bg-border';
	}

	function formatDate(dateStr: string | null | undefined): string {
		if (!dateStr) return '—';
		return new Date(dateStr).toLocaleDateString('en-US', {
			month: 'short',
			day: 'numeric',
			year: 'numeric',
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

	function humanizeTaskName(name: string): string {
		return name
			.replace(/_/g, ' ')
			.replace(/\b\w/g, (c) => c.toUpperCase());
	}
</script>

<svelte:head>
	<title>Pipeline Run — Neuco</title>
</svelte:head>

<div class="p-6 space-y-6">
	<!-- Back nav -->
	<div>
		<Button
			variant="ghost"
			size="sm"
			href={`/${orgSlug}/projects/${projectId}/pipelines`}
			class="gap-1.5 -ml-2 text-muted-foreground hover:text-foreground"
		>
			<ArrowLeft class="h-4 w-4" />
			Back to Pipelines
		</Button>
	</div>

	{#if pipelineQuery.isLoading}
		<div class="space-y-6">
			<div class="flex items-center justify-between">
				<div class="space-y-2">
					<Skeleton class="h-7 w-56"></Skeleton>
					<Skeleton class="h-4 w-40"></Skeleton>
				</div>
				<Skeleton class="h-9 w-28"></Skeleton>
			</div>
			<div class="grid gap-4 sm:grid-cols-3">
				{#each Array(3) as _, i (i)}
					<Card>
						<CardHeader class="pb-2"><Skeleton class="h-4 w-20"></Skeleton></CardHeader>
						<CardContent><Skeleton class="h-5 w-32"></Skeleton></CardContent>
					</Card>
				{/each}
			</div>
			<Card>
				<CardHeader><Skeleton class="h-5 w-32"></Skeleton></CardHeader>
				<CardContent class="space-y-4">
					{#each Array(4) as _, i (i)}
						<div class="flex gap-4">
							<Skeleton class="h-8 w-8 rounded-full shrink-0"></Skeleton>
							<div class="flex-1 space-y-2 pt-1">
								<Skeleton class="h-4 w-48"></Skeleton>
								<Skeleton class="h-3 w-32"></Skeleton>
							</div>
						</div>
					{/each}
				</CardContent>
			</Card>
		</div>
	{:else if pipelineQuery.isError}
		<Alert variant="destructive">
			<AlertCircle class="h-4 w-4" />
			<AlertTitle>Failed to load pipeline run</AlertTitle>
			<AlertDescription class="flex items-center justify-between">
				<span>{pipelineQuery.error?.message ?? 'An unexpected error occurred.'}</span>
				<Button
					variant="outline"
					size="sm"
					onclick={() => pipelineQuery.refetch()}
					class="ml-4 shrink-0"
				>
					<RefreshCw class="mr-1.5 h-3 w-3" />
					Retry
				</Button>
			</AlertDescription>
		</Alert>
	{:else if pipelineQuery.data}
		{@const pipeline = pipelineQuery.data}
		{@const isActive = pipeline.status === 'running' || pipeline.status === 'pending'}

		<!-- Header -->
		<div class="flex items-start justify-between gap-4">
			<div>
				<h1 class="text-2xl font-bold tracking-tight flex items-center gap-2">
					{pipelineTypeLabels[pipeline.type] ?? pipeline.type}
					{#if isActive}
						<Loader2 class="h-5 w-5 text-blue-500 animate-spin" />
					{/if}
				</h1>
				<p class="text-muted-foreground mt-1 font-mono text-sm">
					Run ID: {pipeline.id}
				</p>
			</div>

			<div class="flex items-center gap-2 shrink-0">
				{#if isActive}
					<Button
						variant="outline"
						size="sm"
						onclick={() => pipelineQuery.refetch()}
						class="gap-1.5"
						disabled={pipelineQuery.isFetching}
					>
						<RefreshCw class="h-3.5 w-3.5 {pipelineQuery.isFetching ? 'animate-spin' : ''}" />
						Refresh
					</Button>
				{/if}

				{#if isRetryable}
					<Button
						size="sm"
						onclick={handleRetry}
						disabled={retryMutation.isPending}
						class="gap-1.5"
					>
						{#if retryMutation.isPending}
							<Loader2 class="h-3.5 w-3.5 animate-spin" />
							Retrying...
						{:else}
							<RotateCcw class="h-3.5 w-3.5" />
							Retry Failed Tasks
						{/if}
					</Button>
				{/if}
			</div>
		</div>

		<!-- Summary cards -->
		<div class="grid gap-4 sm:grid-cols-3">
			<Card>
				<CardHeader class="pb-2">
					<CardTitle class="text-xs font-medium text-muted-foreground uppercase tracking-wide">
						Status
					</CardTitle>
				</CardHeader>
				<CardContent>
					<span class="inline-flex items-center rounded-full border px-2.5 py-0.5 text-xs font-semibold {statusClass(pipeline.status)}">
						{pipeline.status.charAt(0).toUpperCase() + pipeline.status.slice(1)}
					</span>
				</CardContent>
			</Card>

			<Card>
				<CardHeader class="pb-2">
					<CardTitle class="text-xs font-medium text-muted-foreground uppercase tracking-wide">
						Started
					</CardTitle>
				</CardHeader>
				<CardContent class="text-sm">
					{formatDate(pipeline.startedAt ?? pipeline.createdAt)}
				</CardContent>
			</Card>

			<Card>
				<CardHeader class="pb-2">
					<CardTitle class="text-xs font-medium text-muted-foreground uppercase tracking-wide">
						{pipeline.completedAt ? 'Duration' : 'Completed'}
					</CardTitle>
				</CardHeader>
				<CardContent class="text-sm">
					{#if pipeline.completedAt}
						{formatDuration(pipeline.startedAt ?? pipeline.createdAt, pipeline.completedAt)}
					{:else if isActive}
						<span class="flex items-center gap-1.5 text-blue-600 dark:text-blue-400">
							<Loader2 class="h-3.5 w-3.5 animate-spin" />
							In progress
						</span>
					{:else}
						—
					{/if}
				</CardContent>
			</Card>
		</div>

		{#if pipeline.errorMessage}
			<Alert variant="destructive">
				<AlertCircle class="h-4 w-4" />
				<AlertTitle>Pipeline error</AlertTitle>
				<AlertDescription>{pipeline.errorMessage}</AlertDescription>
			</Alert>
		{/if}

		<!-- Task timeline -->
		<Card>
			<CardHeader>
				<CardTitle class="flex items-center gap-2">
					<Activity class="h-4 w-4" />
					Task Timeline
				</CardTitle>
				<CardDescription>
					{#if pipeline.tasks && pipeline.tasks.length > 0}
						{pipeline.tasks.filter((t) => t.status === 'completed').length} of {pipeline.tasks.length} tasks completed
					{:else}
						No tasks recorded
					{/if}
				</CardDescription>
			</CardHeader>
			<CardContent>
				{#if !pipeline.tasks || pipeline.tasks.length === 0}
					<div class="flex flex-col items-center justify-center py-8 text-center">
						<Activity class="h-8 w-8 text-muted-foreground mb-3" />
						<p class="text-sm font-medium">No tasks recorded</p>
						<p class="text-xs text-muted-foreground mt-1">
							Task details will appear here once the pipeline starts executing.
						</p>
					</div>
				{:else}
					<div class="space-y-0">
						{#each pipeline.tasks as task, idx (task.id)}
							{@const TaskStatusIcon = taskStatusIcon(task.status)}
							<div class="flex gap-4">
								<!-- Timeline column -->
								<div class="flex flex-col items-center w-8 shrink-0">
									<div class="flex h-8 w-8 items-center justify-center rounded-full border-2 {taskBorderClass(task.status)}">
										<TaskStatusIcon class="h-4 w-4 {taskStatusIconClass(task.status)}" />
									</div>
									{#if idx < pipeline.tasks.length - 1}
										<div class="w-0.5 flex-1 min-h-4 my-1 {connectorClass(task.status)}"></div>
									{/if}
								</div>

								<!-- Task content -->
								<div class="flex-1 pb-5 min-w-0">
									<div class="flex items-start justify-between gap-2 min-h-8">
										<div>
											<p class="text-sm font-medium leading-none">
												{humanizeTaskName(task.name)}
											</p>
											{#if task.startedAt}
												<p class="text-xs text-muted-foreground mt-1">
													Started {formatDate(task.startedAt)}
												</p>
											{/if}
										</div>

										<div class="flex items-center gap-3 shrink-0 text-right">
											{#if task.startedAt && task.completedAt}
												<span class="text-xs text-muted-foreground">
													{formatDuration(task.startedAt, task.completedAt)}
												</span>
											{/if}
											{#if (task.metadata as Record<string, unknown>)?.attemptCount}
												<span class="text-xs text-muted-foreground">
													Attempt {(task.metadata as Record<string, unknown>).attemptCount}
												</span>
											{/if}
										</div>
									</div>

									{#if task.errorMessage}
										<Alert variant="destructive" class="mt-3 py-3">
											<AlertCircle class="h-3.5 w-3.5" />
											<AlertTitle class="text-xs font-medium">Task failed</AlertTitle>
											<AlertDescription class="text-xs mt-1">
												{task.errorMessage}
											</AlertDescription>
										</Alert>
									{/if}
								</div>
							</div>
						{/each}
					</div>
				{/if}
			</CardContent>
		</Card>

		{#if pipeline.metadata && Object.keys(pipeline.metadata).length > 0}
			<Card>
				<CardHeader>
					<CardTitle class="text-sm">Pipeline Metadata</CardTitle>
				</CardHeader>
				<CardContent>
					<pre class="text-xs font-mono bg-muted/50 rounded-md p-3 overflow-x-auto">{JSON.stringify(pipeline.metadata, null, 2)}</pre>
				</CardContent>
			</Card>
		{/if}
	{/if}
</div>
