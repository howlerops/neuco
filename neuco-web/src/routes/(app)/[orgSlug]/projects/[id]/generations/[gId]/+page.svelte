<script lang="ts">
	import { page } from '$app/stores';
	import { useGeneration } from '$lib/api/queries/generations';
	import { useCopilotNotes } from '$lib/api/queries/copilot';
	import { usePipelines } from '$lib/api/queries/pipelines';
	import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '$lib/components/ui/card';
	import { Badge } from '$lib/components/ui/badge';
	import { Button } from '$lib/components/ui/button';
	import { Skeleton } from '$lib/components/ui/skeleton';
	import { Alert, AlertTitle, AlertDescription } from '$lib/components/ui/alert';
	import { Separator } from '$lib/components/ui/separator';
	import {
		ArrowLeft,
		AlertCircle,
		RefreshCw,
		ExternalLink,
		GitPullRequest,
		ChevronDown,
		ChevronRight,
		FileCode2,
		FilePlus2,
		FileEdit,
		Lightbulb,
		ShieldAlert,
		Info,
		Zap,
		AlertTriangle,
		CheckCircle2,
		Clock,
		Loader2
	} from 'lucide-svelte';
	import type { GeneratedFile, CopilotNote } from '$lib/api/types';

	const projectId = $derived($page.params.id ?? '');
	const generationId = $derived($page.params.gId ?? '');
	const orgSlug = $derived($page.params.orgSlug ?? '');

	const generationQuery = $derived.by(() => useGeneration(projectId, generationId));
	const copilotQuery = $derived.by(() => useCopilotNotes(projectId));
	const pipelinesQuery = $derived.by(() => usePipelines(projectId, { page: 1, pageSize: 20 }));

	// Copilot notes filtered to this generation
	const generationNotes = $derived(
		(copilotQuery.data ?? []).filter(
			(n) => n.entityType === 'generation' && n.entityId === generationId && !n.dismissed
		)
	);

	// The pipeline run associated with this generation (by metadata linkage)
	const relatedPipeline = $derived(
		(pipelinesQuery.data?.data ?? []).find(
			(p) =>
				p.type === 'code_generation' &&
				(p.metadata?.generationId === generationId ||
					p.metadata?.generationId === generationId)
		)
	);

	// Track expanded files
	let expandedFiles = $state<Record<string, boolean>>({});

	function toggleFile(fileId: string) {
		expandedFiles = { ...expandedFiles, [fileId]: !expandedFiles[fileId] };
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

	function statusClass(status: string): string {
		switch (status) {
			case 'completed': return 'bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400 border-transparent';
			case 'running': return 'bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-400 border-transparent';
			case 'failed': return 'bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400 border-transparent';
			default: return 'bg-gray-100 text-gray-700 dark:bg-gray-800 dark:text-gray-400 border-transparent';
		}
	}

	function noteTypeIcon(type: CopilotNote['type']) {
		switch (type) {
			case 'suggestion': return Lightbulb;
			case 'warning': return AlertTriangle;
			case 'error': return AlertCircle;
			case 'security': return ShieldAlert;
			case 'performance': return Zap;
			default: return Info;
		}
	}

	function noteTypeClass(type: CopilotNote['type']): string {
		switch (type) {
			case 'error': return 'border-red-200 bg-red-50 dark:border-red-800 dark:bg-red-900/20';
			case 'warning': return 'border-amber-200 bg-amber-50 dark:border-amber-800 dark:bg-amber-900/20';
			case 'security': return 'border-orange-200 bg-orange-50 dark:border-orange-800 dark:bg-orange-900/20';
			case 'performance': return 'border-blue-200 bg-blue-50 dark:border-blue-800 dark:bg-blue-900/20';
			default: return 'border-violet-200 bg-violet-50 dark:border-violet-800 dark:bg-violet-900/20';
		}
	}

	function fileIcon(file: GeneratedFile) {
		return file.isNew ? FilePlus2 : FileEdit;
	}

	function pipelineTaskStatusClass(status: string): string {
		switch (status) {
			case 'completed': return 'text-green-600 dark:text-green-400';
			case 'failed': return 'text-red-600 dark:text-red-400';
			case 'running': return 'text-blue-600 dark:text-blue-400';
			default: return 'text-muted-foreground';
		}
	}
</script>

<svelte:head>
	<title>Generation Detail — Neuco</title>
</svelte:head>

<div class="p-6 space-y-6">
	<!-- Back nav -->
	<div>
		<Button
			variant="ghost"
			size="sm"
			href={`/${orgSlug}/projects/${projectId}/generations`}
			class="gap-1.5 -ml-2 text-muted-foreground hover:text-foreground"
		>
			<ArrowLeft class="h-4 w-4" />
			Back to Generations
		</Button>
	</div>

	{#if generationQuery.isLoading}
		<div class="space-y-6">
			<div class="space-y-2">
				<Skeleton class="h-7 w-64"></Skeleton>
				<Skeleton class="h-4 w-48"></Skeleton>
			</div>
			<div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
				{#each Array(4) as _, i (i)}
					<Card>
						<CardHeader class="pb-2">
							<Skeleton class="h-4 w-24"></Skeleton>
						</CardHeader>
						<CardContent>
							<Skeleton class="h-5 w-32"></Skeleton>
						</CardContent>
					</Card>
				{/each}
			</div>
		</div>
	{:else if generationQuery.isError}
		<Alert variant="destructive">
			<AlertCircle class="h-4 w-4" />
			<AlertTitle>Failed to load generation</AlertTitle>
			<AlertDescription class="flex items-center justify-between">
				<span>{generationQuery.error?.message ?? 'An unexpected error occurred.'}</span>
				<Button
					variant="outline"
					size="sm"
					onclick={() => generationQuery.refetch()}
					class="ml-4 shrink-0"
				>
					<RefreshCw class="mr-1.5 h-3 w-3" />
					Retry
				</Button>
			</AlertDescription>
		</Alert>
	{:else if generationQuery.data}
		{@const gen = generationQuery.data}

		<!-- Header -->
		<div class="flex items-start justify-between gap-4">
			<div>
				<h1 class="text-2xl font-bold tracking-tight">
					Generation
					<span class="font-mono text-lg text-muted-foreground">{gen.id.slice(0, 8)}</span>
				</h1>
				<p class="text-muted-foreground mt-1">
					Created {formatDate(gen.createdAt)}
				</p>
			</div>
			<span class="inline-flex items-center rounded-full border px-3 py-1 text-sm font-semibold {statusClass(gen.status)}">
				{gen.status.charAt(0).toUpperCase() + gen.status.slice(1)}
			</span>
		</div>

		<!-- Metadata cards -->
		<div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
			<Card>
				<CardHeader class="pb-2">
					<CardTitle class="text-xs font-medium text-muted-foreground uppercase tracking-wide">
						Status
					</CardTitle>
				</CardHeader>
				<CardContent>
					<span class="inline-flex items-center rounded-full border px-2.5 py-0.5 text-xs font-semibold {statusClass(gen.status)}">
						{gen.status.charAt(0).toUpperCase() + gen.status.slice(1)}
					</span>
				</CardContent>
			</Card>

			<Card>
				<CardHeader class="pb-2">
					<CardTitle class="text-xs font-medium text-muted-foreground uppercase tracking-wide">
						Pull Request
					</CardTitle>
				</CardHeader>
				<CardContent>
					{#if gen.prUrl}
						<a
							href={gen.prUrl}
							target="_blank"
							rel="noopener noreferrer"
							class="inline-flex items-center gap-1.5 text-sm text-blue-600 hover:text-blue-500 dark:text-blue-400 font-medium"
						>
							<GitPullRequest class="h-4 w-4" />
							PR #{gen.prNumber}
							<ExternalLink class="h-3.5 w-3.5 opacity-70" />
						</a>
					{:else}
						<span class="text-sm text-muted-foreground">Not created yet</span>
					{/if}
				</CardContent>
			</Card>

			<Card>
				<CardHeader class="pb-2">
					<CardTitle class="text-xs font-medium text-muted-foreground uppercase tracking-wide">
						Created
					</CardTitle>
				</CardHeader>
				<CardContent class="text-sm text-foreground">
					{formatDate(gen.createdAt)}
				</CardContent>
			</Card>

			<Card>
				<CardHeader class="pb-2">
					<CardTitle class="text-xs font-medium text-muted-foreground uppercase tracking-wide">
						Completed
					</CardTitle>
				</CardHeader>
				<CardContent class="text-sm text-foreground">
					{gen.status === 'completed' || gen.status === 'failed'
						? formatDate(gen.updatedAt)
						: '—'}
				</CardContent>
			</Card>
		</div>

		{#if gen.errorMessage}
			<Alert variant="destructive">
				<AlertCircle class="h-4 w-4" />
				<AlertTitle>Generation failed</AlertTitle>
				<AlertDescription>{gen.errorMessage}</AlertDescription>
			</Alert>
		{/if}

		<!-- Co-pilot notes -->
		{#if generationNotes.length > 0}
			<Card>
				<CardHeader>
					<CardTitle class="flex items-center gap-2">
						<Lightbulb class="h-4 w-4 text-violet-500" />
						Co-pilot Review Notes
					</CardTitle>
					<CardDescription>
						Automated review observations from the Neuco co-pilot
					</CardDescription>
				</CardHeader>
				<CardContent class="space-y-3">
					{#each generationNotes as note (note.id)}
						{@const NoteIcon = noteTypeIcon(note.type)}
						<div class="rounded-lg border p-4 {noteTypeClass(note.type)}">
							<div class="flex items-start gap-3">
								<NoteIcon class="h-4 w-4 mt-0.5 shrink-0" />
								<div class="flex-1 min-w-0">
									<p class="text-sm font-medium">{note.title}</p>
									<p class="text-sm mt-1 text-muted-foreground">{note.body}</p>
								</div>
								<Badge variant="outline" class="shrink-0 text-xs capitalize">
									{note.type}
								</Badge>
							</div>
						</div>
					{/each}
				</CardContent>
			</Card>
		{/if}

		<!-- Pipeline progress -->
		{#if relatedPipeline}
			<Card>
				<CardHeader>
					<CardTitle class="text-base">Pipeline Progress</CardTitle>
					<CardDescription>
						Tasks executed during this code generation run
					</CardDescription>
				</CardHeader>
				<CardContent>
					{#if relatedPipeline.tasks && relatedPipeline.tasks.length > 0}
						<div class="space-y-0">
							{#each relatedPipeline.tasks as task, idx (task.id)}
								<div class="flex gap-4">
									<!-- Timeline indicator -->
									<div class="flex flex-col items-center">
										<div class="flex h-8 w-8 items-center justify-center rounded-full border-2 shrink-0
											{task.status === 'completed'
												? 'border-green-500 bg-green-50 dark:bg-green-900/20'
												: task.status === 'failed'
												? 'border-red-500 bg-red-50 dark:bg-red-900/20'
												: task.status === 'running'
												? 'border-blue-500 bg-blue-50 dark:bg-blue-900/20'
												: 'border-border bg-muted/30'}">
											{#if task.status === 'completed'}
												<CheckCircle2 class="h-4 w-4 text-green-600 dark:text-green-400" />
											{:else if task.status === 'failed'}
												<AlertCircle class="h-4 w-4 text-red-600 dark:text-red-400" />
											{:else if task.status === 'running'}
												<Loader2 class="h-4 w-4 text-blue-600 dark:text-blue-400 animate-spin" />
											{:else}
												<Clock class="h-4 w-4 text-muted-foreground" />
											{/if}
										</div>
										{#if idx < relatedPipeline.tasks.length - 1}
											<div class="w-px flex-1 my-1
												{task.status === 'completed' ? 'bg-green-300 dark:bg-green-700' : 'bg-border'}">
											</div>
										{/if}
									</div>

									<!-- Task info -->
									<div class="flex-1 pb-4">
										<div class="flex items-center justify-between gap-2 min-h-8">
											<p class="text-sm font-medium capitalize">
												{task.name.replace(/_/g, ' ')}
											</p>
											{#if task.startedAt && task.completedAt}
												<span class="text-xs text-muted-foreground shrink-0">
													{formatDuration(task.startedAt, task.completedAt)}
												</span>
											{/if}
										</div>
										{#if task.errorMessage}
											<Alert variant="destructive" class="mt-2 py-2">
												<AlertCircle class="h-3.5 w-3.5" />
												<AlertDescription class="text-xs">
													{task.errorMessage}
												</AlertDescription>
											</Alert>
										{/if}
									</div>
								</div>
							{/each}
						</div>
					{:else}
						<p class="text-sm text-muted-foreground">No tasks recorded for this pipeline run.</p>
					{/if}
				</CardContent>
			</Card>
		{/if}

		<!-- Generated files -->
		<Card>
			<CardHeader>
				<CardTitle class="flex items-center justify-between">
					<span class="flex items-center gap-2">
						<FileCode2 class="h-4 w-4" />
						Generated Files
					</span>
					<Badge variant="secondary">{gen.files?.length ?? 0} files</Badge>
				</CardTitle>
				<CardDescription>
					Files produced by this generation run
				</CardDescription>
			</CardHeader>
			<CardContent class="p-0">
				{#if !gen.files || gen.files.length === 0}
					<div class="flex flex-col items-center justify-center py-10 text-center px-6">
						<FileCode2 class="h-8 w-8 text-muted-foreground mb-3" />
						<p class="text-sm font-medium">No files generated yet</p>
						<p class="text-xs text-muted-foreground mt-1">
							Files will appear here once the generation completes.
						</p>
					</div>
				{:else}
					<div class="divide-y divide-border">
						{#each gen.files as file (file.id)}
							{@const FileIcon = fileIcon(file)}
							{@const isExpanded = expandedFiles[file.id] ?? false}
							<div>
								<!-- File header row -->
								<button
									type="button"
									class="w-full flex items-center gap-3 px-4 py-3 hover:bg-muted/40 transition-colors text-left"
									onclick={() => toggleFile(file.id)}
									aria-expanded={isExpanded}
								>
									<div class="shrink-0">
										{#if isExpanded}
											<ChevronDown class="h-4 w-4 text-muted-foreground" />
										{:else}
											<ChevronRight class="h-4 w-4 text-muted-foreground" />
										{/if}
									</div>
									<FileIcon class="h-4 w-4 shrink-0 {file.isNew ? 'text-green-500' : 'text-blue-500'}" />
									<span class="flex-1 font-mono text-sm truncate">{file.path}</span>
									<div class="flex items-center gap-2 shrink-0">
										{#if file.isNew}
											<span class="inline-flex items-center rounded-full border px-2 py-0.5 text-xs font-medium bg-green-100 text-green-700 border-transparent dark:bg-green-900/30 dark:text-green-400">
												New
											</span>
										{:else}
											<span class="inline-flex items-center rounded-full border px-2 py-0.5 text-xs font-medium bg-blue-100 text-blue-700 border-transparent dark:bg-blue-900/30 dark:text-blue-400">
												Modified
											</span>
										{/if}
										{#if file.language}
											<span class="text-xs text-muted-foreground font-mono">{file.language}</span>
										{/if}
									</div>
								</button>

								<!-- File content -->
								{#if isExpanded}
									<div class="border-t border-border bg-muted/20">
										<pre class="overflow-x-auto p-4 text-xs leading-relaxed font-mono text-foreground whitespace-pre"><code>{file.content}</code></pre>
									</div>
								{/if}
							</div>
						{/each}
					</div>
				{/if}
			</CardContent>
		</Card>
	{/if}
</div>
