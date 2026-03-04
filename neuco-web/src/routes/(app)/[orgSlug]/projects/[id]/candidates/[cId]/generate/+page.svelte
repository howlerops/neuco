<script lang="ts">
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import { useGenerateCode, useGeneration } from '$lib/api/queries/generations';
	import { useSpecByCandidate } from '$lib/api/queries/specs';
	import { usePipeline } from '$lib/api/queries/pipelines';
	import { Button } from '$lib/components/ui/button';
	import { Badge } from '$lib/components/ui/badge';
	import { Card, CardContent, CardHeader, CardTitle } from '$lib/components/ui/card';
	import { Skeleton } from '$lib/components/ui/skeleton';
	import { Alert, AlertDescription, AlertTitle } from '$lib/components/ui/alert';
	import { Separator } from '$lib/components/ui/separator';
	import { toast } from '$lib/components/ui/sonner';
	import {
		Clock,
		CheckCircle2,
		XCircle,
		AlertCircle,
		GitPullRequest,
		FileCode2,
		RotateCcw,
		ExternalLink,
		ChevronLeft,
		Loader2
	} from 'lucide-svelte';
	import { cn } from '$lib/utils';
	import type { PipelineStatus } from '$lib/api/types';

	const projectId = $derived($page.params.id ?? '');
	const candidateId = $derived($page.params.cId ?? '');
	const orgSlug = $derived($page.params.orgSlug ?? '');

	// ── Fetch spec to get specId ───────────────────────────────────────────────
	const specQuery = $derived(useSpecByCandidate(projectId, candidateId));

	// ── Generate code mutation ─────────────────────────────────────────────────
	const generateMutation = $derived(useGenerateCode(projectId));

	// ── Reactive generation tracking ──────────────────────────────────────────
	let generationId = $state('');
	let pipelineId = $state('');
	let hasTriggered = $state(false);
	let triggerError = $state('');

	// ── Generation polling ────────────────────────────────────────────────────
	const generationQuery = $derived(
		generationId ? useGeneration(projectId, generationId) : null
	);

	const generation = $derived(generationQuery?.data ?? null);
	const isComplete = $derived(
		generation?.status === 'completed' || generation?.status === 'failed'
	);

	// ── Pipeline polling ──────────────────────────────────────────────────────
	const pipelineQuery = $derived(pipelineId ? usePipeline(projectId, pipelineId) : null);

	// ── Pipeline steps definition ─────────────────────────────────────────────
	type StepName =
		| 'fetch_spec'
		| 'index_repo'
		| 'build_context'
		| 'generate_code'
		| 'create_pr'
		| 'notify';

	const PIPELINE_STEPS: { name: StepName; label: string }[] = [
		{ name: 'fetch_spec', label: 'Fetch Spec' },
		{ name: 'index_repo', label: 'Index Repository' },
		{ name: 'build_context', label: 'Build Context' },
		{ name: 'generate_code', label: 'Generate Code' },
		{ name: 'create_pr', label: 'Create Pull Request' },
		{ name: 'notify', label: 'Send Notifications' }
	];

	// Map pipeline tasks by name — use a plain $state map updated reactively
	let tasksByName = $state(new Map<string, { name: string; status: string; startedAt: string; completedAt: string; errorMessage: string; id: string; pipelineId: string; metadata: Record<string, unknown> }>());

	$effect(() => {
		const tasks = pipelineQuery?.data?.tasks ?? [];
		const map = new Map<string, (typeof tasks)[0]>();
		for (const task of tasks) {
			map.set(task.name, task);
		}
		tasksByName = map;
	});

	// ── Display status ─────────────────────────────────────────────────────────
	type DisplayStatus = 'idle' | 'starting' | 'running' | 'completed' | 'failed';

	const displayStatus = $derived(
		!hasTriggered
			? ('idle' as DisplayStatus)
			: generateMutation.isPending && !generationId
				? ('starting' as DisplayStatus)
				: !generation
					? ('starting' as DisplayStatus)
					: generation.status === 'completed'
						? ('completed' as DisplayStatus)
						: generation.status === 'failed'
							? ('failed' as DisplayStatus)
							: ('running' as DisplayStatus)
	);

	// ── Trigger generation once spec is loaded ────────────────────────────────
	$effect(() => {
		const spec = specQuery.data;
		if (spec && !hasTriggered && !generateMutation.isPending) {
			hasTriggered = true;
			triggerError = '';
			generateMutation.mutate(
				{ specId: spec.id },
				{
					onSuccess: (gen) => {
						generationId = gen.id;
					},
					onError: (err) => {
						triggerError = err.message ?? 'Failed to start code generation';
						hasTriggered = false;
					}
				}
			);
		}
	});

	// ── Helpers ───────────────────────────────────────────────────────────────
	function getStepStatus(stepName: string): PipelineStatus {
		const task = tasksByName.get(stepName);
		if (!task) return 'pending';
		return task.status as PipelineStatus;
	}

	function getStepDuration(stepName: string): string | null {
		const task = tasksByName.get(stepName);
		if (!task?.startedAt || !task?.completedAt) return null;
		const ms =
			new Date(task.completedAt).getTime() - new Date(task.startedAt).getTime();
		if (ms < 1000) return `${ms}ms`;
		return `${(ms / 1000).toFixed(1)}s`;
	}

	function handleRetry() {
		hasTriggered = false;
		generationId = '';
		pipelineId = '';
		triggerError = '';
	}

	function handleBack() {
		goto(`/${orgSlug}/projects/${projectId}/candidates/${candidateId}/spec`);
	}
</script>

<svelte:head>
	<title>Generate Code — Neuco</title>
</svelte:head>

<div class="flex flex-col gap-6 p-6 max-w-3xl mx-auto">
	<!-- Back navigation -->
	<div>
		<Button
			variant="ghost"
			size="sm"
			class="gap-1.5 -ml-2 text-muted-foreground"
			onclick={handleBack}
		>
			<ChevronLeft class="h-4 w-4" />
			Back to Spec
		</Button>
	</div>

	<!-- Header -->
	<div>
		<h2 class="text-lg font-semibold tracking-tight">Code Generation</h2>
		<p class="text-sm text-muted-foreground mt-0.5">
			Generating implementation code from the feature spec
		</p>
	</div>

	<!-- Spec loading state -->
	{#if specQuery.isLoading}
		<Card>
			<CardContent class="p-6 space-y-3">
				<Skeleton class="h-5 w-48" />
				<Skeleton class="h-4 w-full max-w-sm" />
			</CardContent>
		</Card>
	{:else if specQuery.isError}
		<Alert variant="destructive">
			<AlertCircle class="h-4 w-4" />
			<AlertTitle>Spec not found</AlertTitle>
			<AlertDescription class="flex items-center gap-3">
				{specQuery.error?.message ?? 'Could not load the spec for this candidate.'}
				<Button variant="outline" size="sm" class="h-7 text-xs" onclick={handleBack}>
					Back to Spec
				</Button>
			</AlertDescription>
		</Alert>
	{:else}
		<!-- Trigger error -->
		{#if triggerError}
			<Alert variant="destructive">
				<AlertCircle class="h-4 w-4" />
				<AlertTitle>Generation failed to start</AlertTitle>
				<AlertDescription class="flex items-center gap-3">
					{triggerError}
					<Button variant="outline" size="sm" class="h-7 text-xs" onclick={handleRetry}>
						<RotateCcw class="h-3.5 w-3.5 mr-1.5" />
						Retry
					</Button>
				</AlertDescription>
			</Alert>
		{/if}

		<!-- Pipeline timeline -->
		<Card>
			<CardHeader class="pb-4">
				<CardTitle class="text-base flex items-center gap-2">
					Pipeline Progress
					{#if displayStatus === 'running' || displayStatus === 'starting'}
						<Loader2 class="h-4 w-4 animate-spin text-primary" />
					{:else if displayStatus === 'completed'}
						<CheckCircle2 class="h-4 w-4 text-green-500" />
					{:else if displayStatus === 'failed'}
						<XCircle class="h-4 w-4 text-destructive" />
					{/if}
				</CardTitle>
			</CardHeader>
			<CardContent class="pb-6">
				<div class="space-y-0">
					{#each PIPELINE_STEPS as step, idx (step.name)}
						{@const stepStatus = getStepStatus(step.name)}
						{@const duration = getStepDuration(step.name)}
						{@const isLast = idx === PIPELINE_STEPS.length - 1}

						<div class="flex items-start gap-4">
							<!-- Icon + connector column -->
							<div class="flex flex-col items-center">
								<div
									class={cn(
										'flex h-8 w-8 shrink-0 items-center justify-center rounded-full border-2 z-10',
										stepStatus === 'completed'
											? 'bg-green-500 border-green-500 text-white'
											: stepStatus === 'running'
												? 'bg-primary border-primary text-primary-foreground'
												: stepStatus === 'failed'
													? 'bg-destructive border-destructive text-destructive-foreground'
													: stepStatus === 'cancelled'
														? 'bg-muted border-muted-foreground/30 text-muted-foreground'
														: 'bg-background border-muted-foreground/30 text-muted-foreground'
									)}
								>
									{#if stepStatus === 'completed'}
										<CheckCircle2 class="h-4 w-4" />
									{:else if stepStatus === 'running'}
										<Loader2 class="h-4 w-4 animate-spin" />
									{:else if stepStatus === 'failed' || stepStatus === 'cancelled'}
										<XCircle class="h-4 w-4" />
									{:else}
										<Clock class="h-3.5 w-3.5" />
									{/if}
								</div>
								{#if !isLast}
									<div
										class={cn(
											'w-0.5 flex-1 min-h-[24px]',
											stepStatus === 'completed' ? 'bg-green-500' : 'bg-border'
										)}
									></div>
								{/if}
							</div>

							<!-- Step label + info -->
							<div class="flex flex-1 items-center justify-between pb-6">
								<div>
									<p
										class={cn(
											'text-sm font-medium',
											stepStatus === 'running' && 'text-primary',
											stepStatus === 'pending' && 'text-muted-foreground',
											stepStatus === 'completed' && 'text-foreground',
											stepStatus === 'failed' && 'text-destructive'
										)}
									>
										{step.label}
									</p>
									{#if stepStatus === 'running'}
										<p class="text-xs text-muted-foreground mt-0.5 animate-pulse">
											Running…
										</p>
									{:else if stepStatus === 'failed'}
										{@const task = tasksByName.get(step.name)}
										{#if task?.errorMessage}
											<p class="text-xs text-destructive mt-0.5">{task.errorMessage}</p>
										{/if}
									{/if}
								</div>
								{#if duration}
									<span class="text-xs text-muted-foreground tabular-nums">{duration}</span>
								{:else if stepStatus === 'pending' && (displayStatus === 'idle' || displayStatus === 'starting')}
									<span class="text-xs text-muted-foreground">Waiting…</span>
								{/if}
							</div>
						</div>
					{/each}
				</div>
			</CardContent>
		</Card>

		<!-- Success card -->
		{#if displayStatus === 'completed' && generation}
			<Card class="border-green-500/40 bg-green-50/50 dark:bg-green-950/20">
				<CardContent class="p-6">
					<div class="flex items-start gap-3 mb-4">
						<CheckCircle2 class="h-5 w-5 text-green-500 shrink-0 mt-0.5" />
						<div>
							<p class="font-semibold text-foreground">Code generated successfully</p>
							<p class="text-sm text-muted-foreground mt-0.5">
								The implementation has been generated and a pull request has been created.
							</p>
						</div>
					</div>

					{#if generation.prUrl}
						<div class="flex items-center gap-3 mb-4">
							<a
								href={generation.prUrl}
								target="_blank"
								rel="noopener noreferrer"
								class="inline-flex items-center gap-2 rounded-lg border border-border bg-background px-4 py-2.5 text-sm font-medium hover:bg-accent transition-colors"
							>
								<GitPullRequest class="h-4 w-4 text-primary" />
								View Pull Request
								{#if generation.prNumber}
									<span class="text-muted-foreground">#{generation.prNumber}</span>
								{/if}
								<ExternalLink class="h-3.5 w-3.5 text-muted-foreground" />
							</a>
						</div>
					{/if}

					{#if generation.files && generation.files.length > 0}
						<Separator class="my-4" />
						<div class="space-y-2">
							<p class="text-sm font-medium text-muted-foreground">
								{generation.files.length} file{generation.files.length === 1 ? '' : 's'} generated
							</p>
							<div class="space-y-1 max-h-60 overflow-y-auto">
								{#each generation.files as file (file.id)}
									<div
										class="flex items-center gap-2 rounded-md px-2 py-1.5 hover:bg-accent/50 transition-colors"
									>
										<FileCode2 class="h-3.5 w-3.5 text-muted-foreground shrink-0" />
										<span class="text-xs font-mono text-foreground flex-1 truncate">
											{file.path}
										</span>
										{#if file.isNew}
											<Badge variant="secondary" class="text-xs shrink-0">new</Badge>
										{/if}
										{#if file.language}
											<span class="text-xs text-muted-foreground shrink-0"
												>{file.language}</span
											>
										{/if}
									</div>
								{/each}
							</div>
						</div>
					{/if}
				</CardContent>
			</Card>
		{/if}

		<!-- Failure card -->
		{#if displayStatus === 'failed' && generation}
			<Card class="border-destructive/40 bg-destructive/5">
				<CardContent class="p-6">
					<div class="flex items-start gap-3 mb-4">
						<XCircle class="h-5 w-5 text-destructive shrink-0 mt-0.5" />
						<div class="flex-1">
							<p class="font-semibold text-foreground">Generation failed</p>
							{#if generation.errorMessage}
								<p class="text-sm text-muted-foreground mt-1">{generation.errorMessage}</p>
							{/if}
						</div>
					</div>
					<Button variant="outline" onclick={handleRetry} class="gap-2">
						<RotateCcw class="h-4 w-4" />
						Retry Generation
					</Button>
				</CardContent>
			</Card>
		{/if}

		<!-- Starting status -->
		{#if displayStatus === 'starting'}
			<div class="flex items-center gap-2 text-sm text-muted-foreground">
				<Loader2 class="h-4 w-4 animate-spin" />
				<span>Initializing pipeline…</span>
			</div>
		{/if}
	{/if}
</div>
