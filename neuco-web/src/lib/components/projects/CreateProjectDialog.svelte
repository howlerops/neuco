<script lang="ts">
	import {
		Dialog,
		DialogContent,
		DialogHeader,
		DialogTitle,
		DialogDescription,
		DialogFooter
	} from '$lib/components/ui/dialog';
	import { Button } from '$lib/components/ui/button';
	import { Input } from '$lib/components/ui/input';
	import { Label } from '$lib/components/ui/label';
	import { Badge } from '$lib/components/ui/badge';
	import {
		Select,
		SelectTrigger,
		SelectContent,
		SelectItem
	} from '$lib/components/ui/select';
	import { useCreateProject } from '$lib/api/queries/projects';
	import { trackProjectCreated } from '$lib/analytics';
	import { detectFramework, parseGitHubRepo, useGitHubRepos } from '$lib/api/queries/github';
	import type { GitHubRepo } from '$lib/api/queries/github';
	import { toast } from '$lib/components/ui/sonner';
	import type { Framework, Styling, Project } from '$lib/api/types-compat';

	interface Props {
		orgId: string;
		open?: boolean;
		onOpenChange?: (open: boolean) => void;
		onCreated?: () => void;
	}

	let {
		orgId,
		open = $bindable(false),
		onOpenChange,
		onCreated
	}: Props = $props();

	const createProject = $derived.by(() => useCreateProject(orgId));

	// ── Form state ────────────────────────────────────────────────────────────

	let name = $state('');
	let githubRepoInput = $state('');
	let framework = $state<Framework>('react');
	let styling = $state<Styling>('tailwind');

	// ── Detection state ───────────────────────────────────────────────────────

	// The normalized "owner/repo" extracted from the input
	let parsedRepo = $state<string | null>(null);

	// Whether we are currently calling the detection API
	let isDetecting = $state(false);

	// Whether auto-fill values have been applied for the current repo
	let detectionApplied = $state(false);

	// Error message if the repo URL is invalid or the repo is inaccessible
	let repoError = $state<string | null>(null);

	// Whether the user has ever edited name/framework/styling manually after
	// auto-fill — used to avoid clobbering intentional overrides on re-detection
	let nameEdited = $state(false);
	let frameworkEdited = $state(false);
	let stylingEdited = $state(false);

	// ── Repo search state ─────────────────────────────────────────────────────

	// Raw value in the search input (live, used for display)
	let repoSearchInput = $state('');

	// Debounced value actually sent to the query — updated 300 ms after typing stops
	let debouncedSearchQuery = $state('');

	// Whether the dropdown is visible
	let repoDropdownOpen = $state(false);

	// Debounce timer handle — plain variable, NOT $state (to avoid effect loops)
	let debounceTimer: ReturnType<typeof setTimeout> | null = null;

	// Track whether a search selection has been made (suppresses re-opening the dropdown)
	let selectionMade = $state(false);

	// ── Debounced search via oninput handler (not $effect to avoid loops) ─────

	function handleRepoSearchInput(value: string) {
		repoSearchInput = value;
		selectionMade = false;

		if (debounceTimer !== null) {
			clearTimeout(debounceTimer);
			debounceTimer = null;
		}

		if (value.length < 2) {
			debouncedSearchQuery = '';
			repoDropdownOpen = false;
			return;
		}

		debounceTimer = setTimeout(() => {
			debouncedSearchQuery = value;
			repoDropdownOpen = true;
			debounceTimer = null;
		}, 300);
	}

	// ── TanStack Query for repo search ────────────────────────────────────────

	const repoQuery = $derived.by(() => useGitHubRepos(debouncedSearchQuery));

	// ── Derived flags used in the template ───────────────────────────────────

	const isSearching = $derived(
		repoSearchInput.length >= 2 && repoQuery.isFetching
	);

	const repoResults = $derived(
		(repoQuery.data ?? []) as GitHubRepo[]
	);

	// ── Options ───────────────────────────────────────────────────────────────

	const frameworkOptions: { value: Framework; label: string }[] = [
		{ value: 'react', label: 'React' },
		{ value: 'nextjs', label: 'Next.js' },
		{ value: 'vue', label: 'Vue' },
		{ value: 'nuxt', label: 'Nuxt' },
		{ value: 'svelte', label: 'Svelte' },
		{ value: 'sveltekit', label: 'SvelteKit' },
		{ value: 'angular', label: 'Angular' },
		{ value: 'other', label: 'Other' }
	];

	const stylingOptions: { value: Styling; label: string }[] = [
		{ value: 'tailwind', label: 'Tailwind CSS' },
		{ value: 'css_modules', label: 'CSS Modules' },
		{ value: 'styled_components', label: 'Styled Components' },
		{ value: 'sass', label: 'Sass/SCSS' },
		{ value: 'plain_css', label: 'Plain CSS' },
		{ value: 'other', label: 'Other' }
	];

	// ── Derived display values ────────────────────────────────────────────────

	const frameworkLabel = $derived(
		frameworkOptions.find((o) => o.value === framework)?.label ?? 'Select framework'
	);
	const stylingLabel = $derived(
		stylingOptions.find((o) => o.value === styling)?.label ?? 'Select styling'
	);

	const isValid = $derived(name.trim().length >= 2);

	// ── Detection logic ───────────────────────────────────────────────────────

	async function runDetection(ownerRepo: string) {
		isDetecting = true;
		repoError = null;

		const result = await detectFramework(ownerRepo);

		isDetecting = false;

		if (!result) {
			// Detection failed — leave manual selects visible, show a hint
			repoError =
				'Could not detect framework automatically. The repository may be private or does not contain a package.json. Please select manually.';
			detectionApplied = false;
			return;
		}

		// Auto-fill fields that the user has not manually edited
		if (!nameEdited) name = result.name;
		if (!frameworkEdited) framework = result.framework;
		if (!stylingEdited) styling = result.styling;

		detectionApplied = true;
	}

	async function handleRepoBlur() {
		const parsed = parseGitHubRepo(githubRepoInput);

		// No change from the previous parsed value — do nothing
		if (parsed === parsedRepo) return;

		parsedRepo = parsed;
		detectionApplied = false;

		if (!parsed) {
			// Input was cleared or is not recognisable — reset auto-fill guard
			// flags so the next valid repo gets a clean slate
			repoError =
				githubRepoInput.trim().length > 0
					? 'Please enter a valid GitHub URL or owner/repo format'
					: null;
			return;
		}

		repoError = null;
		await runDetection(parsed);
	}

	// ── Repo search handlers ──────────────────────────────────────────────────

	function handleSearchInput() {
		// Any new typing after a previous selection re-enables the dropdown
		selectionMade = false;
	}

	function handleSearchFocus() {
		// Re-open the dropdown if there are results already available
		if (repoResults.length > 0 && repoSearchInput.length >= 2 && !selectionMade) {
			repoDropdownOpen = true;
		}
	}

	function handleSearchBlur() {
		// Delay close so a click on a dropdown item fires before we hide it
		setTimeout(() => {
			repoDropdownOpen = false;
		}, 150);
	}

	async function selectRepo(repo: GitHubRepo) {
		// Close dropdown and mark as selected so it doesn't re-open on blur/focus
		repoDropdownOpen = false;
		selectionMade = true;
		repoSearchInput = repo.fullName;
		debouncedSearchQuery = '';

		// Fill the manual URL field and trigger detection
		githubRepoInput = repo.fullName;
		parsedRepo = repo.fullName;
		detectionApplied = false;
		repoError = null;

		await runDetection(repo.fullName);
	}

	// ── Reset & lifecycle ─────────────────────────────────────────────────────

	function reset() {
		name = '';
		githubRepoInput = '';
		repoSearchInput = '';
		debouncedSearchQuery = '';
		repoDropdownOpen = false;
		selectionMade = false;
		framework = 'react';
		styling = 'tailwind';
		parsedRepo = null;
		isDetecting = false;
		detectionApplied = false;
		repoError = null;
		nameEdited = false;
		frameworkEdited = false;
		stylingEdited = false;
		if (debounceTimer !== null) {
			clearTimeout(debounceTimer);
			debounceTimer = null;
		}
	}

	function handleOpenChange(newOpen: boolean) {
		open = newOpen;
		if (!newOpen) reset();
		onOpenChange?.(newOpen);
	}

	// ── Submit ────────────────────────────────────────────────────────────────

	async function handleSubmit(e: Event) {
		e.preventDefault();
		if (!isValid) return;

		createProject.mutate(
			{
				name: name.trim(),
				githubRepo: parsedRepo ?? undefined,
				framework,
				styling
			},
			{
				onSuccess: (project: Project) => {
					trackProjectCreated(project.id, project.name);
					toast.success(`Project "${project.name}" created successfully`);
					reset();
					open = false;
					onCreated?.();
				},
				onError: (err: Error) => {
					toast.error(err.message ?? 'Failed to create project');
				}
			}
		);
	}
</script>

<Dialog bind:open {onOpenChange}>
	<DialogContent class="sm:max-w-[500px]">
		<DialogHeader>
			<DialogTitle>Create new project</DialogTitle>
			<DialogDescription>
				Search your GitHub repositories or paste a URL to auto-detect the framework.
			</DialogDescription>
		</DialogHeader>

		<form onsubmit={handleSubmit} class="space-y-5 py-2">
			<!-- ── GitHub Repo Search (step 0, prominent) ───────────────────── -->
			<div class="space-y-2">
				<Label for="repo-search">Search your GitHub repositories</Label>

				<!-- Search input with dropdown -->
				<div class="relative">
					<!-- Search icon -->
					<span
						class="pointer-events-none absolute inset-y-0 left-3 flex items-center text-muted-foreground"
						aria-hidden="true"
					>
						<svg
							xmlns="http://www.w3.org/2000/svg"
							viewBox="0 0 24 24"
							fill="none"
							stroke="currentColor"
							stroke-width="2"
							stroke-linecap="round"
							stroke-linejoin="round"
							class="h-4 w-4"
						>
							<circle cx="11" cy="11" r="8" />
							<path d="m21 21-4.35-4.35" />
						</svg>
					</span>

					<Input
						id="repo-search"
						class="pl-9 pr-9"
						placeholder="Search your repositories..."
						value={repoSearchInput}
						oninput={(e) => handleRepoSearchInput(e.currentTarget.value)}
						onfocus={handleSearchFocus}
						onblur={handleSearchBlur}
						autocomplete="off"
						autocorrect="off"
						spellcheck={false}
						role="combobox"
						aria-expanded={repoDropdownOpen}
						aria-controls="repo-search-listbox"
						aria-autocomplete="list"
					/>

					<!-- Spinner shown while searching -->
					{#if isSearching}
						<span
							class="pointer-events-none absolute inset-y-0 right-3 flex items-center"
							aria-hidden="true"
						>
							<svg
								class="h-4 w-4 animate-spin text-muted-foreground"
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
						</span>
					{/if}

					<!-- Dropdown results -->
					{#if repoDropdownOpen && repoResults.length > 0}
						<div
							id="repo-search-listbox"
							role="listbox"
							aria-label="GitHub repository results"
							class="absolute left-0 right-0 top-full z-50 mt-1 max-h-60 overflow-y-auto rounded-md border border-border bg-popover shadow-md"
						>
							{#each repoResults as repo (repo.fullName)}
								<button
									type="button"
									role="option"
									aria-selected="false"
									class="flex w-full flex-col gap-0.5 px-3 py-2.5 text-left hover:bg-accent focus:bg-accent focus:outline-none"
									onclick={() => selectRepo(repo)}
								>
									<div class="flex items-center gap-2">
										<span class="truncate text-sm font-medium text-foreground">
											{repo.name}
										</span>
										<span class="shrink-0 text-xs text-muted-foreground">
											{repo.fullName.split('/')[0]}
										</span>
										<div class="ml-auto flex shrink-0 items-center gap-1.5">
											{#if repo.language}
												<span
													class="rounded bg-secondary px-1.5 py-0.5 text-[10px] font-medium text-secondary-foreground"
												>
													{repo.language}
												</span>
											{/if}
											<span
												class="rounded border px-1.5 py-0.5 text-[10px] font-medium {repo.private
													? 'border-amber-500/40 text-amber-600 dark:text-amber-400'
													: 'border-border text-muted-foreground'}"
											>
												{repo.private ? 'Private' : 'Public'}
											</span>
										</div>
									</div>
									{#if repo.description}
										<p class="truncate text-xs text-muted-foreground">
											{repo.description}
										</p>
									{/if}
								</button>
							{/each}
						</div>
					{/if}
				</div>

				{#if repoSearchInput.length > 0 && repoSearchInput.length < 2}
					<p class="text-xs text-muted-foreground">Type at least 2 characters to search.</p>
				{:else if !repoSearchInput}
					<p class="text-xs text-muted-foreground">
						Optional. Search to auto-fill name, framework, and styling.
					</p>
				{/if}
			</div>

			<!-- ── Divider ──────────────────────────────────────────────────── -->
			<div class="flex items-center gap-3">
				<div class="h-px flex-1 bg-border"></div>
				<span class="text-xs text-muted-foreground">or paste a URL</span>
				<div class="h-px flex-1 bg-border"></div>
			</div>

			<!-- ── GitHub Repo URL (manual fallback) ────────────────────────── -->
			<div class="space-y-2">
				<Label for="github-repo">GitHub repository URL</Label>

				<!-- Input with GitHub icon prefix -->
				<div class="relative">
					<!-- GitHub mark icon -->
					<span
						class="pointer-events-none absolute inset-y-0 left-3 flex items-center text-muted-foreground"
						aria-hidden="true"
					>
						<svg
							xmlns="http://www.w3.org/2000/svg"
							viewBox="0 0 24 24"
							class="h-4 w-4 fill-current"
						>
							<path
								d="M12 0C5.37 0 0 5.37 0 12c0 5.303 3.438 9.8 8.205 11.387.6.113.82-.258.82-.577
								0-.285-.01-1.04-.015-2.04-3.338.724-4.042-1.61-4.042-1.61-.546-1.387-1.333-1.756-1.333-1.756-1.09-.745.083-.73.083-.73
								1.205.085 1.84 1.237 1.84 1.237 1.07 1.834 2.807 1.304 3.492.997.108-.775.418-1.305.76-1.605-2.665-.3-5.466-1.332-5.466-5.93
								0-1.31.465-2.38 1.235-3.22-.135-.303-.54-1.523.105-3.176 0 0 1.005-.322 3.3 1.23a11.52 11.52 0 0 1 3-.405c1.02.005 2.045.138
								3 .405 2.28-1.552 3.285-1.23 3.285-1.23.645 1.653.24 2.873.12 3.176.765.84 1.23 1.91 1.23 3.22 0 4.61-2.805 5.625-5.475
								5.92.42.36.81 1.096.81 2.22 0 1.605-.015 2.896-.015 3.286 0 .315.21.69.825.57C20.565 21.795 24 17.298 24 12c0-6.63-5.37-12-12-12z"
							/>
						</svg>
					</span>

					<Input
						id="github-repo"
						class="pl-9"
						placeholder="https://github.com/owner/repo or owner/repo"
						bind:value={githubRepoInput}
						onblur={handleRepoBlur}
						autocomplete="off"
						autocorrect="off"
						spellcheck={false}
					/>

					<!-- Spinner shown while detecting -->
					{#if isDetecting}
						<span
							class="pointer-events-none absolute inset-y-0 right-3 flex items-center"
							aria-hidden="true"
						>
							<svg
								class="h-4 w-4 animate-spin text-muted-foreground"
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
						</span>
					{/if}
				</div>

				<!-- Detection status feedback -->
				{#if isDetecting}
					<p class="text-xs text-muted-foreground">Detecting framework from repository...</p>
				{:else if repoError}
					<p class="text-xs text-destructive">{repoError}</p>
				{:else if detectionApplied && parsedRepo}
					<p class="text-xs text-muted-foreground">
						Detected from
						<span class="font-medium text-foreground">{parsedRepo}</span>. You can override any
						value below.
					</p>
				{:else if !githubRepoInput}
					<p class="text-xs text-muted-foreground">
						Optional. Paste a URL to auto-fill name, framework, and styling.
					</p>
				{/if}
			</div>

			<!-- ── Project name ────────────────────────────────────────────── -->
			<div class="space-y-2">
				<Label for="project-name">
					Project name <span class="text-destructive">*</span>
				</Label>
				<Input
					id="project-name"
					placeholder="My Awesome App"
					bind:value={name}
					oninput={() => {
						nameEdited = true;
					}}
					required
					minlength={2}
					maxlength={100}
					autocomplete="off"
				/>
			</div>

			<!-- ── Framework & Styling ─────────────────────────────────────── -->
			<div class="grid grid-cols-2 gap-4">
				<!-- Framework -->
				<div class="space-y-2">
					<div class="flex items-center gap-2">
						<Label>Framework <span class="text-destructive">*</span></Label>
						{#if detectionApplied && !frameworkEdited}
							<Badge variant="secondary" class="px-1.5 py-0 text-[10px]">auto</Badge>
						{/if}
					</div>
					<Select
						value={framework}
						onValueChange={(v) => {
							framework = v as Framework;
							frameworkEdited = true;
						}}
						disabled={isDetecting}
					>
						<SelectTrigger>
							<span>{frameworkLabel}</span>
						</SelectTrigger>
						<SelectContent>
							{#each frameworkOptions as opt (opt.value)}
								<SelectItem value={opt.value} label={opt.label} />
							{/each}
						</SelectContent>
					</Select>
				</div>

				<!-- Styling -->
				<div class="space-y-2">
					<div class="flex items-center gap-2">
						<Label>Styling <span class="text-destructive">*</span></Label>
						{#if detectionApplied && !stylingEdited}
							<Badge variant="secondary" class="px-1.5 py-0 text-[10px]">auto</Badge>
						{/if}
					</div>
					<Select
						value={styling}
						onValueChange={(v) => {
							styling = v as Styling;
							stylingEdited = true;
						}}
						disabled={isDetecting}
					>
						<SelectTrigger>
							<span>{stylingLabel}</span>
						</SelectTrigger>
						<SelectContent>
							{#each stylingOptions as opt (opt.value)}
								<SelectItem value={opt.value} label={opt.label} />
							{/each}
						</SelectContent>
					</Select>
				</div>
			</div>

			<!-- ── Footer actions ──────────────────────────────────────────── -->
			<DialogFooter class="pt-2">
				<Button
					type="button"
					variant="outline"
					onclick={() => handleOpenChange(false)}
					disabled={createProject.isPending}
				>
					Cancel
				</Button>
				<Button type="submit" disabled={!isValid || createProject.isPending || isDetecting}>
					{#if createProject.isPending}
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
						Creating...
					{:else}
						Create project
					{/if}
				</Button>
			</DialogFooter>
		</form>
	</DialogContent>
</Dialog>
