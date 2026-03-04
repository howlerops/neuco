<script lang="ts">
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import {
		useCandidates,
		useRefreshCandidates,
		useUpdateCandidateStatus
	} from '$lib/api/queries/candidates';
	import { useCopilotNotes, useDismissNote } from '$lib/api/queries/copilot';
	import { Card, CardContent } from '$lib/components/ui/card';
	import { Badge } from '$lib/components/ui/badge';
	import { Button } from '$lib/components/ui/button';
	import { Skeleton } from '$lib/components/ui/skeleton';
	import { Alert, AlertDescription, AlertTitle } from '$lib/components/ui/alert';
	import {
		DropdownMenu,
		DropdownMenuTrigger,
		DropdownMenuContent,
		DropdownMenuItem,
		DropdownMenuSeparator,
		DropdownMenuLabel
	} from '$lib/components/ui/dropdown-menu';
	import { toast } from '$lib/components/ui/sonner';
	import {
		RefreshCw,
		Lightbulb,
		AlertCircle,
		AlertTriangle,
		Sparkles,
		ChevronDown,
		X,
		ArrowRight,
		TrendingUp
	} from 'lucide-svelte';
	import { cn } from '$lib/utils';
	import type { CandidateStatus, CopilotNoteType } from '$lib/api/types';

	const projectId = $derived($page.params.id ?? '');
	const orgSlug = $derived($page.params.orgSlug ?? '');

	// ── Candidates ────────────────────────────────────────────────────────────
	const candidatesQuery = $derived(useCandidates(projectId));
	const refreshMutation = $derived(useRefreshCandidates(projectId));

	// ── Copilot notes ─────────────────────────────────────────────────────────
	const copilotQuery = $derived(useCopilotNotes(projectId));
	const dismissMutation = $derived(useDismissNote(projectId));

	const copilotNotes = $derived(
		(copilotQuery.data ?? []).filter((n) => n.entityType === 'synthesis' && !n.dismissed)
	);

	function handleRefresh() {
		refreshMutation.mutate(undefined, {
			onSuccess: () => {
				toast.success('Candidate refresh started — results will appear shortly');
			},
			onError: (err) => {
				toast.error(err.message ?? 'Failed to refresh candidates');
			}
		});
	}

	function handleDismissNote(noteId: string) {
		dismissMutation.mutate(noteId, {
			onError: (err) => {
				toast.error(err.message ?? 'Failed to dismiss note');
			}
		});
	}

	const sortedCandidates = $derived(
		[...(candidatesQuery.data?.data ?? [])].sort((a, b) => b.priority - a.priority)
	);

	// ── Status helpers ─────────────────────────────────────────────────────────
	const statusLabels: Record<CandidateStatus, string> = {
		pending: 'Pending',
		approved: 'Approved',
		rejected: 'Rejected',
		deferred: 'Deferred'
	};

	const statusColors: Record<CandidateStatus, string> = {
		pending: 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-300',
		approved: 'bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-300',
		rejected: 'bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-300',
		deferred: 'bg-gray-100 text-gray-800 dark:bg-gray-900/30 dark:text-gray-300'
	};

	const noteTypeIcons: Record<CopilotNoteType, typeof Lightbulb> = {
		suggestion: Sparkles,
		warning: AlertTriangle,
		error: AlertCircle,
		info: Lightbulb,
		performance: TrendingUp,
		security: AlertTriangle
	};

	const noteTypeColors: Record<CopilotNoteType, string> = {
		suggestion: 'text-violet-500',
		warning: 'text-amber-500',
		error: 'text-red-500',
		info: 'text-blue-500',
		performance: 'text-cyan-500',
		security: 'text-orange-500'
	};

	// Per-card status mutations — keyed by candidate id
	type StatusUpdater = ReturnType<typeof useUpdateCandidateStatus>;
	const statusMutations = new Map<string, StatusUpdater>();

	function getStatusMutation(candidateId: string): StatusUpdater {
		if (!statusMutations.has(candidateId)) {
			statusMutations.set(candidateId, useUpdateCandidateStatus(projectId, candidateId));
		}
		return statusMutations.get(candidateId)!;
	}

	function updateStatus(candidateId: string, status: CandidateStatus) {
		const mutation = getStatusMutation(candidateId);
		mutation.mutate(
			{ status },
			{
				onSuccess: () => {
					toast.success(`Status updated to ${statusLabels[status]}`);
				},
				onError: (err) => {
					toast.error(err.message ?? 'Failed to update status');
				}
			}
		);
	}

	function navigateToSpec(candidateId: string) {
		goto(`/${orgSlug}/projects/${projectId}/candidates/${candidateId}/spec`);
	}
</script>

<svelte:head>
	<title>Candidates — Neuco</title>
</svelte:head>

<div class="flex h-full">
	<!-- Main content -->
	<div class="flex flex-1 flex-col gap-6 p-6 min-w-0">
		<!-- Header -->
		<div class="flex items-center justify-between">
			<div>
				<h2 class="text-lg font-semibold tracking-tight">Feature Candidates</h2>
				<p class="text-sm text-muted-foreground mt-0.5">
					AI-extracted feature opportunities ranked by signal strength
				</p>
			</div>
			<Button onclick={handleRefresh} disabled={refreshMutation.isPending} class="gap-2">
				<RefreshCw class={cn('h-4 w-4', refreshMutation.isPending && 'animate-spin')} />
				{refreshMutation.isPending ? 'Refreshing…' : 'Refresh Candidates'}
			</Button>
		</div>

		<!-- Loading skeletons -->
		{#if candidatesQuery.isLoading}
			<div class="flex flex-col gap-3">
				{#each Array(5) as _, i (i)}
					<Card>
						<CardContent class="p-4">
							<div class="flex items-start gap-4">
								<Skeleton class="h-8 w-8 rounded-full shrink-0" />
								<div class="flex-1 space-y-2">
									<Skeleton class="h-5 w-64" />
									<Skeleton class="h-4 w-full max-w-lg" />
									<div class="flex gap-2 pt-1">
										<Skeleton class="h-5 w-16 rounded-full" />
										<Skeleton class="h-5 w-20 rounded-full" />
									</div>
								</div>
							</div>
						</CardContent>
					</Card>
				{/each}
			</div>
		{:else if candidatesQuery.isError}
			<Alert variant="destructive">
				<AlertCircle class="h-4 w-4" />
				<AlertTitle>Failed to load candidates</AlertTitle>
				<AlertDescription class="flex items-center gap-3">
					{candidatesQuery.error?.message ?? 'An unexpected error occurred.'}
					<Button
						variant="outline"
						size="sm"
						class="h-7 text-xs"
						onclick={() => candidatesQuery.refetch()}
					>
						Retry
					</Button>
				</AlertDescription>
			</Alert>
		{:else if sortedCandidates.length === 0}
			<div
				class="flex flex-col items-center justify-center rounded-xl border-2 border-dashed border-border py-20 text-center"
			>
				<Lightbulb class="h-12 w-12 text-muted-foreground mb-4" />
				<h3 class="text-lg font-semibold">No candidates yet</h3>
				<p class="text-sm text-muted-foreground mt-2 max-w-sm">
					Click "Refresh Candidates" to have the AI analyze your signals and extract feature
					opportunities.
				</p>
				<Button
					onclick={handleRefresh}
					disabled={refreshMutation.isPending}
					class="mt-6 gap-2"
				>
					<RefreshCw
						class={cn('h-4 w-4', refreshMutation.isPending && 'animate-spin')}
					/>
					{refreshMutation.isPending ? 'Analyzing…' : 'Refresh Candidates'}
				</Button>
			</div>
		{:else}
			<div class="flex flex-col gap-3">
				{#each sortedCandidates as candidate, index (candidate.id)}
					<Card
						class="group relative cursor-pointer hover:shadow-md transition-shadow border-border hover:border-primary/30"
					>
						<CardContent class="p-4">
							<div class="flex items-start gap-4">
								<!-- Rank indicator -->
								<div
									class="flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-muted text-xs font-bold text-muted-foreground"
								>
									#{index + 1}
								</div>

								<!-- Content -->
								<div class="flex-1 min-w-0">
									<div class="flex items-start justify-between gap-3">
										<button
											class="font-semibold text-foreground leading-tight text-left hover:text-primary transition-colors"
											onclick={() => navigateToSpec(candidate.id)}
										>
											{candidate.title}
										</button>
										<div
											class="flex items-center gap-1 shrink-0 text-sm font-semibold text-primary"
										>
											<TrendingUp class="h-3.5 w-3.5" />
											{candidate.priority.toFixed(1)}
										</div>
									</div>

									{#if candidate.description}
										<p class="text-sm text-muted-foreground mt-1 line-clamp-2">
											{candidate.description}
										</p>
									{/if}

									{#if candidate.rationale}
										<p class="text-xs text-muted-foreground mt-1 line-clamp-1 italic">
											{candidate.rationale}
										</p>
									{/if}

									<!-- Badges row -->
									<div class="flex items-center gap-2 mt-3 flex-wrap">
										<!-- Signal count badge -->
										<Badge variant="secondary" class="text-xs gap-1">
											{candidate.signalIds?.length ?? 0}
											{(candidate.signalIds?.length ?? 0) === 1 ? 'signal' : 'signals'}
										</Badge>

										<!-- Status badge with dropdown -->
										<DropdownMenu>
											<DropdownMenuTrigger>
												<button
													class={cn(
														'inline-flex items-center gap-1.5 rounded-full px-2.5 py-0.5 text-xs font-medium transition-colors',
														statusColors[candidate.status]
													)}
													aria-label="Change status for {candidate.title}"
													onclick={(e) => e.stopPropagation()}
												>
													{statusLabels[candidate.status]}
													<ChevronDown class="h-3 w-3 opacity-60" />
												</button>
											</DropdownMenuTrigger>
											<DropdownMenuContent align="start" class="min-w-[140px]">
												<DropdownMenuLabel class="text-xs">Update status</DropdownMenuLabel>
												<DropdownMenuSeparator />
												{#each (Object.entries(statusLabels) as [CandidateStatus, string][]) as [val, label] (val)}
													<DropdownMenuItem
														onSelect={() => updateStatus(candidate.id, val)}
														class={cn(
															'text-xs',
															candidate.status === val && 'font-semibold'
														)}
													>
														{label}
													</DropdownMenuItem>
												{/each}
											</DropdownMenuContent>
										</DropdownMenu>
									</div>
								</div>

								<!-- Arrow -->
								<button
									class="p-1 rounded-sm text-muted-foreground hover:text-foreground opacity-0 group-hover:opacity-100 transition-opacity shrink-0 mt-1"
									aria-label="View spec"
									onclick={() => navigateToSpec(candidate.id)}
								>
									<ArrowRight class="h-4 w-4" />
								</button>
							</div>
						</CardContent>
					</Card>
				{/each}
			</div>
		{/if}
	</div>

	<!-- Co-pilot sidebar -->
	{#if copilotNotes.length > 0}
		<div class="w-80 shrink-0 border-l border-border p-5 flex flex-col gap-4 overflow-y-auto">
			<div class="flex items-center gap-2">
				<Sparkles class="h-4 w-4 text-violet-500" />
				<h3 class="text-sm font-semibold">Co-pilot Insights</h3>
				<Badge variant="secondary" class="ml-auto text-xs">{copilotNotes.length}</Badge>
			</div>

			<div class="flex flex-col gap-3">
				{#each copilotNotes as note (note.id)}
					{@const IconComponent = noteTypeIcons[note.type] ?? Lightbulb}
					<div class="rounded-lg border border-border bg-card p-3 relative group/note">
						<div class="flex items-start gap-2">
							<IconComponent
								class="h-4 w-4 shrink-0 mt-0.5 {noteTypeColors[note.type]}"
							/>
							<div class="flex-1 min-w-0">
								{#if note.title}
									<p class="text-xs font-semibold text-foreground mb-1">{note.title}</p>
								{/if}
								<p class="text-xs text-muted-foreground leading-relaxed">{note.body}</p>
							</div>
							<button
								class="h-5 w-5 rounded-sm flex items-center justify-center text-muted-foreground hover:text-foreground hover:bg-accent transition-colors opacity-0 group-hover/note:opacity-100 shrink-0"
								aria-label="Dismiss note"
								onclick={() => handleDismissNote(note.id)}
							>
								<X class="h-3 w-3" />
							</button>
						</div>
					</div>
				{/each}
			</div>
		</div>
	{/if}
</div>
