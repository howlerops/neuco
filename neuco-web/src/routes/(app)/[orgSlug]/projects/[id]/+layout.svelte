<script lang="ts">
	import { page } from '$app/stores';
	import { authStore } from '$lib/stores/auth.svelte';
	import { useProject } from '$lib/api/queries/projects';
	import { Skeleton } from '$lib/components/ui/skeleton';
	import { Alert, AlertDescription } from '$lib/components/ui/alert';
	import { Button } from '$lib/components/ui/button';
	import {
		Radio,
		Lightbulb,
		Code2,
		GitBranch,
		Puzzle,
		Brain,
		AlertCircle,
		ChevronRight
	} from 'lucide-svelte';
	import { cn } from '$lib/utils';
	import type { Framework } from '$lib/api/types';
	import { setContext } from 'svelte';

	let { children } = $props();

	const orgSlug = $derived($page.params.orgSlug ?? '');
	const projectId = $derived($page.params.id ?? '');

	// Use slug from URL — backend resolves both UUID and slug
	const projectQuery = $derived.by(() => useProject(orgSlug, projectId));

	// Expose project to child routes via context
	setContext('project', {
		get data() {
			return projectQuery.data;
		},
		get isLoading() {
			return projectQuery.isLoading;
		}
	});

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

	const tabs = $derived([
		{
			label: 'Signals',
			href: `/${orgSlug}/projects/${projectId}/signals`,
			icon: Radio,
			segment: 'signals'
		},
		{
			label: 'Candidates',
			href: `/${orgSlug}/projects/${projectId}/candidates`,
			icon: Lightbulb,
			segment: 'candidates'
		},
		{
			label: 'Generations',
			href: `/${orgSlug}/projects/${projectId}/generations`,
			icon: Code2,
			segment: 'generations'
		},
		{
			label: 'Pipelines',
			href: `/${orgSlug}/projects/${projectId}/pipelines`,
			icon: GitBranch,
			segment: 'pipelines'
		},
		{
			label: 'Memory',
			href: `/${orgSlug}/projects/${projectId}/memory`,
			icon: Brain,
			segment: 'memory'
		},
		{
			label: 'Integrations',
			href: `/${orgSlug}/projects/${projectId}/integrations`,
			icon: Puzzle,
			segment: 'integrations'
		}
	]);

	function isActiveTab(segment: string): boolean {
		return $page.url.pathname.includes(`/${segment}`);
	}
</script>

<div class="flex flex-col h-full">
	<!-- Sub-header -->
	<div class="border-b border-border bg-background">
		<!-- Breadcrumb + project name row -->
		<div class="flex items-center gap-2 px-6 pt-4 pb-0 text-sm text-muted-foreground">
			<a href="/{orgSlug}/projects" class="hover:text-foreground transition-colors">
				Projects
			</a>
			<ChevronRight class="h-3.5 w-3.5 shrink-0" />
			{#if projectQuery.isLoading}
				<Skeleton class="h-4 w-32" />
			{:else if projectQuery.data}
				<span class="text-foreground font-medium">{projectQuery.data.name}</span>
			{:else}
				<span class="text-foreground font-medium">Project</span>
			{/if}
		</div>

		<!-- Project title + framework badge -->
		<div class="flex items-center gap-3 px-6 pt-2 pb-3">
			{#if projectQuery.isLoading}
				<Skeleton class="h-7 w-48" />
				<Skeleton class="h-5 w-20 rounded-full" />
			{:else if projectQuery.isError}
				<Alert variant="destructive" class="max-w-lg">
					<AlertCircle class="h-4 w-4" />
					<AlertDescription class="flex items-center gap-3">
						Failed to load project.
						<Button
							variant="outline"
							size="sm"
							onclick={() => projectQuery.refetch()}
							class="h-7 text-xs"
						>
							Retry
						</Button>
					</AlertDescription>
				</Alert>
			{:else if projectQuery.data}
				<h1 class="text-xl font-semibold tracking-tight">{projectQuery.data.name}</h1>
				<span
					class="inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium {frameworkColors[
						projectQuery.data.framework
					]}"
				>
					{frameworkLabels[projectQuery.data.framework]}
				</span>
			{/if}
		</div>

		<!-- Tab navigation -->
		<nav class="flex items-center gap-0 px-6" aria-label="Project sections">
			{#each tabs as tab (tab.segment)}
				<a
					href={tab.href}
					class={cn(
						'flex items-center gap-1.5 px-3 py-2.5 text-sm font-medium border-b-2 transition-colors whitespace-nowrap',
						isActiveTab(tab.segment)
							? 'border-primary text-foreground'
							: 'border-transparent text-muted-foreground hover:text-foreground hover:border-border'
					)}
				>
					<tab.icon class="h-3.5 w-3.5 shrink-0" />
					{tab.label}
				</a>
			{/each}
		</nav>
	</div>

	<!-- Page content -->
	<div class="flex-1 overflow-y-auto">
		{@render children()}
	</div>
</div>
