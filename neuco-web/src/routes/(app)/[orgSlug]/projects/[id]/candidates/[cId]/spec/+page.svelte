<script lang="ts">
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import { useSpecByCandidate, useGenerateSpec, useUpdateSpec } from '$lib/api/queries/specs';
	import { useCopilotNotes, useDismissNote } from '$lib/api/queries/copilot';
	import { Button } from '$lib/components/ui/button';
	import { Input } from '$lib/components/ui/input';
	import { Label } from '$lib/components/ui/label';
	import { Badge } from '$lib/components/ui/badge';
	import { Skeleton } from '$lib/components/ui/skeleton';
	import { Alert, AlertDescription, AlertTitle } from '$lib/components/ui/alert';
	import { Separator } from '$lib/components/ui/separator';
	import { toast } from '$lib/components/ui/sonner';
	import { trackSpecGenerated } from '$lib/analytics';
	import {
		Sparkles,
		AlertCircle,
		AlertTriangle,
		Lightbulb,
		TrendingUp,
		Plus,
		Trash2,
		X,
		Code2,
		FileText,
		Save,
		GitBranch
	} from 'lucide-svelte';
	import { cn } from '$lib/utils';
	import type { UserStory, CopilotNoteType } from '$lib/api/types-compat';

	const projectId = $derived($page.params.id ?? '');
	const candidateId = $derived($page.params.cId ?? '');
	const orgSlug = $derived($page.params.orgSlug ?? '');

	// ── Spec query ─────────────────────────────────────────────────────────────
	const specQuery = $derived(useSpecByCandidate(projectId, candidateId));
	const generateMutation = $derived(useGenerateSpec(projectId, candidateId));

	// We always create the update mutation — it only fires when specId is valid
	let specId = $state('');
	$effect(() => {
		if (specQuery.data?.id) {
			specId = specQuery.data.id;
		}
	});
	const updateMutation = $derived(useUpdateSpec(projectId, specId));

	// ── Copilot notes (spec review types) ─────────────────────────────────────
	const copilotQuery = $derived(useCopilotNotes(projectId));
	const dismissMutation = $derived(useDismissNote(projectId));

	const specCopilotNotes = $derived(
		(copilotQuery.data ?? []).filter(
			(n) =>
				!n.dismissed &&
				(n.entityType === 'spec' || n.entityType === 'spec_review') &&
				(n.entityId === candidateId || !n.entityId)
		)
	);

	// ── Form state ─────────────────────────────────────────────────────────────
	let formTitle = $state('');
	let formSummary = $state('');
	let formTechnicalNotes = $state('');
	let formUserStories = $state<UserStory[]>([]);
	let isDirty = $state(false);
	let syncedSpecId = $state('');

	// Sync form when spec data arrives (only once per spec version)
	$effect(() => {
		const spec = specQuery.data;
		if (spec && spec.id !== syncedSpecId) {
			syncedSpecId = spec.id;
			formTitle = spec.title ?? '';
			formSummary = spec.summary ?? '';
			formTechnicalNotes = spec.technicalNotes ?? '';
			formUserStories = spec.userStories
				? spec.userStories.map((s) => ({
						...s,
						acceptanceCriteria: [...(s.acceptanceCriteria ?? [])]
					}))
				: [];
			isDirty = false;
		}
	});

	function markDirty() {
		isDirty = true;
	}

	// ── User stories helpers ───────────────────────────────────────────────────
	function addUserStory() {
		formUserStories = [
			...formUserStories,
			{
				id: crypto.randomUUID(),
				asA: '',
				iWantTo: '',
				soThat: '',
				acceptanceCriteria: []
			}
		];
		markDirty();
	}

	function removeUserStory(index: number) {
		formUserStories = formUserStories.filter((_, i) => i !== index);
		markDirty();
	}

	function updateUserStoryField(index: number, field: keyof UserStory, value: string) {
		formUserStories = formUserStories.map((s, i) =>
			i === index ? { ...s, [field]: value } : s
		);
		markDirty();
	}

	function addAcceptanceCriteria(storyIndex: number) {
		formUserStories = formUserStories.map((s, i) =>
			i === storyIndex
				? { ...s, acceptanceCriteria: [...(s.acceptanceCriteria ?? []), ''] }
				: s
		);
		markDirty();
	}

	function updateAcceptanceCriteria(storyIndex: number, acIndex: number, value: string) {
		formUserStories = formUserStories.map((s, i) => {
			if (i !== storyIndex) return s;
			const updated = [...(s.acceptanceCriteria ?? [])];
			updated[acIndex] = value;
			return { ...s, acceptanceCriteria: updated };
		});
		markDirty();
	}

	function removeAcceptanceCriteria(storyIndex: number, acIndex: number) {
		formUserStories = formUserStories.map((s, i) => {
			if (i !== storyIndex) return s;
			return {
				...s,
				acceptanceCriteria: (s.acceptanceCriteria ?? []).filter((_, idx) => idx !== acIndex)
			};
		});
		markDirty();
	}

	// ── Save ───────────────────────────────────────────────────────────────────
	function handleSave() {
		if (!specId) return;
		updateMutation.mutate(
			{
				title: formTitle,
				summary: formSummary,
				userStories: formUserStories,
				technicalNotes: formTechnicalNotes
			},
			{
				onSuccess: () => {
					toast.success('Spec saved successfully');
					isDirty = false;
				},
				onError: (err) => {
					toast.error(err.message ?? 'Failed to save spec');
				}
			}
		);
	}

	// ── Generate spec ──────────────────────────────────────────────────────────
	function handleGenerate() {
		generateMutation.mutate(undefined, {
			onSuccess: () => {
				trackSpecGenerated(projectId, candidateId);
				toast.success('Spec generated successfully');
			},
			onError: (err) => {
				toast.error(err.message ?? 'Failed to generate spec');
			}
		});
	}

	function handleGenerateCode() {
		goto(`/${orgSlug}/projects/${projectId}/candidates/${candidateId}/generate`);
	}

	function handleDismissNote(noteId: string) {
		dismissMutation.mutate(noteId, {
			onError: (err) => toast.error(err.message ?? 'Failed to dismiss note')
		});
	}

	// ── Copilot icon helpers ───────────────────────────────────────────────────
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

	// Determine if no-spec state (404 or empty query error)
	const hasNoSpec = $derived(
		specQuery.isError &&
			((specQuery.error as { status?: number })?.status === 404 ||
				(specQuery.error as { status?: number })?.status === undefined)
	);
</script>

<svelte:head>
	<title>Spec — Neuco</title>
</svelte:head>

<div class="flex h-full">
	<!-- Main spec content -->
	<div class="flex flex-1 flex-col min-w-0 overflow-y-auto">
		<div class="p-6 flex flex-col gap-6 max-w-4xl w-full">
			<!-- Header -->
			<div class="flex items-start justify-between gap-4">
				<div>
					<h2 class="text-lg font-semibold tracking-tight">Feature Spec</h2>
					<p class="text-sm text-muted-foreground mt-0.5">
						Define requirements and acceptance criteria for this feature
					</p>
				</div>
				{#if specQuery.data}
					<div class="flex items-center gap-2 shrink-0">
						<span class="text-xs text-muted-foreground">v{specQuery.data.version}</span>
						<Button variant="outline" onclick={handleGenerateCode} class="gap-2">
							<Code2 class="h-4 w-4" />
							Generate Code
						</Button>
						<Button
							onclick={handleSave}
							disabled={!isDirty || updateMutation.isPending || !specId}
							class="gap-2"
						>
							{#if updateMutation.isPending}
								<svg
									class="h-4 w-4 animate-spin"
									xmlns="http://www.w3.org/2000/svg"
									fill="none"
									viewBox="0 0 24 24"
								>
									<circle
										class="opacity-25"
										cx="12"
										cy="12"
										r="10"
										stroke="currentColor"
										stroke-width="4"
									></circle>
									<path
										class="opacity-75"
										fill="currentColor"
										d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"
									></path>
								</svg>
								Saving…
							{:else}
								<Save class="h-4 w-4" />
								Save Changes
							{/if}
						</Button>
					</div>
				{/if}
			</div>

			<!-- Loading skeleton -->
			{#if specQuery.isLoading}
				<div class="space-y-6">
					{#each Array(4) as _, i (i)}
						<div class="space-y-2">
							<Skeleton class="h-4 w-32" />
							<Skeleton class="h-24 w-full rounded-md" />
						</div>
					{/each}
				</div>

			<!-- No spec yet (404) -->
			{:else if hasNoSpec}
				<div
					class="flex flex-col items-center justify-center rounded-xl border-2 border-dashed border-border py-20 text-center"
				>
					<FileText class="h-12 w-12 text-muted-foreground mb-4" />
					<h3 class="text-lg font-semibold">No spec yet</h3>
					<p class="text-sm text-muted-foreground mt-2 max-w-sm">
						Generate an AI-powered spec from the signals and candidate data for this feature.
					</p>
					<Button
						onclick={handleGenerate}
						disabled={generateMutation.isPending}
						class="mt-6 gap-2"
					>
						{#if generateMutation.isPending}
							<svg
								class="h-4 w-4 animate-spin"
								xmlns="http://www.w3.org/2000/svg"
								fill="none"
								viewBox="0 0 24 24"
							>
								<circle
									class="opacity-25"
									cx="12"
									cy="12"
									r="10"
									stroke="currentColor"
									stroke-width="4"
								></circle>
								<path
									class="opacity-75"
									fill="currentColor"
									d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"
								></path>
							</svg>
							Generating spec…
						{:else}
							<Sparkles class="h-4 w-4" />
							Generate Spec
						{/if}
					</Button>
					{#if generateMutation.isPending}
						<p class="text-xs text-muted-foreground mt-3 animate-pulse">
							Analyzing signals and building your spec…
						</p>
					{/if}
				</div>

			<!-- API error (non-404) -->
			{:else if specQuery.isError}
				<Alert variant="destructive">
					<AlertCircle class="h-4 w-4" />
					<AlertTitle>Failed to load spec</AlertTitle>
					<AlertDescription class="flex items-center gap-3">
						{specQuery.error?.message ?? 'An unexpected error occurred.'}
						<Button
							variant="outline"
							size="sm"
							class="h-7 text-xs"
							onclick={() => specQuery.refetch()}
						>
							Retry
						</Button>
					</AlertDescription>
				</Alert>

			<!-- Spec form -->
			{:else if specQuery.data}
				<div class="space-y-8">
					<!-- Title -->
					<div class="space-y-2">
						<Label for="spec-title" class="text-sm font-semibold">Title</Label>
						<Input
							id="spec-title"
							value={formTitle}
							oninput={(e) => {
								formTitle = (e.currentTarget as HTMLInputElement).value;
								markDirty();
							}}
							placeholder="Feature title"
							class="text-base font-medium"
						/>
					</div>

					<!-- Problem Statement -->
					<div class="space-y-2">
						<Label for="spec-summary" class="text-sm font-semibold">Problem Statement</Label>
						<textarea
							id="spec-summary"
							class="flex min-h-[120px] w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50 resize-y"
							value={formSummary}
							oninput={(e) => {
								formSummary = (e.currentTarget as HTMLTextAreaElement).value;
								markDirty();
							}}
							placeholder="Describe the problem this feature solves…"
						></textarea>
					</div>

					<Separator />

					<!-- User Stories -->
					<div class="space-y-4">
						<div class="flex items-center justify-between">
							<Label class="text-sm font-semibold">User Stories</Label>
							<Button
								variant="outline"
								size="sm"
								class="gap-1.5 text-xs h-8"
								onclick={addUserStory}
							>
								<Plus class="h-3.5 w-3.5" />
								Add story
							</Button>
						</div>

						{#if formUserStories.length === 0}
							<p class="text-sm text-muted-foreground italic">
								No user stories yet. Add one above.
							</p>
						{:else}
							<div class="space-y-4">
								{#each formUserStories as story, si (story.id)}
									<div class="rounded-lg border border-border p-4 space-y-3 relative">
										<button
											class="absolute right-3 top-3 h-6 w-6 rounded-sm flex items-center justify-center text-muted-foreground hover:text-destructive hover:bg-destructive/10 transition-colors"
											aria-label="Remove user story"
											onclick={() => removeUserStory(si)}
										>
											<Trash2 class="h-3.5 w-3.5" />
										</button>

										<!-- As a -->
										<div class="grid grid-cols-[72px_1fr] items-center gap-2">
											<Label
												for="story-{si}-as-a"
												class="text-xs text-muted-foreground font-medium text-right"
											>
												As a
											</Label>
											<Input
												id="story-{si}-as-a"
												value={story.asA}
												oninput={(e) =>
													updateUserStoryField(
														si,
														'asA',
														(e.currentTarget as HTMLInputElement).value
													)}
												placeholder="product manager"
												class="h-8 text-sm"
											/>
										</div>

										<!-- I want to -->
										<div class="grid grid-cols-[72px_1fr] items-center gap-2">
											<Label
												for="story-{si}-want"
												class="text-xs text-muted-foreground font-medium text-right"
											>
												I want to
											</Label>
											<Input
												id="story-{si}-want"
												value={story.iWantTo}
												oninput={(e) =>
													updateUserStoryField(
														si,
														'iWantTo',
														(e.currentTarget as HTMLInputElement).value
													)}
												placeholder="filter candidates by status"
												class="h-8 text-sm"
											/>
										</div>

										<!-- So that -->
										<div class="grid grid-cols-[72px_1fr] items-center gap-2">
											<Label
												for="story-{si}-so-that"
												class="text-xs text-muted-foreground font-medium text-right"
											>
												So that
											</Label>
											<Input
												id="story-{si}-so-that"
												value={story.soThat}
												oninput={(e) =>
													updateUserStoryField(
														si,
														'soThat',
														(e.currentTarget as HTMLInputElement).value
													)}
												placeholder="I can prioritize my backlog"
												class="h-8 text-sm"
											/>
										</div>

										<!-- Acceptance criteria -->
										<div class="space-y-2 pt-1">
											<div class="flex items-center justify-between">
												<span class="text-xs font-medium text-muted-foreground">
													Acceptance Criteria
												</span>
												<button
													class="text-xs text-primary hover:underline"
													onclick={() => addAcceptanceCriteria(si)}
												>
													+ Add criterion
												</button>
											</div>
											{#if (story.acceptanceCriteria ?? []).length === 0}
												<p class="text-xs text-muted-foreground italic">None added.</p>
											{:else}
												<div class="space-y-1.5">
													{#each story.acceptanceCriteria ?? [] as criterion, ai (ai)}
														<div class="flex items-center gap-2">
															<label
																for="ac-{si}-{ai}"
																class="text-xs text-muted-foreground w-4 shrink-0 text-right"
															>
																{ai + 1}.
															</label>
															<Input
																id="ac-{si}-{ai}"
																value={criterion}
																oninput={(e) =>
																	updateAcceptanceCriteria(
																		si,
																		ai,
																		(e.currentTarget as HTMLInputElement).value
																	)}
																placeholder="Given… when… then…"
																class="h-8 text-xs flex-1"
															/>
															<button
																class="h-6 w-6 shrink-0 rounded-sm flex items-center justify-center text-muted-foreground hover:text-destructive hover:bg-destructive/10 transition-colors"
																aria-label="Remove criterion"
																onclick={() => removeAcceptanceCriteria(si, ai)}
															>
																<X class="h-3 w-3" />
															</button>
														</div>
													{/each}
												</div>
											{/if}
										</div>
									</div>
								{/each}
							</div>
						{/if}
					</div>

					<Separator />

					<!-- Technical Notes -->
					<div class="space-y-2">
						<Label for="spec-technical" class="text-sm font-semibold">Technical Notes</Label>
						<p class="text-xs text-muted-foreground">
							Architecture decisions, data model changes, UI considerations
						</p>
						<textarea
							id="spec-technical"
							class="flex min-h-[160px] w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50 resize-y font-mono"
							value={formTechnicalNotes}
							oninput={(e) => {
								formTechnicalNotes = (e.currentTarget as HTMLTextAreaElement).value;
								markDirty();
							}}
							placeholder="Describe technical implementation details, data model changes, UI changes, and open questions…"
						></textarea>
					</div>

					<!-- Footer actions -->
					<div class="flex items-center justify-between pt-2 pb-6">
						<div class="flex items-center gap-2 text-xs text-muted-foreground">
							<GitBranch class="h-3.5 w-3.5" />
							<span>Version {specQuery.data.version}</span>
							{#if isDirty}
								<span class="text-amber-500 font-medium">• Unsaved changes</span>
							{/if}
						</div>
						<div class="flex items-center gap-2">
							<Button variant="outline" onclick={handleGenerateCode} class="gap-2">
								<Code2 class="h-4 w-4" />
								Generate Code
							</Button>
							<Button
								onclick={handleSave}
								disabled={!isDirty || updateMutation.isPending || !specId}
								class="gap-2"
							>
								{#if updateMutation.isPending}
									<svg
										class="h-4 w-4 animate-spin"
										xmlns="http://www.w3.org/2000/svg"
										fill="none"
										viewBox="0 0 24 24"
									>
										<circle
											class="opacity-25"
											cx="12"
											cy="12"
											r="10"
											stroke="currentColor"
											stroke-width="4"
										></circle>
										<path
											class="opacity-75"
											fill="currentColor"
											d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"
										></path>
									</svg>
									Saving…
								{:else}
									<Save class="h-4 w-4" />
									Save Changes
								{/if}
							</Button>
						</div>
					</div>
				</div>
			{/if}
		</div>
	</div>

	<!-- Co-pilot sidebar -->
	{#if specCopilotNotes.length > 0}
		<div
			class="w-80 shrink-0 border-l border-border p-5 flex flex-col gap-4 overflow-y-auto"
		>
			<div class="flex items-center gap-2">
				<Sparkles class="h-4 w-4 text-violet-500" />
				<h3 class="text-sm font-semibold">Spec Review</h3>
				<Badge variant="secondary" class="ml-auto text-xs">{specCopilotNotes.length}</Badge>
			</div>

			<div class="flex flex-col gap-3">
				{#each specCopilotNotes as note (note.id)}
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
