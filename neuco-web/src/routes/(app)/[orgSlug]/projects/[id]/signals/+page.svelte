<script lang="ts">
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import { useSignals, useUploadSignals, useDeleteSignal } from '$lib/api/queries/signals';
	import { Button } from '$lib/components/ui/button';
	import { Input } from '$lib/components/ui/input';
	import { Skeleton } from '$lib/components/ui/skeleton';
	import { Alert, AlertDescription, AlertTitle } from '$lib/components/ui/alert';
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
	import {
		Table,
		TableBody,
		TableCell,
		TableHead,
		TableHeader,
		TableRow
	} from '$lib/components/ui/table';
	import { toast } from '$lib/components/ui/sonner';
	import {
		Upload,
		Radio,
		AlertCircle,
		Trash2,
		ChevronLeft,
		ChevronRight,
		X,
		Filter
	} from 'lucide-svelte';
	import { cn } from '$lib/utils';
	import type { SignalSource, SignalType } from '$lib/api/types';

	const projectId = $derived($page.params.id ?? '');

	// ── Filter state (synced with URL search params) ──────────────────────────
	let filterSource = $state<SignalSource | ''>('');
	let filterType = $state<SignalType | ''>('');
	let filterDateFrom = $state('');
	let filterDateTo = $state('');
	let currentPage = $state(1);
	const pageSize = 20;

	// Sync filters from URL on mount
	$effect(() => {
		const sp = $page.url.searchParams;
		filterSource = (sp.get('source') as SignalSource) ?? '';
		filterType = (sp.get('type') as SignalType) ?? '';
		filterDateFrom = sp.get('from') ?? '';
		filterDateTo = sp.get('to') ?? '';
		currentPage = parseInt(sp.get('page') ?? '1', 10) || 1;
	});

	function updateSearchParams() {
		const sp = new URLSearchParams($page.url.searchParams);
		if (filterSource) sp.set('source', filterSource);
		else sp.delete('source');
		if (filterType) sp.set('type', filterType);
		else sp.delete('type');
		if (filterDateFrom) sp.set('from', filterDateFrom);
		else sp.delete('from');
		if (filterDateTo) sp.set('to', filterDateTo);
		else sp.delete('to');
		sp.set('page', String(currentPage));
		goto(`?${sp.toString()}`, { replaceState: true, noScroll: true });
	}

	function resetFilters() {
		filterSource = '';
		filterType = '';
		filterDateFrom = '';
		filterDateTo = '';
		currentPage = 1;
		updateSearchParams();
	}

	// ── Signal query ──────────────────────────────────────────────────────────
	const signalsQuery = $derived(
		useSignals(projectId, {
			page: currentPage,
			pageSize,
			source: filterSource || undefined,
			type: filterType || undefined
		})
	);

	// ── Upload state ──────────────────────────────────────────────────────────
	const uploadMutation = $derived(useUploadSignals(projectId));
	let isDragging = $state(false);
	let uploadProgress = $state<number | null>(null);
	let fileInputEl = $state<HTMLInputElement | null>(null);

	function handleDragOver(e: DragEvent) {
		e.preventDefault();
		isDragging = true;
	}

	function handleDragLeave(e: DragEvent) {
		if (!(e.currentTarget as HTMLElement).contains(e.relatedTarget as Node)) {
			isDragging = false;
		}
	}

	function handleDrop(e: DragEvent) {
		e.preventDefault();
		isDragging = false;
		const files = e.dataTransfer?.files;
		if (files && files.length > 0) {
			uploadFiles(files);
		}
	}

	function handleFileInputChange(e: Event) {
		const input = e.currentTarget as HTMLInputElement;
		if (input.files && input.files.length > 0) {
			uploadFiles(input.files);
			input.value = '';
		}
	}

	function uploadFiles(files: FileList) {
		const allowed = ['text/csv', 'text/plain', 'application/csv'];
		for (const file of Array.from(files)) {
			if (!allowed.includes(file.type) && !file.name.match(/\.(csv|txt)$/i)) {
				toast.error(`"${file.name}" is not a CSV or text file`);
				return;
			}
		}

		const formData = new FormData();
		for (const file of Array.from(files)) {
			formData.append('files', file);
		}

		uploadProgress = 0;
		const interval = setInterval(() => {
			if (uploadProgress !== null && uploadProgress < 85) {
				uploadProgress += 5;
			}
		}, 200);

		uploadMutation.mutate(formData, {
			onSuccess: (signals) => {
				clearInterval(interval);
				uploadProgress = 100;
				setTimeout(() => {
					uploadProgress = null;
				}, 1200);
				toast.success(
					`Uploaded ${signals.length} signal${signals.length === 1 ? '' : 's'} successfully`
				);
			},
			onError: (err) => {
				clearInterval(interval);
				uploadProgress = null;
				toast.error(err.message ?? 'Upload failed');
			}
		});
	}

	// ── Delete ─────────────────────────────────────────────────────────────────
	const deleteMutation = $derived(useDeleteSignal(projectId));
	let deleteTargetId = $state<string | null>(null);
	let confirmDeleteOpen = $state(false);

	function promptDelete(signalId: string) {
		deleteTargetId = signalId;
		confirmDeleteOpen = true;
	}

	function confirmDelete() {
		if (!deleteTargetId) return;
		deleteMutation.mutate(deleteTargetId, {
			onSuccess: () => {
				toast.success('Signal deleted');
				confirmDeleteOpen = false;
				deleteTargetId = null;
			},
			onError: (err) => {
				toast.error(err.message ?? 'Failed to delete signal');
			}
		});
	}

	// ── Pagination helpers ────────────────────────────────────────────────────
	function goToPage(p: number) {
		currentPage = p;
		updateSearchParams();
	}

	const totalPages = $derived(signalsQuery.data?.totalPages ?? 1);

	// ── Label maps ────────────────────────────────────────────────────────────
	const sourceLabels: Record<SignalSource, string> = {
		github_issue: 'GitHub Issue',
		github_pr: 'GitHub PR',
		slack: 'Slack',
		csv: 'CSV',
		api: 'API',
		manual: 'Manual'
	};

	const sourceColors: Record<SignalSource, string> = {
		github_issue: 'bg-violet-100 text-violet-800 dark:bg-violet-900/30 dark:text-violet-300',
		github_pr: 'bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-300',
		slack: 'bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-300',
		csv: 'bg-amber-100 text-amber-800 dark:bg-amber-900/30 dark:text-amber-300',
		api: 'bg-cyan-100 text-cyan-800 dark:bg-cyan-900/30 dark:text-cyan-300',
		manual: 'bg-gray-100 text-gray-800 dark:bg-gray-900/30 dark:text-gray-300'
	};

	const typeLabels: Record<SignalType, string> = {
		bug: 'Bug',
		feature_request: 'Feature Request',
		improvement: 'Improvement',
		question: 'Question',
		other: 'Other'
	};

	const typeColors: Record<SignalType, string> = {
		bug: 'bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-300',
		feature_request: 'bg-indigo-100 text-indigo-800 dark:bg-indigo-900/30 dark:text-indigo-300',
		improvement: 'bg-teal-100 text-teal-800 dark:bg-teal-900/30 dark:text-teal-300',
		question: 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-300',
		other: 'bg-gray-100 text-gray-800 dark:bg-gray-900/30 dark:text-gray-300'
	};

	function formatDate(dateStr: string): string {
		if (!dateStr) return '—';
		return new Date(dateStr).toLocaleDateString(undefined, {
			year: 'numeric',
			month: 'short',
			day: 'numeric'
		});
	}

	function truncate(text: string, max = 100): string {
		if (!text) return '—';
		return text.length > max ? `${text.slice(0, max)}\u2026` : text;
	}

	const hasActiveFilters = $derived(
		!!filterSource || !!filterType || !!filterDateFrom || !!filterDateTo
	);

	const sourceLabel = $derived(
		filterSource ? sourceLabels[filterSource as SignalSource] : 'All sources'
	);
	const typeLabel = $derived(
		filterType ? typeLabels[filterType as SignalType] : 'All types'
	);
</script>

<svelte:head>
	<title>Signals — Neuco</title>
</svelte:head>

<div class="flex flex-col gap-6 p-6">
	<!-- Upload zone -->
	<div
		role="button"
		tabindex="0"
		aria-label="Drop CSV or text files to upload signals"
		class={cn(
			'relative flex flex-col items-center justify-center rounded-xl border-2 border-dashed p-8 text-center transition-colors cursor-pointer select-none',
			isDragging
				? 'border-primary bg-primary/5'
				: 'border-border hover:border-primary/50 hover:bg-accent/30'
		)}
		ondragover={handleDragOver}
		ondragleave={handleDragLeave}
		ondrop={handleDrop}
		onclick={() => fileInputEl?.click()}
		onkeydown={(e) => e.key === 'Enter' && fileInputEl?.click()}
	>
		<input
			bind:this={fileInputEl}
			type="file"
			accept=".csv,.txt,text/csv,text/plain"
			multiple
			class="sr-only"
			onchange={handleFileInputChange}
		/>

		{#if uploadMutation.isPending || uploadProgress !== null}
			<div class="flex flex-col items-center gap-3 w-full max-w-sm">
				<div class="flex items-center gap-2 text-sm font-medium text-foreground">
					<svg
						class="h-4 w-4 animate-spin text-primary"
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
					{uploadProgress === 100 ? 'Processing…' : 'Uploading…'}
				</div>
				<div class="w-full rounded-full bg-muted h-2 overflow-hidden">
					<div
						class="h-full bg-primary rounded-full transition-all duration-200"
						style="width: {uploadProgress ?? 0}%"
					></div>
				</div>
				<p class="text-xs text-muted-foreground">{uploadProgress ?? 0}%</p>
			</div>
		{:else}
			<Upload class="h-8 w-8 text-muted-foreground mb-3" />
			<p class="text-sm font-medium text-foreground">
				Drop CSV or text files here, or click to browse
			</p>
			<p class="text-xs text-muted-foreground mt-1">
				Supports .csv and .txt files — multiple files allowed
			</p>
		{/if}
	</div>

	<!-- Filter bar -->
	<div class="flex flex-wrap items-end gap-3">
		<div class="flex items-center gap-1.5 text-sm text-muted-foreground mr-1">
			<Filter class="h-4 w-4" />
			<span class="font-medium">Filters</span>
		</div>

		<!-- Source filter -->
		<div class="min-w-[160px]">
			<Select
				value={filterSource}
				onValueChange={(v) => {
					filterSource = v === 'all' ? '' : (v as SignalSource);
					currentPage = 1;
					updateSearchParams();
				}}
			>
				<SelectTrigger class="h-9 text-xs">
					<span>{sourceLabel}</span>
				</SelectTrigger>
				<SelectContent>
					<SelectItem value="all" label="All sources" />
					<SelectItem value="github_issue" label="GitHub Issue" />
					<SelectItem value="github_pr" label="GitHub PR" />
					<SelectItem value="slack" label="Slack" />
					<SelectItem value="csv" label="CSV" />
					<SelectItem value="api" label="API" />
					<SelectItem value="manual" label="Manual" />
				</SelectContent>
			</Select>
		</div>

		<!-- Type filter -->
		<div class="min-w-[160px]">
			<Select
				value={filterType}
				onValueChange={(v) => {
					filterType = v === 'all' ? '' : (v as SignalType);
					currentPage = 1;
					updateSearchParams();
				}}
			>
				<SelectTrigger class="h-9 text-xs">
					<span>{typeLabel}</span>
				</SelectTrigger>
				<SelectContent>
					<SelectItem value="all" label="All types" />
					<SelectItem value="bug" label="Bug" />
					<SelectItem value="feature_request" label="Feature Request" />
					<SelectItem value="improvement" label="Improvement" />
					<SelectItem value="question" label="Question" />
					<SelectItem value="other" label="Other" />
				</SelectContent>
			</Select>
		</div>

		<!-- Date from -->
		<div class="flex flex-col gap-1">
			<label for="filter-date-from" class="text-xs text-muted-foreground">From</label>
			<Input
				id="filter-date-from"
				type="date"
				class="h-9 text-xs w-36"
				value={filterDateFrom}
				oninput={(e) => {
					filterDateFrom = (e.currentTarget as HTMLInputElement).value;
					currentPage = 1;
					updateSearchParams();
				}}
			/>
		</div>

		<!-- Date to -->
		<div class="flex flex-col gap-1">
			<label for="filter-date-to" class="text-xs text-muted-foreground">To</label>
			<Input
				id="filter-date-to"
				type="date"
				class="h-9 text-xs w-36"
				value={filterDateTo}
				oninput={(e) => {
					filterDateTo = (e.currentTarget as HTMLInputElement).value;
					currentPage = 1;
					updateSearchParams();
				}}
			/>
		</div>

		{#if hasActiveFilters}
			<Button variant="ghost" size="sm" class="h-9 gap-1.5 text-xs" onclick={resetFilters}>
				<X class="h-3.5 w-3.5" />
				Clear filters
			</Button>
		{/if}

		{#if signalsQuery.data}
			<span class="ml-auto text-xs text-muted-foreground self-end pb-1">
				{signalsQuery.data.total} signal{signalsQuery.data.total === 1 ? '' : 's'}
			</span>
		{/if}
	</div>

	<!-- Table area -->
	{#if signalsQuery.isLoading}
		<div class="rounded-lg border border-border overflow-hidden">
			<Table>
				<TableHeader>
					<TableRow>
						<TableHead class="w-40">Source</TableHead>
						<TableHead class="w-40">Type</TableHead>
						<TableHead>Content</TableHead>
						<TableHead class="w-32">Date</TableHead>
						<TableHead class="w-12"></TableHead>
					</TableRow>
				</TableHeader>
				<TableBody>
					{#each Array(8) as _, i (i)}
						<TableRow>
							<TableCell><Skeleton class="h-5 w-24 rounded-full" /></TableCell>
							<TableCell><Skeleton class="h-5 w-28 rounded-full" /></TableCell>
							<TableCell><Skeleton class="h-4 w-full max-w-sm" /></TableCell>
							<TableCell><Skeleton class="h-4 w-24" /></TableCell>
							<TableCell></TableCell>
						</TableRow>
					{/each}
				</TableBody>
			</Table>
		</div>
	{:else if signalsQuery.isError}
		<Alert variant="destructive">
			<AlertCircle class="h-4 w-4" />
			<AlertTitle>Failed to load signals</AlertTitle>
			<AlertDescription class="flex items-center gap-3">
				{signalsQuery.error?.message ?? 'An unexpected error occurred.'}
				<Button
					variant="outline"
					size="sm"
					class="h-7 text-xs"
					onclick={() => signalsQuery.refetch()}
				>
					Retry
				</Button>
			</AlertDescription>
		</Alert>
	{:else if !signalsQuery.data || signalsQuery.data.data.length === 0}
		<div
			class="flex flex-col items-center justify-center rounded-xl border-2 border-dashed border-border py-20 text-center"
		>
			<Radio class="h-12 w-12 text-muted-foreground mb-4" />
			<h3 class="text-lg font-semibold">No signals yet</h3>
			<p class="text-sm text-muted-foreground mt-2 max-w-sm">
				{hasActiveFilters
					? 'No signals match your current filters. Try clearing them.'
					: 'Upload a CSV or text file above to start ingesting user feedback signals.'}
			</p>
			{#if hasActiveFilters}
				<Button variant="outline" size="sm" class="mt-4" onclick={resetFilters}>
					Clear filters
				</Button>
			{/if}
		</div>
	{:else}
		<div class="rounded-lg border border-border overflow-hidden">
			<Table>
				<TableHeader>
					<TableRow>
						<TableHead class="w-40">Source</TableHead>
						<TableHead class="w-40">Type</TableHead>
						<TableHead>Content</TableHead>
						<TableHead class="w-32">Date</TableHead>
						<TableHead class="w-12 text-right pr-4"></TableHead>
					</TableRow>
				</TableHeader>
				<TableBody>
					{#each signalsQuery.data.data as signal (signal.id)}
						<TableRow class="group">
							<TableCell>
								<span
									class="inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium {sourceColors[
										signal.source
									]}"
								>
									{sourceLabels[signal.source]}
								</span>
							</TableCell>
							<TableCell>
								<span
									class="inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium {typeColors[
										signal.type
									]}"
								>
									{typeLabels[signal.type]}
								</span>
							</TableCell>
							<TableCell class="max-w-0">
								<p
									class="truncate text-sm text-foreground"
									title={signal.body || signal.title}
								>
									{truncate(signal.body || signal.title)}
								</p>
								{#if signal.title && signal.body}
									<p class="truncate text-xs text-muted-foreground mt-0.5">
										{signal.title}
									</p>
								{/if}
							</TableCell>
							<TableCell class="text-sm text-muted-foreground whitespace-nowrap">
								{formatDate(signal.createdAt)}
							</TableCell>
							<TableCell class="text-right pr-4">
								<button
									type="button"
									class="inline-flex h-7 w-7 items-center justify-center rounded-md text-muted-foreground opacity-0 group-hover:opacity-100 hover:text-destructive hover:bg-destructive/10 transition-opacity"
									aria-label="Delete signal"
									onclick={() => promptDelete(signal.id)}
								>
									<Trash2 class="h-3.5 w-3.5" />
								</button>
							</TableCell>
						</TableRow>
					{/each}
				</TableBody>
			</Table>
		</div>

		{#if totalPages > 1}
			<div class="flex items-center justify-between">
				<p class="text-sm text-muted-foreground">
					Page {currentPage} of {totalPages}
				</p>
				<div class="flex items-center gap-2">
					<Button
						variant="outline"
						size="sm"
						class="gap-1.5"
						disabled={currentPage <= 1}
						onclick={() => goToPage(currentPage - 1)}
					>
						<ChevronLeft class="h-4 w-4" />
						Previous
					</Button>
					<Button
						variant="outline"
						size="sm"
						class="gap-1.5"
						disabled={currentPage >= totalPages}
						onclick={() => goToPage(currentPage + 1)}
					>
						Next
						<ChevronRight class="h-4 w-4" />
					</Button>
				</div>
			</div>
		{/if}
	{/if}
</div>

<!-- Delete confirmation dialog -->
<Dialog bind:open={confirmDeleteOpen}>
	<DialogContent class="sm:max-w-[400px]">
		<DialogHeader>
			<DialogTitle>Delete signal</DialogTitle>
			<DialogDescription>
				This action cannot be undone. The signal will be permanently removed from this
				project.
			</DialogDescription>
		</DialogHeader>
		<DialogFooter>
			<Button
				variant="outline"
				onclick={() => {
					confirmDeleteOpen = false;
					deleteTargetId = null;
				}}
				disabled={deleteMutation.isPending}
			>
				Cancel
			</Button>
			<Button
				variant="destructive"
				onclick={confirmDelete}
				disabled={deleteMutation.isPending}
			>
				{#if deleteMutation.isPending}
					<svg
						class="mr-2 h-4 w-4 animate-spin"
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
					Deleting…
				{:else}
					Delete signal
				{/if}
			</Button>
		</DialogFooter>
	</DialogContent>
</Dialog>
