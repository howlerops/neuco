<script lang="ts">
	import { page } from '$app/stores';
	import { authStore } from '$lib/stores/auth.svelte';
	import { useProjectStats, usePipelines } from '$lib/api/queries/pipelines';
	import { useProjects } from '$lib/api/queries/projects';
	import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '$lib/components/ui/card';
	import { Badge } from '$lib/components/ui/badge';
	import { Skeleton } from '$lib/components/ui/skeleton';
	import { Button } from '$lib/components/ui/button';
	import { Alert, AlertTitle, AlertDescription } from '$lib/components/ui/alert';
	import {
		Radio,
		Lightbulb,
		GitPullRequest,
		Activity,
		AlertTriangle,
		RefreshCw,
		CheckCircle2,
		XCircle,
		Clock,
		Loader2,
		FolderKanban
	} from 'lucide-svelte';

	const orgSlug = $derived($page.params.orgSlug);

	// Reactive values needed for query hooks
	let orgId = $state(authStore.currentOrg?.id ?? '');
	$effect(() => {
		orgId = authStore.currentOrg?.id ?? '';
	});

	// Get projects to find first project ID for stats
	const projectsQuery = $derived.by(() => useProjects(orgId));
	const firstProjectId = $derived(projectsQuery.data?.[0]?.id ?? '');

	const statsQuery = $derived.by(() => useProjectStats(firstProjectId));
	const pipelinesQuery = $derived.by(() => usePipelines(firstProjectId, { page: 1, pageSize: 10 }));

	const failedPipelines = $derived(
		(pipelinesQuery.data?.data ?? []).filter((p) => p.status === 'failed')
	);

	const recentPipelines = $derived(pipelinesQuery.data?.data ?? []);

	function formatPipelineType(type: string): string {
		return type
			.split('_')
			.map((w) => w.charAt(0).toUpperCase() + w.slice(1))
			.join(' ');
	}

	function formatRelativeTime(dateStr: string): string {
		if (!dateStr) return 'Unknown';
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

	function successRate(rate: number): string {
		return `${Math.round(rate * 100)}%`;
	}

	const statCards = $derived([
		{
			label: 'Signals Ingested',
			value: statsQuery.data?.signalsIngested ?? 0,
			icon: Radio,
			description: 'Total signals processed'
		},
		{
			label: 'Candidates Found',
			value: statsQuery.data?.candidatesFound ?? 0,
			icon: Lightbulb,
			description: 'Feature candidates extracted'
		},
		{
			label: 'PRs Created',
			value: statsQuery.data?.prsCreated ?? 0,
			icon: GitPullRequest,
			description: 'Pull requests opened'
		},
		{
			label: 'Pipeline Success Rate',
			value: statsQuery.data ? successRate(statsQuery.data.pipelineSuccessRate) : '—',
			icon: Activity,
			description: `${statsQuery.data?.totalPipelines ?? 0} total runs`
		}
	]);
</script>

<svelte:head>
	<title>Dashboard — Neuco</title>
</svelte:head>

<div class="p-6 space-y-6">
	<!-- Page header -->
	<div>
		<h1 class="text-2xl font-bold tracking-tight">Dashboard</h1>
		<p class="text-muted-foreground mt-1">
			{#if authStore.currentOrg}
				{authStore.currentOrg.name} overview
			{:else}
				Your project overview
			{/if}
		</p>
	</div>

	<!-- Stats cards -->
	<div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
		{#each statCards as stat, i (i)}
			<Card>
				<CardHeader class="flex flex-row items-center justify-between space-y-0 pb-2">
					<CardTitle class="text-sm font-medium">{stat.label}</CardTitle>
					<stat.icon class="h-4 w-4 text-muted-foreground" />
				</CardHeader>
				<CardContent>
					{#if statsQuery.isLoading}
						<Skeleton class="h-8 w-24"></Skeleton>
						<Skeleton class="mt-2 h-3 w-32"></Skeleton>
					{:else}
						<div class="text-2xl font-bold">{stat.value}</div>
						<p class="text-xs text-muted-foreground mt-1">{stat.description}</p>
					{/if}
				</CardContent>
			</Card>
		{/each}
	</div>

	<div class="grid gap-6 lg:grid-cols-3">
		<!-- Recent pipeline activity -->
		<Card class="lg:col-span-2">
			<CardHeader>
				<CardTitle>Recent Pipeline Activity</CardTitle>
				<CardDescription>Latest pipeline runs across all projects</CardDescription>
			</CardHeader>
			<CardContent>
				{#if pipelinesQuery.isLoading}
					<div class="space-y-3">
						{#each Array(5) as _, i (i)}
							<div class="flex items-center gap-3">
								<Skeleton class="h-8 w-8 rounded-full"></Skeleton>
								<div class="flex-1 space-y-1">
									<Skeleton class="h-4 w-48"></Skeleton>
									<Skeleton class="h-3 w-32"></Skeleton>
								</div>
								<Skeleton class="h-5 w-20"></Skeleton>
							</div>
						{/each}
					</div>
				{:else if recentPipelines.length === 0}
					<div class="flex flex-col items-center justify-center py-10 text-center">
						<Activity class="h-10 w-10 text-muted-foreground mb-3" />
						<p class="text-sm font-medium">No pipeline runs yet</p>
						<p class="text-xs text-muted-foreground mt-1">
							Pipelines will appear here once your project is set up.
						</p>
					</div>
				{:else}
					<div class="space-y-3">
						{#each recentPipelines as pipeline (pipeline.id)}
							<div class="flex items-center gap-3 rounded-lg border border-border p-3">
								<div class="shrink-0">
									{#if pipeline.status === 'completed'}
										<CheckCircle2 class="h-5 w-5 text-green-500" />
									{:else if pipeline.status === 'failed'}
										<XCircle class="h-5 w-5 text-destructive" />
									{:else if pipeline.status === 'running'}
										<Loader2 class="h-5 w-5 text-blue-500 animate-spin" />
									{:else}
										<Clock class="h-5 w-5 text-muted-foreground" />
									{/if}
								</div>

								<div class="flex-1 min-w-0">
									<p class="text-sm font-medium truncate">
										{formatPipelineType(pipeline.type)}
									</p>
									<p class="text-xs text-muted-foreground">
										{formatRelativeTime(pipeline.createdAt)}
										{#if pipeline.errorMessage}
											&middot; {pipeline.errorMessage.slice(0, 50)}...
										{/if}
									</p>
								</div>

								<Badge
									variant={pipeline.status === 'completed'
										? 'default'
										: pipeline.status === 'failed'
											? 'destructive'
											: 'secondary'}
									class="shrink-0"
								>
									{pipeline.status}
								</Badge>
							</div>
						{/each}
					</div>
				{/if}
			</CardContent>
		</Card>

		<!-- Right column -->
		<div class="space-y-4">
			<!-- Needs Attention -->
			<Card>
				<CardHeader>
					<CardTitle class="flex items-center gap-2">
						<AlertTriangle class="h-4 w-4 text-amber-500" />
						Needs Attention
					</CardTitle>
					<CardDescription>Failed pipelines requiring review</CardDescription>
				</CardHeader>
				<CardContent>
					{#if pipelinesQuery.isLoading}
						<div class="space-y-2">
							{#each Array(3) as _, i (i)}
								<Skeleton class="h-16 w-full rounded-lg"></Skeleton>
							{/each}
						</div>
					{:else if failedPipelines.length === 0}
						<div class="flex flex-col items-center justify-center py-6 text-center">
							<CheckCircle2 class="h-8 w-8 text-green-500 mb-2" />
							<p class="text-sm font-medium">All clear</p>
							<p class="text-xs text-muted-foreground mt-1">
								No failed pipelines to review.
							</p>
						</div>
					{:else}
						<div class="space-y-2">
							{#each failedPipelines as pipeline (pipeline.id)}
								<Alert variant="destructive" class="p-3">
									<AlertTitle class="text-xs font-medium">
										{formatPipelineType(pipeline.type)}
									</AlertTitle>
									<AlertDescription class="text-xs mt-1">
										{pipeline.errorMessage
											? pipeline.errorMessage.slice(0, 80)
											: 'Pipeline failed without error message.'}
									</AlertDescription>
									<Button
										variant="outline"
										size="sm"
										class="mt-2 h-7 text-xs"
										href={`/${orgSlug}/pipelines/${pipeline.id}`}
									>
										<RefreshCw class="mr-1.5 h-3 w-3" />
										View &amp; Retry
									</Button>
								</Alert>
							{/each}
						</div>
					{/if}
				</CardContent>
			</Card>

			<!-- Quick links -->
			<Card>
				<CardHeader>
					<CardTitle class="text-sm">Quick Actions</CardTitle>
				</CardHeader>
				<CardContent class="flex flex-col gap-2">
					<Button
						variant="outline"
						class="w-full justify-start gap-2"
						href={`/${orgSlug}/projects`}
					>
						<FolderKanban class="h-4 w-4" />
						View Projects
					</Button>
					<Button
						variant="outline"
						class="w-full justify-start gap-2"
						href={`/${orgSlug}/signals`}
					>
						<Radio class="h-4 w-4" />
						Manage Signals
					</Button>
					<Button
						variant="outline"
						class="w-full justify-start gap-2"
						href={`/${orgSlug}/candidates`}
					>
						<Lightbulb class="h-4 w-4" />
						Review Candidates
					</Button>
				</CardContent>
			</Card>
		</div>
	</div>
</div>
