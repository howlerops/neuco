<script lang="ts">
	import { page } from '$app/stores';
	import {
		useNangoConnections,
		useCreateNangoConnection,
		useDeleteNangoConnection,
		useSyncConnection,
		type NangoConnection
	} from '$lib/api/queries/nango';
	import {
		useCreateIntegration,
		useIntegrations,
		useIntercomDisconnect,
		useIntercomSync,
		useSlackDisconnect,
		useSlackSync,
		useLinearDisconnect,
		useLinearSync,
		useJiraDisconnect,
		useJiraSync,
		type ExtendedIntegrationProvider,
		type IntercomAuthorizeResponse,
		type SlackAuthorizeResponse,
		type LinearAuthorizeResponse,
		type JiraAuthorizeResponse,
		type IntegrationRecord
	} from '$lib/api/queries/integrations';
	import { apiClient } from '$lib/api/client';
	import { createNangoSession } from '$lib/nango';
	import { Card, CardContent, CardHeader, CardFooter } from '$lib/components/ui/card';
	import { Button } from '$lib/components/ui/button';
	import { Skeleton } from '$lib/components/ui/skeleton';
	import { Alert, AlertTitle, AlertDescription } from '$lib/components/ui/alert';
	import {
		Dialog,
		DialogContent,
		DialogHeader,
		DialogTitle,
		DialogDescription,
		DialogFooter
	} from '$lib/components/ui/dialog';
	import { toast } from 'svelte-sonner';
	import {
		Plus,
		AlertCircle,
		RefreshCw,
		Puzzle,
		Trash2,
		Copy,
		Check,
		Loader2,
		ChevronRight,
		Webhook,
		Globe,
		ArrowRight,
		CheckCircle2,
		RotateCw
	} from 'lucide-svelte';
	import IntegrationIcon from '$lib/components/icons/integrations.svelte';
	import type { IntegrationIconName } from '$lib/components/icons/integrations.svelte';
	import { cn } from '$lib/utils';

	const orgSlug = $derived($page.params.orgSlug ?? '');
	const projectId = $derived($page.params.id ?? '');

	// ── Unified connection type ───────────────────────────────────────────────
	interface UnifiedConnection {
		id: string;
		projectId: string;
		providerConfigKey: string;
		connectionId: string;
		createdAt: string;
		lastSyncAt?: string | null;
		/** 'nango' for Nango-managed, 'native' for native integrations */
		source: 'nango' | 'native';
	}

	// ── Queries & mutations ────────────────────────────────────────────────────
	const nangoConnectionsQuery = $derived.by(() => useNangoConnections(projectId));
	const nativeIntegrationsQuery = $derived.by(() => useIntegrations(projectId));
	const createConnectionMutation = $derived.by(() => useCreateNangoConnection(projectId));
	const deleteConnectionMutation = $derived.by(() => useDeleteNangoConnection(projectId));
	const syncMutation = $derived.by(() => useSyncConnection(projectId));

	// Native Intercom mutations
	const intercomDisconnectMutation = $derived.by(() => useIntercomDisconnect(projectId));
	const intercomSyncMutation = $derived.by(() => useIntercomSync(projectId));

	// Native Slack mutations
	const slackDisconnectMutation = $derived.by(() => useSlackDisconnect(projectId));
	const slackSyncMutation = $derived.by(() => useSlackSync(projectId));

	// Native Linear mutations
	const linearDisconnectMutation = $derived.by(() => useLinearDisconnect(projectId));
	const linearSyncMutation = $derived.by(() => useLinearSync(projectId));

	// Native Jira mutations
	const jiraDisconnectMutation = $derived.by(() => useJiraDisconnect(projectId));
	const jiraSyncMutation = $derived.by(() => useJiraSync(projectId));

	// Webhook integration mutation (Custom Webhook only — uses existing Make backend)
	const createWebhookMutation = $derived.by(() => useCreateIntegration(projectId));

	// ── Merged connections list ───────────────────────────────────────────────
	const connectionsQuery = $derived.by(() => {
		const nangoLoading = nangoConnectionsQuery.isLoading;
		const nativeLoading = nativeIntegrationsQuery.isLoading;
		const isLoading = nangoLoading || nativeLoading;
		const isError = nangoConnectionsQuery.isError || nativeIntegrationsQuery.isError;
		const error = nangoConnectionsQuery.error || nativeIntegrationsQuery.error;

		const nangoConns: UnifiedConnection[] = (nangoConnectionsQuery.data ?? []).map((c) => ({
			id: c.id,
			projectId: c.projectId,
			providerConfigKey: c.providerConfigKey,
			connectionId: c.connectionId,
			createdAt: c.createdAt,
			lastSyncAt: c.lastSyncAt,
			source: 'nango' as const
		}));

		const nativeConns: UnifiedConnection[] = (nativeIntegrationsQuery.data ?? [])
			.filter((i) => i.provider !== 'webhook') // webhooks are shown via Nango query
			.map((i) => ({
				id: i.id,
				projectId: i.projectId,
				providerConfigKey: i.provider,
				connectionId: i.id, // use integration ID as connection ID
				createdAt: i.createdAt,
				lastSyncAt: i.lastSyncAt ?? null,
				source: 'native' as const
			}));

		const data = [...nangoConns, ...nativeConns];

		return {
			isLoading,
			isError,
			error,
			data: isLoading ? undefined : data,
			refetch: () => {
				nangoConnectionsQuery.refetch();
				nativeIntegrationsQuery.refetch();
			}
		};
	});

	// ── Dialog state ──────────────────────────────────────────────────────────
	let addDialogOpen = $state(false);

	// 'choose' → pick provider
	// 'connecting' → OAuth popup in progress
	// 'success' → OAuth done, connection saved
	// 'webhook' → Custom Webhook URL display
	type DialogStep = 'choose' | 'connecting' | 'success' | 'webhook';
	let dialogStep = $state<DialogStep>('choose');

	let selectedProvider = $state<ExtendedIntegrationProvider | null>(null);
	let connectedProviderLabel = $state('');

	// Custom Webhook state
	let webhookUrl = $state('');
	let webhookUrlCopied = $state(false);

	// Delete confirm
	let deleteConfirmId = $state<string | null>(null);

	// Per-card sync tracking (connectionId → pending)
	let syncingIds = $state<Set<string>>(new Set());

	// ── Provider catalogue ────────────────────────────────────────────────────
	interface ProviderDef {
		value: ExtendedIntegrationProvider;
		label: string;
		description: string;
		color: string;
		iconBg: string;
		/** Nango providerConfigKey. null = Custom Webhook or native OAuth. */
		nangoKey: string | null;
		/** True if this provider uses native OAuth (not Nango). */
		nativeOAuth?: boolean;
	}

	const PROVIDERS: ProviderDef[] = [
		{
			value: 'gong',
			label: 'Gong',
			description: 'Import call transcripts and meeting recordings',
			color: 'text-indigo-600 dark:text-indigo-400',
			iconBg: 'bg-indigo-50 dark:bg-indigo-950/50',
			nangoKey: 'gong'
		},
		{
			value: 'intercom',
			label: 'Intercom',
			description: 'Import support conversations and tickets',
			color: 'text-blue-600 dark:text-blue-400',
			iconBg: 'bg-blue-50 dark:bg-blue-950/50',
			nangoKey: null,
			nativeOAuth: true
		},
		{
			value: 'slack',
			label: 'Slack',
			description: 'Import messages from selected channels',
			color: 'text-pink-600 dark:text-pink-400',
			iconBg: 'bg-pink-50 dark:bg-pink-950/50',
			nangoKey: null,
			nativeOAuth: true
		},
		{
			value: 'hubspot',
			label: 'HubSpot',
			description: 'Import deal notes and contact activity',
			color: 'text-orange-600 dark:text-orange-400',
			iconBg: 'bg-orange-50 dark:bg-orange-950/50',
			nangoKey: 'hubspot'
		},
		{
			value: 'linear',
			label: 'Linear',
			description: 'Import issues and comments',
			color: 'text-violet-600 dark:text-violet-400',
			iconBg: 'bg-violet-50 dark:bg-violet-950/50',
			nangoKey: null,
			nativeOAuth: true
		},
		{
			value: 'jira',
			label: 'Jira',
			description: 'Import tickets and comments',
			color: 'text-cyan-600 dark:text-cyan-400',
			iconBg: 'bg-cyan-50 dark:bg-cyan-950/50',
			nangoKey: null,
			nativeOAuth: true
		},
		{
			value: 'notion',
			label: 'Notion',
			description: 'Import selected pages and databases',
			color: 'text-gray-700 dark:text-gray-300',
			iconBg: 'bg-gray-100 dark:bg-gray-800',
			nangoKey: 'notion'
		},
		{
			value: 'webhook',
			label: 'Custom Webhook',
			description: 'Send signals from any source via HTTP',
			color: 'text-emerald-600 dark:text-emerald-400',
			iconBg: 'bg-emerald-50 dark:bg-emerald-950/50',
			nangoKey: null
		}
	];

	// ── Badge colours ─────────────────────────────────────────────────────────
	const BADGE_COLORS: Record<string, string> = {
		gong: 'bg-indigo-100 text-indigo-800 dark:bg-indigo-900/30 dark:text-indigo-400',
		intercom: 'bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-400',
		linear: 'bg-violet-100 text-violet-800 dark:bg-violet-900/30 dark:text-violet-400',
		jira: 'bg-cyan-100 text-cyan-800 dark:bg-cyan-900/30 dark:text-cyan-400',
		hubspot: 'bg-orange-100 text-orange-800 dark:bg-orange-900/30 dark:text-orange-400',
		notion: 'bg-gray-100 text-gray-800 dark:bg-gray-800 dark:text-gray-300',
		slack: 'bg-pink-100 text-pink-800 dark:bg-pink-900/30 dark:text-pink-400',
		webhook: 'bg-emerald-100 text-emerald-800 dark:bg-emerald-900/30 dark:text-emerald-400',
		github: 'bg-neutral-100 text-neutral-800 dark:bg-neutral-900/30 dark:text-neutral-300'
	};

	// ── Helpers ───────────────────────────────────────────────────────────────
	function providerDef(value: string): ProviderDef | undefined {
		return PROVIDERS.find((p) => p.value === value);
	}

	function providerLabel(value: string): string {
		return providerDef(value)?.label ?? value;
	}

	function badgeColor(kind: string): string {
		return BADGE_COLORS[kind] ?? 'bg-muted text-muted-foreground';
	}

	function formatDate(dateStr: string | null | undefined): string {
		if (!dateStr) return 'Never';
		return new Date(dateStr).toLocaleDateString('en-US', {
			month: 'short',
			day: 'numeric',
			year: 'numeric'
		});
	}

	function maskedWebhookUrl(pid: string): string {
		const apiBase = import.meta.env.VITE_API_BASE_URL ?? 'http://localhost:8080';
		return `${apiBase}/api/v1/webhooks/${pid}/••••••••`;
	}

	// ── Dialog controls ───────────────────────────────────────────────────────
	function openAddDialog() {
		selectedProvider = null;
		dialogStep = 'choose';
		webhookUrl = '';
		webhookUrlCopied = false;
		connectedProviderLabel = '';
		addDialogOpen = true;
	}

	function closeAddDialog() {
		addDialogOpen = false;
	}

	// ── OAuth flow (Nango) ────────────────────────────────────────────────────
	async function handleConnectOAuth() {
		if (!selectedProvider) return;

		const def = providerDef(selectedProvider);
		if (!def || !def.nangoKey) return;

		const providerConfigKey = def.nangoKey;

		dialogStep = 'connecting';

		try {
			// Create a short-lived Nango Connect session and open OAuth popup
			const nango = await createNangoSession();
			const result = await nango.auth(providerConfigKey);

			// Use the connectionId returned by Nango (auto-generated in connect session flow)
			const connectionId = result.connectionId;

			// OAuth succeeded — tell the Neuco backend about the connection
			await createConnectionMutation.mutateAsync({ providerConfigKey, connectionId });

			connectedProviderLabel = def.label;
			dialogStep = 'success';
		} catch (err) {
			// Nango auth cancelled or backend save failed — return to choose step
			dialogStep = 'choose';
			const message = err instanceof Error ? err.message : 'An unexpected error occurred.';
			// User-cancelled OAuth (Nango throws with specific message) — silent
			if (!message.toLowerCase().includes('cancel') && !message.toLowerCase().includes('closed')) {
				toast.error(`Failed to connect ${def.label}`, { description: message });
			}
		}
	}

	// ── Native Intercom OAuth flow ───────────────────────────────────────────
	async function handleConnectIntercom() {
		dialogStep = 'connecting';

		try {
			// Fetch the authorize URL from our backend.
			const authData = await apiClient.get<IntercomAuthorizeResponse>(
				`/api/v1/projects/${projectId}/intercom/authorize`
			);

			// Open Intercom OAuth in a popup.
			const popup = window.open(
				authData.authorize_url,
				'intercom-oauth',
				'width=600,height=700,scrollbars=yes'
			);

			// Poll for the popup closing (the callback page auto-closes on success).
			await new Promise<void>((resolve, reject) => {
				const interval = setInterval(() => {
					if (!popup || popup.closed) {
						clearInterval(interval);
						resolve();
					}
				}, 500);
				// Timeout after 5 minutes.
				setTimeout(() => {
					clearInterval(interval);
					reject(new Error('Authorization timed out'));
				}, 5 * 60 * 1000);
			});

			connectedProviderLabel = 'Intercom';
			dialogStep = 'success';
		} catch (err) {
			dialogStep = 'choose';
			const message = err instanceof Error ? err.message : 'An unexpected error occurred.';
			if (!message.toLowerCase().includes('cancel') && !message.toLowerCase().includes('closed')) {
				toast.error('Failed to connect Intercom', { description: message });
			}
		}
	}

	// ── Native Slack OAuth flow ──────────────────────────────────────────────
	async function handleConnectSlack() {
		dialogStep = 'connecting';

		try {
			// Fetch the authorize URL from our backend.
			const authData = await apiClient.get<SlackAuthorizeResponse>(
				`/api/v1/projects/${projectId}/slack/authorize`
			);

			// Open Slack OAuth in a popup.
			const popup = window.open(
				authData.authorize_url,
				'slack-oauth',
				'width=600,height=700,scrollbars=yes'
			);

			// Poll for the popup closing (the callback page auto-closes on success).
			await new Promise<void>((resolve, reject) => {
				const interval = setInterval(() => {
					if (!popup || popup.closed) {
						clearInterval(interval);
						resolve();
					}
				}, 500);
				// Timeout after 5 minutes.
				setTimeout(() => {
					clearInterval(interval);
					reject(new Error('Authorization timed out'));
				}, 5 * 60 * 1000);
			});

			connectedProviderLabel = 'Slack';
			dialogStep = 'success';
		} catch (err) {
			dialogStep = 'choose';
			const message = err instanceof Error ? err.message : 'An unexpected error occurred.';
			if (!message.toLowerCase().includes('cancel') && !message.toLowerCase().includes('closed')) {
				toast.error('Failed to connect Slack', { description: message });
			}
		}
	}

	// ── Native Jira OAuth flow ─────────────────────────────────────────────
	async function handleConnectJira() {
		dialogStep = 'connecting';

		try {
			// Fetch the authorize URL from our backend.
			const authData = await apiClient.get<JiraAuthorizeResponse>(
				`/api/v1/projects/${projectId}/jira/authorize`
			);

			// Open Jira OAuth in a popup.
			const popup = window.open(
				authData.authorize_url,
				'jira-oauth',
				'width=600,height=700,scrollbars=yes'
			);

			// Poll for the popup closing (the callback page auto-closes on success).
			await new Promise<void>((resolve, reject) => {
				const interval = setInterval(() => {
					if (!popup || popup.closed) {
						clearInterval(interval);
						resolve();
					}
				}, 500);
				// Timeout after 5 minutes.
				setTimeout(() => {
					clearInterval(interval);
					reject(new Error('Authorization timed out'));
				}, 5 * 60 * 1000);
			});

			connectedProviderLabel = 'Jira';
			dialogStep = 'success';
		} catch (err) {
			dialogStep = 'choose';
			const message = err instanceof Error ? err.message : 'An unexpected error occurred.';
			if (!message.toLowerCase().includes('cancel') && !message.toLowerCase().includes('closed')) {
				toast.error('Failed to connect Jira', { description: message });
			}
		}
	}

	// ── Native Linear OAuth flow ────────────────────────────────────────────
	async function handleConnectLinear() {
		dialogStep = 'connecting';

		try {
			// Fetch the authorize URL from our backend.
			const authData = await apiClient.get<LinearAuthorizeResponse>(
				`/api/v1/projects/${projectId}/linear/authorize`
			);

			// Open Linear OAuth in a popup.
			const popup = window.open(
				authData.authorize_url,
				'linear-oauth',
				'width=600,height=700,scrollbars=yes'
			);

			// Poll for the popup closing (the callback page auto-closes on success).
			await new Promise<void>((resolve, reject) => {
				const interval = setInterval(() => {
					if (!popup || popup.closed) {
						clearInterval(interval);
						resolve();
					}
				}, 500);
				// Timeout after 5 minutes.
				setTimeout(() => {
					clearInterval(interval);
					reject(new Error('Authorization timed out'));
				}, 5 * 60 * 1000);
			});

			connectedProviderLabel = 'Linear';
			dialogStep = 'success';
		} catch (err) {
			dialogStep = 'choose';
			const message = err instanceof Error ? err.message : 'An unexpected error occurred.';
			if (!message.toLowerCase().includes('cancel') && !message.toLowerCase().includes('closed')) {
				toast.error('Failed to connect Linear', { description: message });
			}
		}
	}

	// ── Custom Webhook flow ───────────────────────────────────────────────────
	async function handleConnectWebhook() {
		try {
			const result = await createWebhookMutation.mutateAsync({
				provider: 'webhook',
				config: {}
			});

			const apiBase = import.meta.env.VITE_API_BASE_URL ?? 'http://localhost:8080';
			webhookUrl = `${apiBase}/api/v1/webhooks/${projectId}/${result.webhookSecret}`;
			webhookUrlCopied = false;
			dialogStep = 'webhook';
		} catch (err) {
			toast.error('Failed to create webhook', {
				description: err instanceof Error ? err.message : 'An unexpected error occurred.'
			});
		}
	}

	async function handleNext() {
		if (!selectedProvider) return;
		const def = providerDef(selectedProvider);
		if (!def) return;

		if (def.value === 'slack' && def.nativeOAuth) {
			await handleConnectSlack();
		} else if (def.value === 'linear' && def.nativeOAuth) {
			await handleConnectLinear();
		} else if (def.value === 'jira' && def.nativeOAuth) {
			await handleConnectJira();
		} else if (def.nativeOAuth) {
			await handleConnectIntercom();
		} else if (def.nangoKey !== null) {
			await handleConnectOAuth();
		} else {
			await handleConnectWebhook();
		}
	}

	async function copyWebhookUrl() {
		try {
			await navigator.clipboard.writeText(webhookUrl);
			webhookUrlCopied = true;
			setTimeout(() => {
				webhookUrlCopied = false;
			}, 2500);
		} catch {
			toast.error('Failed to copy to clipboard');
		}
	}

	// ── Disconnect ────────────────────────────────────────────────────────────
	async function handleDisconnect(connection: UnifiedConnection) {
		try {
			if (connection.source === 'native' && connection.providerConfigKey === 'slack') {
				await slackDisconnectMutation.mutateAsync(connection.id);
			} else if (connection.source === 'native' && connection.providerConfigKey === 'linear') {
				await linearDisconnectMutation.mutateAsync(connection.id);
			} else if (connection.source === 'native' && connection.providerConfigKey === 'jira') {
				await jiraDisconnectMutation.mutateAsync(connection.id);
			} else if (connection.source === 'native') {
				await intercomDisconnectMutation.mutateAsync(connection.id);
			} else {
				await deleteConnectionMutation.mutateAsync(connection.connectionId);
			}
			toast.success(`${providerLabel(connection.providerConfigKey)} disconnected`);
			deleteConfirmId = null;
		} catch (err) {
			toast.error('Failed to disconnect', {
				description: err instanceof Error ? err.message : 'An unexpected error occurred.'
			});
		}
	}

	// ── Sync now ──────────────────────────────────────────────────────────────
	async function handleSync(connection: UnifiedConnection) {
		if (syncingIds.has(connection.connectionId)) return;

		syncingIds = new Set([...syncingIds, connection.connectionId]);
		try {
			if (connection.source === 'native' && connection.providerConfigKey === 'slack') {
				await slackSyncMutation.mutateAsync(connection.id);
			} else if (connection.source === 'native' && connection.providerConfigKey === 'linear') {
				await linearSyncMutation.mutateAsync(connection.id);
			} else if (connection.source === 'native' && connection.providerConfigKey === 'jira') {
				await jiraSyncMutation.mutateAsync(connection.id);
			} else if (connection.source === 'native') {
				await intercomSyncMutation.mutateAsync(connection.id);
			} else {
				await syncMutation.mutateAsync(connection.connectionId);
			}
			toast.success(`${providerLabel(connection.providerConfigKey)} sync triggered`);
		} catch (err) {
			toast.error('Failed to trigger sync', {
				description: err instanceof Error ? err.message : 'An unexpected error occurred.'
			});
		} finally {
			const next = new Set(syncingIds);
			next.delete(connection.connectionId);
			syncingIds = next;
		}
	}

	// ── Derived ───────────────────────────────────────────────────────────────
	const currentProviderDef = $derived(selectedProvider ? providerDef(selectedProvider) : null);

	const isOAuthProvider = $derived(
		currentProviderDef !== null && currentProviderDef !== undefined && (currentProviderDef.nangoKey !== null || currentProviderDef.nativeOAuth === true)
	);

	const isBusy = $derived(
		createConnectionMutation.isPending || createWebhookMutation.isPending
	);

	// Step labels shown in the breadcrumb
	const STEP_LABELS: Record<DialogStep, string> = {
		choose: 'Choose Provider',
		connecting: 'Connecting…',
		success: 'Connected',
		webhook: 'Webhook URL'
	};
</script>

<svelte:head>
	<title>Integrations — Neuco</title>
</svelte:head>

<div class="p-6 space-y-6">
	<!-- Page header -->
	<div class="flex items-center justify-between">
		<div>
			<h1 class="text-2xl font-bold tracking-tight">Integrations</h1>
			<p class="text-muted-foreground mt-1 text-sm">
				Connect external tools to automatically sync data into Neuco
			</p>
		</div>
		<Button onclick={openAddDialog} class="gap-2">
			<Plus class="h-4 w-4" />
			Add Integration
		</Button>
	</div>

	<!-- Connection grid -->
	{#if connectionsQuery.isLoading}
		<div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
			{#each Array(3) as _, i (i)}
				<Card>
					<CardHeader class="pb-3">
						<div class="flex items-start justify-between gap-2">
							<div class="space-y-2">
								<Skeleton class="h-5 w-32" />
								<Skeleton class="h-4 w-20 rounded-full" />
							</div>
							<Skeleton class="h-5 w-16 rounded-full" />
						</div>
					</CardHeader>
					<CardContent class="space-y-3">
						<Skeleton class="h-4 w-full" />
						<Skeleton class="h-4 w-3/4" />
					</CardContent>
					<CardFooter>
						<Skeleton class="h-7 w-16" />
					</CardFooter>
				</Card>
			{/each}
		</div>
	{:else if connectionsQuery.isError}
		<Alert variant="destructive">
			<AlertCircle class="h-4 w-4" />
			<AlertTitle>Failed to load integrations</AlertTitle>
			<AlertDescription class="flex items-center justify-between mt-1">
				<span>{connectionsQuery.error?.message ?? 'An unexpected error occurred.'}</span>
				<Button
					variant="outline"
					size="sm"
					onclick={() => connectionsQuery.refetch()}
					class="ml-4 shrink-0 gap-1.5"
				>
					<RefreshCw class="h-3 w-3" />
					Retry
				</Button>
			</AlertDescription>
		</Alert>
	{:else if !connectionsQuery.data || connectionsQuery.data.length === 0}
		<!-- Empty state -->
		<div class="flex flex-col items-center justify-center rounded-xl border-2 border-dashed border-border py-20 text-center">
			<div class="h-14 w-14 rounded-xl bg-muted flex items-center justify-center mb-4">
				<Puzzle class="h-7 w-7 text-muted-foreground" />
			</div>
			<h3 class="text-lg font-semibold">No integrations yet</h3>
			<p class="text-sm text-muted-foreground mt-2 max-w-sm">
				Connect your first tool to automatically sync signals from Gong, Intercom, Slack, and more.
			</p>
			<Button onclick={openAddDialog} class="mt-6 gap-2">
				<Plus class="h-4 w-4" />
				Add your first integration
			</Button>
		</div>
	{:else}
		<div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
			{#each connectionsQuery.data as connection (connection.id)}
				{@const def = providerDef(connection.providerConfigKey)}
				{@const isSyncing = syncingIds.has(connection.connectionId)}
				<Card class="flex flex-col group">
					<CardHeader class="pb-3">
						<div class="flex items-start justify-between gap-2">
							<!-- Provider icon + name -->
							<div class="flex items-center gap-2.5 min-w-0">
								<div class={cn('h-8 w-8 rounded-lg flex items-center justify-center shrink-0', def?.iconBg ?? 'bg-muted')}>
									{#if connection.providerConfigKey === 'webhook'}
										<Webhook class="h-4 w-4 {def?.color}" />
									{:else if ['slack', 'intercom', 'hubspot', 'linear', 'jira', 'gong', 'notion'].includes(connection.providerConfigKey)}
										<IntegrationIcon name={connection.providerConfigKey as IntegrationIconName} class="h-4 w-4" />
									{:else}
										<Globe class="h-4 w-4 {def?.color}" />
									{/if}
								</div>
								<div class="min-w-0">
									<p class="text-sm font-semibold leading-tight truncate">
										{providerLabel(connection.providerConfigKey)}
									</p>
									<span class="inline-flex items-center mt-0.5 rounded-full px-2 py-0.5 text-[11px] font-medium {badgeColor(connection.providerConfigKey)}">
										{connection.providerConfigKey.replace('_', ' ')}
									</span>
								</div>
							</div>
							<!-- Connected status -->
							<span class="inline-flex items-center gap-1 rounded-full px-2.5 py-0.5 text-xs font-medium bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400 shrink-0">
								<span class="h-1.5 w-1.5 rounded-full bg-green-500 inline-block"></span>
								Connected
							</span>
						</div>
					</CardHeader>

					<CardContent class="flex-1 space-y-3 pb-3 text-sm">
						{#if connection.providerConfigKey === 'webhook'}
							<!-- Custom Webhook: show masked endpoint -->
							<div>
								<p class="text-xs font-medium text-muted-foreground mb-1">Webhook endpoint</p>
								<p class="text-xs font-mono text-muted-foreground break-all leading-relaxed bg-muted/60 rounded-md px-2 py-1.5">
									{maskedWebhookUrl(connection.projectId)}
								</p>
							</div>
						{:else}
							<!-- OAuth connection: show sync status -->
							<div class="flex items-center gap-1.5 text-xs text-muted-foreground">
								{#if isSyncing}
									<Loader2 class="h-3 w-3 animate-spin shrink-0" />
									<span>Syncing…</span>
								{:else}
									<RefreshCw class="h-3 w-3 shrink-0" />
									<span>Last synced: {formatDate(connection.lastSyncAt)}</span>
								{/if}
							</div>
						{/if}

						<!-- Connected date -->
						<div class="text-xs text-muted-foreground">
							Connected {formatDate(connection.createdAt)}
						</div>
					</CardContent>

					<CardFooter class="pt-0 border-t border-border/50 mt-auto gap-2">
						{#if deleteConfirmId === connection.id}
							<div class="flex items-center gap-2 w-full py-1">
								<p class="text-xs text-muted-foreground flex-1">Disconnect this integration?</p>
								<Button
									variant="destructive"
									size="sm"
									class="h-7 text-xs"
									onclick={() => handleDisconnect(connection)}
									disabled={deleteConnectionMutation.isPending}
								>
									{#if deleteConnectionMutation.isPending}
										<Loader2 class="h-3 w-3 animate-spin" />
									{:else}
										Confirm
									{/if}
								</Button>
								<Button
									variant="ghost"
									size="sm"
									class="h-7 text-xs"
									onclick={() => (deleteConfirmId = null)}
								>
									Cancel
								</Button>
							</div>
						{:else}
							<!-- Sync Now (OAuth providers only) -->
							{#if connection.providerConfigKey !== 'webhook'}
								<Button
									variant="ghost"
									size="sm"
									class="h-7 text-xs gap-1.5"
									onclick={() => handleSync(connection)}
									disabled={isSyncing}
								>
									<RotateCw class="h-3.5 w-3.5 {isSyncing ? 'animate-spin' : ''}" />
									Sync Now
								</Button>
							{/if}
							<!-- Disconnect -->
							<Button
								variant="ghost"
								size="sm"
								class="h-7 text-xs text-muted-foreground hover:text-destructive hover:bg-destructive/10 gap-1.5 ml-auto"
								onclick={() => (deleteConfirmId = connection.id)}
							>
								<Trash2 class="h-3.5 w-3.5" />
								Disconnect
							</Button>
						{/if}
					</CardFooter>
				</Card>
			{/each}
		</div>
	{/if}
</div>

<!-- ── Add Integration Dialog ─────────────────────────────────────────────── -->
<Dialog bind:open={addDialogOpen}>
	<DialogContent
		class={cn(
			'transition-all duration-200',
			dialogStep === 'choose' ? 'sm:max-w-2xl' : 'sm:max-w-lg'
		)}
	>
		<!-- Progress breadcrumb -->
		<div class="flex items-center gap-1.5 text-xs text-muted-foreground mb-1">
			{#each (['choose', isOAuthProvider ? 'connecting' : 'webhook', isOAuthProvider ? 'success' : 'webhook'] as const) as step, i}
				{@const steps: DialogStep[] = isOAuthProvider
					? ['choose', 'connecting', 'success']
					: ['choose', 'webhook']}
				{@const stepIndex = steps.indexOf(step as DialogStep)}
				{@const currentIndex = steps.indexOf(dialogStep)}
				{@const isActive = step === dialogStep}
				{@const isDone = stepIndex >= 0 && stepIndex < currentIndex}
				{#if i > 0 && stepIndex >= 0}
					<ChevronRight class="h-3 w-3 shrink-0 text-muted-foreground/40" />
				{/if}
				{#if stepIndex >= 0}
					<span class={cn(
						'font-medium',
						isActive ? 'text-foreground' : isDone ? 'text-muted-foreground' : 'text-muted-foreground/40'
					)}>
						{STEP_LABELS[step as DialogStep]}
					</span>
				{/if}
			{/each}
		</div>

		<!-- ── STEP 1: Choose provider ─────────────────────────────────────── -->
		{#if dialogStep === 'choose'}
			<DialogHeader>
				<DialogTitle>Add Integration</DialogTitle>
				<DialogDescription>
					Choose a provider to connect. OAuth providers open a secure login popup — no configuration required.
				</DialogDescription>
			</DialogHeader>

			<div class="grid grid-cols-2 sm:grid-cols-4 gap-2 py-2">
				{#each PROVIDERS as provider (provider.value)}
					<button
						type="button"
						onclick={() => (selectedProvider = provider.value)}
						class={cn(
							'relative flex flex-col items-start gap-2 rounded-lg border p-3 text-left transition-all hover:bg-accent/50 focus:outline-none focus-visible:ring-2 focus-visible:ring-ring',
							selectedProvider === provider.value
								? 'border-primary bg-primary/5 ring-1 ring-primary'
								: 'border-border hover:border-primary/40'
						)}
					>
						<!-- Icon -->
						<div class={cn('h-8 w-8 rounded-md flex items-center justify-center', provider.iconBg)}>
							{#if provider.value === 'webhook'}
								<Webhook class="h-4 w-4 {provider.color}" />
							{:else}
								<IntegrationIcon name={provider.value as IntegrationIconName} class="h-4 w-4" />
							{/if}
						</div>
						<div>
							<p class="text-sm font-semibold leading-tight">{provider.label}</p>
							<p class="text-[11px] text-muted-foreground leading-snug mt-0.5">{provider.description}</p>
						</div>
						<!-- Selected check -->
						{#if selectedProvider === provider.value}
							<span class="absolute top-2 right-2 h-4 w-4 rounded-full bg-primary flex items-center justify-center">
								<Check class="h-2.5 w-2.5 text-primary-foreground" />
							</span>
						{/if}
						<!-- Badge: OAuth vs Webhook -->
						<span class={cn(
							'text-[10px] font-medium px-1.5 py-0.5 rounded-full',
							(provider.nangoKey !== null || provider.nativeOAuth)
								? 'bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400'
								: 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-400'
						)}>
							{(provider.nangoKey !== null || provider.nativeOAuth) ? 'OAuth' : 'Webhook'}
						</span>
					</button>
				{/each}
			</div>

			<DialogFooter>
				<Button variant="outline" onclick={closeAddDialog}>
					Cancel
				</Button>
				<Button
					onclick={handleNext}
					disabled={!selectedProvider || isBusy}
					class="gap-2"
				>
					{#if isBusy}
						<Loader2 class="h-3.5 w-3.5 animate-spin" />
						Connecting…
					{:else if currentProviderDef?.nangoKey !== null || currentProviderDef?.nativeOAuth}
						Connect with OAuth
						<ArrowRight class="h-3.5 w-3.5" />
					{:else}
						Generate Webhook
						<ArrowRight class="h-3.5 w-3.5" />
					{/if}
				</Button>
			</DialogFooter>

		<!-- ── STEP: OAuth in progress ─────────────────────────────────────── -->
		{:else if dialogStep === 'connecting'}
			<DialogHeader>
				<DialogTitle class="flex items-center gap-2">
					{#if currentProviderDef}
						<div class={cn('h-6 w-6 rounded flex items-center justify-center', currentProviderDef.iconBg)}>
							<Loader2 class="h-3.5 w-3.5 {currentProviderDef.color} animate-spin" />
						</div>
					{/if}
					Connecting to {currentProviderDef?.label ?? 'provider'}…
				</DialogTitle>
				<DialogDescription>
					A login popup has opened. Complete the authorization in that window to continue.
				</DialogDescription>
			</DialogHeader>

			<div class="flex flex-col items-center justify-center py-8 gap-4">
				<div class="h-12 w-12 rounded-full bg-muted flex items-center justify-center">
					<Loader2 class="h-6 w-6 text-muted-foreground animate-spin" />
				</div>
				<p class="text-sm text-muted-foreground text-center max-w-xs">
					Waiting for you to authorize access in the popup window. Do not close this dialog.
				</p>
			</div>

			<DialogFooter>
				<Button variant="outline" onclick={() => (dialogStep = 'choose')}>
					Cancel
				</Button>
			</DialogFooter>

		<!-- ── STEP: OAuth success ─────────────────────────────────────────── -->
		{:else if dialogStep === 'success'}
			<DialogHeader>
				<DialogTitle class="flex items-center gap-2">
					<CheckCircle2 class="h-5 w-5 text-green-500" />
					{connectedProviderLabel} connected
				</DialogTitle>
				<DialogDescription>
					Neuco will now sync your {connectedProviderLabel} data automatically.
				</DialogDescription>
			</DialogHeader>

			<div class="flex flex-col items-center justify-center py-6 gap-3">
				<div class="h-14 w-14 rounded-full bg-green-100 dark:bg-green-900/30 flex items-center justify-center">
					<CheckCircle2 class="h-7 w-7 text-green-600 dark:text-green-400" />
				</div>
				<p class="text-sm text-center text-muted-foreground max-w-xs">
					Connected! Neuco will now sync your <span class="font-medium text-foreground">{connectedProviderLabel}</span> data automatically.
				</p>
			</div>

			<DialogFooter class="flex-col sm:flex-row gap-2">
				<Button variant="outline" onclick={openAddDialog} class="sm:mr-auto">
					Add Another
				</Button>
				<Button onclick={closeAddDialog}>
					Done
				</Button>
			</DialogFooter>

		<!-- ── STEP: Custom Webhook URL ────────────────────────────────────── -->
		{:else if dialogStep === 'webhook'}
			<DialogHeader>
				<DialogTitle class="flex items-center gap-2">
					<div class="h-6 w-6 rounded flex items-center justify-center bg-emerald-50 dark:bg-emerald-950/50">
						<Webhook class="h-3.5 w-3.5 text-emerald-600 dark:text-emerald-400" />
					</div>
					Custom Webhook created
				</DialogTitle>
				<DialogDescription>
					Send a POST request to this URL to push signals into Neuco from any tool or script.
				</DialogDescription>
			</DialogHeader>

			<div class="space-y-4 py-1">
				<!-- Webhook URL -->
				<div class="space-y-2">
					<p class="text-xs font-medium text-muted-foreground">Webhook URL</p>
					<div class="relative">
						<code class="block w-full text-xs font-mono bg-muted rounded-lg px-3 py-3 pr-24 break-all leading-relaxed">
							{webhookUrl}
						</code>
						<Button
							variant={webhookUrlCopied ? 'secondary' : 'default'}
							size="sm"
							onclick={copyWebhookUrl}
							class="absolute right-2 top-1/2 -translate-y-1/2 gap-1.5 h-7 text-xs"
						>
							{#if webhookUrlCopied}
								<Check class="h-3.5 w-3.5 text-green-500" />
								Copied
							{:else}
								<Copy class="h-3.5 w-3.5" />
								Copy URL
							{/if}
						</Button>
					</div>
				</div>

				<!-- Payload hint -->
				<div class="space-y-1.5">
					<p class="text-xs font-medium text-muted-foreground">Expected JSON payload</p>
					<pre class="text-xs font-mono bg-muted rounded-lg px-3 py-3 overflow-x-auto leading-relaxed"><code>{`{
  "content": "The full text content of the signal",
  "type": "custom",
  "source": "my-tool",
  "meta": {
    "any_key": "any_value"
  }
}`}</code></pre>
				</div>

				<Alert class="border-amber-200 bg-amber-50 dark:border-amber-800 dark:bg-amber-950/30">
					<AlertTitle class="text-amber-800 dark:text-amber-300 text-sm">Save this URL — the secret is shown only once</AlertTitle>
					<AlertDescription class="text-amber-700 dark:text-amber-400 text-xs mt-0.5">
						Copy the URL above before closing this dialog.
					</AlertDescription>
				</Alert>
			</div>

			<DialogFooter>
				<Button onclick={closeAddDialog}>
					Done
				</Button>
			</DialogFooter>
		{/if}
	</DialogContent>
</Dialog>
