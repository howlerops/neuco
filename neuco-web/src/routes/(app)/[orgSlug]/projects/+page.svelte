<script lang="ts">
	import { page } from '$app/stores';
	import { authStore } from '$lib/stores/auth.svelte';
	import { useProjects } from '$lib/api/queries/projects';
	import { Card, CardContent, CardHeader, CardTitle, CardFooter } from '$lib/components/ui/card';
	import { Badge } from '$lib/components/ui/badge';
	import { Button } from '$lib/components/ui/button';
	import { Skeleton } from '$lib/components/ui/skeleton';
	import CreateProjectDialog from '$lib/components/projects/CreateProjectDialog.svelte';
	import {
		FolderKanban,
		Plus,
		Radio,
		Clock,
		Github,
		ArrowRight
	} from 'lucide-svelte';
	import type { Framework, Styling } from '$lib/api/types';

	const orgSlug = $derived($page.params.orgSlug ?? '');

	// Use slug directly — the backend's ResolveOrg middleware accepts both UUID and slug
	const projectsQuery = $derived.by(() => useProjects(orgSlug));

	let dialogOpen = $state(false);

	const frameworkLabels: Record<Framework, string> = {
		react: 'React',
		nextjs: 'Next.js',
		vue: 'Vue',
		nuxt: 'Nuxt',
		svelte: 'Svelte',
		sveltekit: 'SvelteKit',
		angular: 'Angular',
		other: 'Other'
	};

	const frameworkColors: Record<Framework, string> = {
		react: 'bg-cyan-100 text-cyan-800 dark:bg-cyan-900/30 dark:text-cyan-400',
		nextjs: 'bg-neutral-100 text-neutral-800 dark:bg-neutral-900/30 dark:text-neutral-300',
		vue: 'bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400',
		nuxt: 'bg-emerald-100 text-emerald-800 dark:bg-emerald-900/30 dark:text-emerald-400',
		svelte: 'bg-orange-100 text-orange-800 dark:bg-orange-900/30 dark:text-orange-400',
		sveltekit: 'bg-orange-100 text-orange-800 dark:bg-orange-900/30 dark:text-orange-400',
		angular: 'bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400',
		other: 'bg-gray-100 text-gray-800 dark:bg-gray-900/30 dark:text-gray-400'
	};

	const stylingLabels: Record<Styling, string> = {
		tailwind: 'Tailwind',
		css_modules: 'CSS Modules',
		styled_components: 'Styled',
		sass: 'Sass',
		plain_css: 'CSS',
		other: 'Other'
	};

	function formatRelativeTime(dateStr: string): string {
		if (!dateStr) return 'Never';
		const date = new Date(dateStr);
		const now = new Date();
		const diffMs = now.getTime() - date.getTime();
		const diffMins = Math.floor(diffMs / 60000);
		const diffHours = Math.floor(diffMins / 60);
		const diffDays = Math.floor(diffHours / 24);

		if (diffMins < 1) return 'Just now';
		if (diffMins < 60) return `${diffMins}m ago`;
		if (diffHours < 24) return `${diffHours}h ago`;
		if (diffDays < 30) return `${diffDays}d ago`;
		return date.toLocaleDateString();
	}
</script>

<svelte:head>
	<title>Projects — Neuco</title>
</svelte:head>

<div class="p-6 space-y-6">
	<!-- Header -->
	<div class="flex items-center justify-between">
		<div>
			<h1 class="text-2xl font-bold tracking-tight">Projects</h1>
			<p class="text-muted-foreground mt-1">
				{projectsQuery.data?.length ?? 0} project{(projectsQuery.data?.length ?? 0) === 1
					? ''
					: 's'} in {authStore.currentOrg?.name ?? 'your organization'}
			</p>
		</div>
		<Button onclick={() => (dialogOpen = true)} class="gap-2">
			<Plus class="h-4 w-4" />
			New Project
		</Button>
	</div>

	<!-- Project grid -->
	{#if projectsQuery.isLoading}
		<div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
			{#each Array(6) as _, i (i)}
				<Card>
					<CardHeader>
						<Skeleton class="h-5 w-40"></Skeleton>
						<Skeleton class="h-4 w-28 mt-1"></Skeleton>
					</CardHeader>
					<CardContent>
						<div class="flex gap-2">
							<Skeleton class="h-5 w-16 rounded-full"></Skeleton>
							<Skeleton class="h-5 w-16 rounded-full"></Skeleton>
						</div>
					</CardContent>
					<CardFooter>
						<Skeleton class="h-4 w-32"></Skeleton>
					</CardFooter>
				</Card>
			{/each}
		</div>
	{:else if projectsQuery.isError}
		<div
			class="flex flex-col items-center justify-center rounded-xl border border-destructive/30 bg-destructive/5 p-12 text-center"
		>
			<p class="text-sm font-medium text-destructive">Failed to load projects</p>
			<p class="text-xs text-muted-foreground mt-1">{projectsQuery.error?.message}</p>
			<Button
				variant="outline"
				size="sm"
				class="mt-4"
				onclick={() => projectsQuery.refetch()}
			>
				Retry
			</Button>
		</div>
	{:else if !projectsQuery.data || projectsQuery.data.length === 0}
		<div
			class="flex flex-col items-center justify-center rounded-xl border-2 border-dashed border-border py-20 text-center"
		>
			<FolderKanban class="h-12 w-12 text-muted-foreground mb-4" />
			<h3 class="text-lg font-semibold">No projects yet</h3>
			<p class="text-sm text-muted-foreground mt-2 max-w-sm">
				Create your first project to start ingesting signals and generating AI-powered feature
				candidates.
			</p>
			<Button onclick={() => (dialogOpen = true)} class="mt-6 gap-2">
				<Plus class="h-4 w-4" />
				Create your first project
			</Button>
		</div>
	{:else}
		<div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
			{#each projectsQuery.data as project (project.id)}
				<Card class="group flex flex-col hover:shadow-md transition-shadow">
					<CardHeader class="pb-3">
						<div class="flex items-start justify-between gap-2">
							<CardTitle class="text-base leading-tight truncate">
								{project.name}
							</CardTitle>
							<span
								class="inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium shrink-0 {frameworkColors[
									project.framework
								]}"
							>
								{frameworkLabels[project.framework]}
							</span>
						</div>
						{#if project.githubRepo}
							<div class="flex items-center gap-1.5 text-xs text-muted-foreground mt-1">
								<Github class="h-3 w-3 shrink-0" />
								<span class="truncate">{project.githubRepo}</span>
							</div>
						{/if}
					</CardHeader>

					<CardContent class="flex-1 pb-3">
						<div class="flex flex-wrap gap-1.5">
							<Badge variant="secondary" class="text-xs">
								{stylingLabels[project.styling]}
							</Badge>
							<div class="flex items-center gap-1 text-xs text-muted-foreground">
								<Radio class="h-3 w-3" />
								<span>{project.signalCount} signals</span>
							</div>
						</div>
					</CardContent>

					<CardFooter class="flex items-center justify-between pt-0">
						<div class="flex items-center gap-1 text-xs text-muted-foreground">
							<Clock class="h-3 w-3" />
							<span>{formatRelativeTime(project.lastActivityAt)}</span>
						</div>
						<Button
							variant="ghost"
							size="sm"
							href={`/${orgSlug}/projects/${project.id}`}
							class="h-7 gap-1.5 text-xs opacity-0 group-hover:opacity-100 transition-opacity"
						>
							Open
							<ArrowRight class="h-3 w-3" />
						</Button>
					</CardFooter>
				</Card>
			{/each}
		</div>
	{/if}
</div>

{#if orgSlug}
	<CreateProjectDialog
		orgId={orgSlug}
		bind:open={dialogOpen}
		onCreated={() => projectsQuery.refetch()}
	/>
{/if}
