<script lang="ts">
	import { page } from '$app/stores';
	import { useProjectContexts, useCreateProjectContext, useDeleteProjectContext } from '$lib/api/queries/contexts';
	import { Button } from '$lib/components/ui/button';
	import { Input } from '$lib/components/ui/input';
	import { Skeleton } from '$lib/components/ui/skeleton';
	import { Alert, AlertDescription } from '$lib/components/ui/alert';
	import {
		Dialog,
		DialogContent,
		DialogHeader,
		DialogTitle,
		DialogDescription,
		DialogFooter
	} from '$lib/components/ui/dialog';
	import {
		Select,
		SelectTrigger,
		SelectContent,
		SelectItem
	} from '$lib/components/ui/select';
	import { toast } from '$lib/components/ui/sonner';
	import {
		Brain,
		Plus,
		Trash2,
		AlertCircle,
		Lightbulb,
		TrendingUp,
		AlertTriangle,
		Target,
		Layers
	} from 'lucide-svelte';
	import { cn } from '$lib/utils';
	import type { ContextCategory, CreateProjectContextPayload } from '$lib/api/types-compat';

	const projectId = $derived($page.params.id ?? '');

	let filterCategory = $state<ContextCategory | ''>('');
	let currentPage = $state(1);
	const pageSize = 20;

	const contextsQuery = $derived.by(() =>
		useProjectContexts(projectId, {
			page: currentPage,
			pageSize,
			category: filterCategory || undefined
		})
	);

	// ── Create dialog ─────────────────────────────────────────────────────────
	let showCreateDialog = $state(false);
	let newTitle = $state('');
	let newContent = $state('');
	let newCategory = $state<ContextCategory>('insight');

	const createMutation = $derived.by(() => useCreateProjectContext(projectId));

	async function handleCreate() {
		if (!newTitle.trim() || !newContent.trim()) return;

		const payload: CreateProjectContextPayload = {
			category: newCategory,
			title: newTitle.trim(),
			content: newContent.trim()
		};

		try {
			await createMutation.mutateAsync(payload);
			toast.success('Context added');
			showCreateDialog = false;
			newTitle = '';
			newContent = '';
			newCategory = 'insight';
		} catch {
			toast.error('Failed to create context');
		}
	}

	// ── Delete ────────────────────────────────────────────────────────────────
	let deletingId = $state<string | null>(null);

	async function handleDelete(contextId: string) {
		deletingId = contextId;
		try {
			const deleteMutation = useDeleteProjectContext(projectId, contextId);
			await deleteMutation.mutateAsync();
			toast.success('Context removed');
		} catch {
			toast.error('Failed to delete context');
		} finally {
			deletingId = null;
		}
	}

	// ── Category helpers ──────────────────────────────────────────────────────
	const categoryConfig: Record<ContextCategory, { label: string; color: string; icon: typeof Lightbulb }> = {
		insight: { label: 'Insight', color: 'bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-400', icon: Lightbulb },
		theme: { label: 'Theme', color: 'bg-purple-100 text-purple-800 dark:bg-purple-900/30 dark:text-purple-400', icon: Layers },
		decision: { label: 'Decision', color: 'bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400', icon: Target },
		risk: { label: 'Risk', color: 'bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400', icon: AlertTriangle },
		opportunity: { label: 'Opportunity', color: 'bg-amber-100 text-amber-800 dark:bg-amber-900/30 dark:text-amber-400', icon: TrendingUp }
	};

	function formatDate(dateStr: string): string {
		return new Date(dateStr).toLocaleDateString('en-US', {
			month: 'short',
			day: 'numeric',
			year: 'numeric'
		});
	}
</script>

<div class="p-6 space-y-6 max-w-5xl mx-auto">
	<!-- Header -->
	<div class="flex items-center justify-between">
		<div>
			<h2 class="text-lg font-semibold tracking-tight flex items-center gap-2">
				<Brain class="h-5 w-5 text-muted-foreground" />
				Project Memory
			</h2>
			<p class="text-sm text-muted-foreground mt-1">
				Accumulated insights that persist across synthesis runs. The AI builds on this context over time.
			</p>
		</div>
		<Button onclick={() => (showCreateDialog = true)} size="sm">
			<Plus class="h-4 w-4 mr-1.5" />
			Add Context
		</Button>
	</div>

	<!-- Filters -->
	<div class="flex items-center gap-3">
		<Select
			value={filterCategory || undefined}
			onValueChange={(v) => {
				filterCategory = (v as ContextCategory) ?? '';
				currentPage = 1;
			}}
		>
			<SelectTrigger class="w-[160px]">
				{filterCategory ? categoryConfig[filterCategory]?.label : 'All categories'}
			</SelectTrigger>
			<SelectContent>
				<SelectItem value="">All categories</SelectItem>
				{#each Object.entries(categoryConfig) as [key, cfg]}
					<SelectItem value={key}>{cfg.label}</SelectItem>
				{/each}
			</SelectContent>
		</Select>
		{#if filterCategory}
			<Button
				variant="ghost"
				size="sm"
				onclick={() => {
					filterCategory = '';
					currentPage = 1;
				}}
			>
				Clear
			</Button>
		{/if}
	</div>

	<!-- Content -->
	{#if contextsQuery.isLoading}
		<div class="space-y-3">
			{#each Array(3) as _}
				<Skeleton class="h-24 w-full rounded-lg" />
			{/each}
		</div>
	{:else if contextsQuery.isError}
		<Alert variant="destructive">
			<AlertCircle class="h-4 w-4" />
			<AlertDescription>
				Failed to load project context.
				<Button variant="outline" size="sm" onclick={() => contextsQuery.refetch()} class="ml-2 h-7 text-xs">
					Retry
				</Button>
			</AlertDescription>
		</Alert>
	{:else if contextsQuery.data && contextsQuery.data.data.length === 0}
		<div class="text-center py-16 text-muted-foreground">
			<Brain class="h-12 w-12 mx-auto mb-4 opacity-30" />
			<p class="text-sm font-medium">No context yet</p>
			<p class="text-xs mt-1">
				Run a synthesis to generate insights automatically, or add context manually.
			</p>
		</div>
	{:else if contextsQuery.data}
		<div class="space-y-3">
			{#each contextsQuery.data.data as ctx (ctx.id)}
				{@const cfg = categoryConfig[ctx.category] ?? categoryConfig.insight}
				<div class="border border-border rounded-lg p-4 hover:bg-muted/30 transition-colors group">
					<div class="flex items-start justify-between gap-3">
						<div class="flex-1 min-w-0">
							<div class="flex items-center gap-2 mb-1.5">
								<span
									class={cn(
										'inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-xs font-medium',
										cfg.color
									)}
								>
									{#if cfg.icon}<cfg.icon class="h-3 w-3" />{/if}
									{cfg.label}
								</span>
								<span class="text-xs text-muted-foreground">
									{formatDate(ctx.created_at)}
								</span>
								{#if ctx.source_run_id}
									<span class="text-xs text-muted-foreground opacity-60">
										via synthesis
									</span>
								{/if}
							</div>
							<h3 class="font-medium text-sm">{ctx.title}</h3>
							<p class="text-sm text-muted-foreground mt-1 whitespace-pre-wrap">{ctx.content}</p>
						</div>
						<Button
							variant="ghost"
							size="icon"
							class="h-8 w-8 opacity-0 group-hover:opacity-100 transition-opacity text-muted-foreground hover:text-destructive"
							onclick={() => handleDelete(ctx.id)}
							disabled={deletingId === ctx.id}
						>
							<Trash2 class="h-4 w-4" />
						</Button>
					</div>
				</div>
			{/each}
		</div>

		<!-- Pagination -->
		{#if contextsQuery.data.totalPages > 1}
			<div class="flex items-center justify-between pt-2">
				<span class="text-sm text-muted-foreground">
					{contextsQuery.data.total} total
				</span>
				<div class="flex items-center gap-2">
					<Button
						variant="outline"
						size="sm"
						disabled={currentPage <= 1}
						onclick={() => (currentPage = Math.max(1, currentPage - 1))}
					>
						Previous
					</Button>
					<span class="text-sm text-muted-foreground">
						Page {currentPage} of {contextsQuery.data.totalPages}
					</span>
					<Button
						variant="outline"
						size="sm"
						disabled={currentPage >= contextsQuery.data.totalPages}
						onclick={() => (currentPage = currentPage + 1)}
					>
						Next
					</Button>
				</div>
			</div>
		{/if}
	{/if}
</div>

<!-- Create Context Dialog -->
<Dialog bind:open={showCreateDialog}>
	<DialogContent class="sm:max-w-lg">
		<DialogHeader>
			<DialogTitle>Add Project Context</DialogTitle>
			<DialogDescription>
				Add a manual insight, decision, or observation to the project's memory.
			</DialogDescription>
		</DialogHeader>
		<div class="space-y-4 py-2">
			<div>
				<label for="ctx-category" class="text-sm font-medium block mb-1.5">Category</label>
				<Select
					value={newCategory}
					onValueChange={(v) => {
						if (v) newCategory = v as ContextCategory;
					}}
				>
					<SelectTrigger>
						{categoryConfig[newCategory]?.label ?? 'Select category'}
					</SelectTrigger>
					<SelectContent>
						{#each Object.entries(categoryConfig) as [key, cfg]}
							<SelectItem value={key}>{cfg.label}</SelectItem>
						{/each}
					</SelectContent>
				</Select>
			</div>
			<div>
				<label for="ctx-title" class="text-sm font-medium block mb-1.5">Title</label>
				<Input
					id="ctx-title"
					bind:value={newTitle}
					placeholder="e.g., Users want offline support"
				/>
			</div>
			<div>
				<label for="ctx-content" class="text-sm font-medium block mb-1.5">Content</label>
				<textarea
					id="ctx-content"
					bind:value={newContent}
					placeholder="Describe the insight, decision, or observation..."
					rows={4}
					class="flex w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"
				></textarea>
			</div>
		</div>
		<DialogFooter>
			<Button variant="outline" onclick={() => (showCreateDialog = false)}>Cancel</Button>
			<Button
				onclick={handleCreate}
				disabled={!newTitle.trim() || !newContent.trim() || createMutation.isPending}
			>
				{createMutation.isPending ? 'Adding...' : 'Add Context'}
			</Button>
		</DialogFooter>
	</DialogContent>
</Dialog>
