<script lang="ts">
	import { page } from '$app/stores';
	import { ApiError } from '$lib/api/client';
	import {
		useAgentConfig,
		useAgentProviders,
		useUpsertAgentConfig,
		useDeleteAgentConfig,
		useValidateAgentConfig
	} from '$lib/api/queries/agent-config';
	import type { AgentProviderName } from '$lib/api/types-compat';
	import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '$lib/components/ui/card';
	import { Button } from '$lib/components/ui/button';
	import { Input } from '$lib/components/ui/input';
	import { Label } from '$lib/components/ui/label';
	import {
		Select,
		SelectContent,
		SelectItem,
		SelectTrigger
	} from '$lib/components/ui/select';
	import { Skeleton } from '$lib/components/ui/skeleton';
	import { Alert, AlertTitle, AlertDescription } from '$lib/components/ui/alert';
	import { Separator } from '$lib/components/ui/separator';
	import { toast } from 'svelte-sonner';
	import {
		AlertCircle,
		RefreshCw,
		Save,
		Loader2,
		Eye,
		EyeOff,
		PlugZap,
		Trash2,
		ShieldCheck,
		FlaskConical
	} from 'lucide-svelte';

	const orgSlug = $derived($page.params.orgSlug ?? '');
	const projectId = $derived($page.params.id ?? '');

	const configQuery = $derived.by(() => useAgentConfig(projectId));
	const providersQuery = $derived.by(() => useAgentProviders());
	const upsertMutation = $derived.by(() => useUpsertAgentConfig(projectId));
	const deleteMutation = $derived.by(() => useDeleteAgentConfig(projectId));
	const validateMutation = $derived.by(() => useValidateAgentConfig(projectId));

	const configNotFound = $derived(
		configQuery.isError && configQuery.error instanceof ApiError && configQuery.error.status === 404
	);
	const configLoadError = $derived(
		configQuery.isError && !configNotFound ? configQuery.error : null
	);

	let provider = $state<AgentProviderName | ''>('');
	let apiKey = $state('');
	let modelOverride = $state('');
	let showApiKey = $state(false);

	let initialProvider = $state<AgentProviderName | ''>('');
	let initialModelOverride = $state('');

	const existingConfig = $derived(configNotFound ? null : (configQuery.data ?? null));

	const selectedProvider = $derived(
		(providersQuery.data ?? []).find((p) => p.name === provider)
	);

	const providerLabel = $derived(
		selectedProvider?.displayName ?? (provider ? provider : 'Select a provider')
	);

	const isDirty = $derived(
		provider !== initialProvider || modelOverride.trim() !== initialModelOverride || apiKey.trim().length > 0
	);

	$effect(() => {
		if (existingConfig) {
			provider = existingConfig.provider;
			modelOverride = existingConfig.modelOverride ?? '';
			apiKey = '';
			initialProvider = existingConfig.provider;
			initialModelOverride = existingConfig.modelOverride ?? '';
		}
	});

	$effect(() => {
		if (!existingConfig && configNotFound) {
			initialProvider = '';
			initialModelOverride = '';
			if (!provider && providersQuery.data?.length) {
				provider = (providersQuery.data.find((p) => p.installed)?.name ?? providersQuery.data[0].name) as AgentProviderName;
			}
		}
	});

	function errorMessage(err: unknown): string {
		if (err instanceof Error) return err.message;
		return 'An unexpected error occurred.';
	}

	async function handleTestConnection() {
		if (!provider) {
			toast.error('Please choose a provider before testing');
			return;
		}

		try {
			const result = await validateMutation.mutateAsync({
				provider,
				apiKey: apiKey.trim() || undefined,
				modelOverride: modelOverride.trim() || undefined
			});

			if (result.valid) {
				toast.success('Connection test successful');
			} else {
				toast.error('Connection test failed', {
					description: result.error || 'Validation did not pass.'
				});
			}
		} catch (err) {
			toast.error('Failed to validate configuration', {
				description: errorMessage(err)
			});
		}
	}

	async function handleSave() {
		if (!provider) {
			toast.error('Please select a provider');
			return;
		}

		try {
			await upsertMutation.mutateAsync({
				provider,
				apiKey: apiKey.trim() || undefined,
				modelOverride: modelOverride.trim() || undefined
			});

			initialProvider = provider;
			initialModelOverride = modelOverride.trim();
			apiKey = '';
			await configQuery.refetch();
			toast.success(existingConfig ? 'Agent configuration updated' : 'Agent configuration saved');
		} catch (err) {
			toast.error('Failed to save configuration', {
				description: errorMessage(err)
			});
		}
	}

	async function handleDelete() {
		if (!existingConfig) return;

		try {
			await deleteMutation.mutateAsync({ provider: existingConfig.provider });
			apiKey = '';
			modelOverride = '';
			initialProvider = '';
			initialModelOverride = '';
			await configQuery.refetch();
			toast.success('Agent configuration deleted');
		} catch (err) {
			toast.error('Failed to delete configuration', {
				description: errorMessage(err)
			});
		}
	}
</script>

<svelte:head>
	<title>Agent Settings — {orgSlug} — Neuco</title>
</svelte:head>

<div class="py-6 space-y-8 max-w-2xl">
	{#if configQuery.isLoading || providersQuery.isLoading}
		<Card>
			<CardHeader>
				<Skeleton class="h-5 w-44"></Skeleton>
				<Skeleton class="h-4 w-72 mt-1"></Skeleton>
			</CardHeader>
			<CardContent class="space-y-5">
				<div class="space-y-2">
					<Skeleton class="h-4 w-24"></Skeleton>
					<Skeleton class="h-10 w-full"></Skeleton>
				</div>
				<div class="space-y-2">
					<Skeleton class="h-4 w-20"></Skeleton>
					<Skeleton class="h-10 w-full"></Skeleton>
				</div>
				<div class="space-y-2">
					<Skeleton class="h-4 w-32"></Skeleton>
					<Skeleton class="h-10 w-full"></Skeleton>
				</div>
				<div class="flex gap-2 pt-2">
					<Skeleton class="h-9 w-28"></Skeleton>
					<Skeleton class="h-9 w-20"></Skeleton>
				</div>
			</CardContent>
		</Card>
	{:else if providersQuery.isError}
		<Alert variant="destructive">
			<AlertCircle class="h-4 w-4" />
			<AlertTitle>Failed to load agent providers</AlertTitle>
			<AlertDescription class="flex items-center justify-between">
				<span>{providersQuery.error?.message ?? 'An unexpected error occurred.'}</span>
				<Button
					variant="outline"
					size="sm"
					onclick={() => providersQuery.refetch()}
					class="ml-4 shrink-0"
				>
					<RefreshCw class="mr-1.5 h-3 w-3" />
					Retry
				</Button>
			</AlertDescription>
		</Alert>
	{:else if configLoadError}
		<Alert variant="destructive">
			<AlertCircle class="h-4 w-4" />
			<AlertTitle>Failed to load current configuration</AlertTitle>
			<AlertDescription class="flex items-center justify-between">
				<span>{configLoadError?.message ?? 'An unexpected error occurred.'}</span>
				<Button
					variant="outline"
					size="sm"
					onclick={() => configQuery.refetch()}
					class="ml-4 shrink-0"
				>
					<RefreshCw class="mr-1.5 h-3 w-3" />
					Retry
				</Button>
			</AlertDescription>
		</Alert>
	{:else}
		<Card>
			<CardHeader>
				<CardTitle class="flex items-center gap-2">
					<PlugZap class="h-4 w-4" />
					Agent Configuration
				</CardTitle>
				<CardDescription>
					Configure your project agent provider, credentials, and optional model override.
				</CardDescription>
			</CardHeader>
			<CardContent class="space-y-5">
				{#if !existingConfig}
					<div class="rounded-md border border-border bg-muted/30 px-3 py-2 text-xs text-muted-foreground">
						No agent config exists for this project yet. Create one to override organization defaults.
					</div>
				{/if}

				<div class="space-y-2">
					<Label for="provider">Provider</Label>
					<Select
						value={provider}
						onValueChange={(value) => {
							provider = value as AgentProviderName;
						}}
						disabled={upsertMutation.isPending || deleteMutation.isPending}
					>
						<SelectTrigger placeholder="Select provider">
							<span>{providerLabel}</span>
						</SelectTrigger>
						<SelectContent>
							{#each providersQuery.data ?? [] as p (p.name)}
								<SelectItem value={p.name} label={p.displayName}>
									<div class="flex items-center justify-between w-full gap-3">
										<span>{p.displayName}</span>
										{#if p.installed}
											<span class="text-xs text-emerald-600 dark:text-emerald-400">Installed</span>
										{/if}
									</div>
								</SelectItem>
							{/each}
						</SelectContent>
					</Select>
				</div>

				{#if selectedProvider}
					<div class="rounded-md border border-border p-3 space-y-2">
						<div class="flex items-center gap-2 text-sm font-medium">
							{#if selectedProvider.installed}
								<ShieldCheck class="h-4 w-4 text-emerald-600 dark:text-emerald-400" />
							{:else}
								<AlertCircle class="h-4 w-4 text-amber-600 dark:text-amber-400" />
							{/if}
							{selectedProvider.displayName} setup
						</div>
						<p class="text-xs text-muted-foreground whitespace-pre-wrap">
							{selectedProvider.installInstructions}
						</p>
					</div>
				{/if}

				<Separator />

				<div class="space-y-2">
					<Label for="api-key">API Key</Label>
					<div class="flex items-center gap-2">
						<Input
							id="api-key"
							type={showApiKey ? 'text' : 'password'}
							value={apiKey}
							oninput={(e) => (apiKey = (e.target as HTMLInputElement).value)}
							placeholder={existingConfig?.hasApiKey ? '•••••••••••• (leave empty to keep existing key)' : 'Enter API key'}
							disabled={upsertMutation.isPending || deleteMutation.isPending}
							class="flex-1"
						/>
						<Button
							type="button"
							variant="outline"
							size="icon"
							onclick={() => (showApiKey = !showApiKey)}
							class="shrink-0"
						>
							{#if showApiKey}
								<EyeOff class="h-4 w-4" />
							{:else}
								<Eye class="h-4 w-4" />
							{/if}
						</Button>
					</div>
					<p class="text-xs text-muted-foreground">
						{existingConfig?.hasApiKey
							? 'A key is already stored for this config. Provide a new key only if you want to rotate it.'
							: 'Your key is stored securely and never shown after saving.'}
					</p>
				</div>

				<div class="space-y-2">
					<Label for="model-override">Model Override <span class="text-muted-foreground">(optional)</span></Label>
					<Input
						id="model-override"
						value={modelOverride}
						oninput={(e) => (modelOverride = (e.target as HTMLInputElement).value)}
						placeholder="e.g. claude-3-7-sonnet"
						disabled={upsertMutation.isPending || deleteMutation.isPending}
					/>
				</div>

				<div class="flex flex-wrap items-center gap-2 pt-2">
					<Button
						variant="outline"
						onclick={handleTestConnection}
						disabled={!provider || validateMutation.isPending || upsertMutation.isPending || deleteMutation.isPending}
					>
						{#if validateMutation.isPending}
							<Loader2 class="mr-1.5 h-3.5 w-3.5 animate-spin" />
							Testing...
						{:else}
							<FlaskConical class="mr-1.5 h-3.5 w-3.5" />
							Test Connection
						{/if}
					</Button>

					<Button
						onclick={handleSave}
						disabled={!provider || !isDirty || upsertMutation.isPending || deleteMutation.isPending}
					>
						{#if upsertMutation.isPending}
							<Loader2 class="mr-1.5 h-3.5 w-3.5 animate-spin" />
							Saving...
						{:else}
							<Save class="mr-1.5 h-3.5 w-3.5" />
							Save
						{/if}
					</Button>

					{#if existingConfig}
						<Button
							variant="destructive"
							onclick={handleDelete}
							disabled={deleteMutation.isPending || upsertMutation.isPending}
						>
							{#if deleteMutation.isPending}
								<Loader2 class="mr-1.5 h-3.5 w-3.5 animate-spin" />
								Deleting...
							{:else}
								<Trash2 class="mr-1.5 h-3.5 w-3.5" />
								Delete
							{/if}
						</Button>
					{/if}
				</div>
			</CardContent>
		</Card>
	{/if}
</div>
