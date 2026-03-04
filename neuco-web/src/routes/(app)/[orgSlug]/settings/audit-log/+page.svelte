<script lang="ts">
	import { page } from '$app/stores';
	import { authStore } from '$lib/stores/auth.svelte';
	import { useAuditLog } from '$lib/api/queries/audit';
	import {
		Table,
		TableBody,
		TableCell,
		TableHead,
		TableHeader,
		TableRow
	} from '$lib/components/ui/table';
	import { Avatar, AvatarImage, AvatarFallback } from '$lib/components/ui/avatar';
	import { Badge } from '$lib/components/ui/badge';
	import { Button } from '$lib/components/ui/button';
	import { Skeleton } from '$lib/components/ui/skeleton';
	import { Alert, AlertTitle, AlertDescription } from '$lib/components/ui/alert';
	import { Separator } from '$lib/components/ui/separator';
	import {
		AlertCircle,
		RefreshCw,
		ScrollText,
		Filter,
		ChevronLeft,
		ChevronRight
	} from 'lucide-svelte';

	let orgId = $state(authStore.currentOrg?.id ?? '');
	$effect(() => {
		orgId = authStore.currentOrg?.id ?? '';
	});

	const PAGE_SIZE = 25;
	let currentPage = $state(1);
	let filterAction = $state('');
	let filterEntityType = $state('');

	const auditQuery = $derived.by(() =>
		useAuditLog(orgId, {
			page: currentPage,
			pageSize: PAGE_SIZE,
			action: filterAction || undefined,
			entityType: filterEntityType || undefined
		})
	);

	const entries = $derived(auditQuery.data?.data ?? []);

	// Static filter options for the action and entity type dropdowns
	const knownActions = [
		'org.created',
		'org.updated',
		'member.invited',
		'member.removed',
		'member.role_changed',
		'project.created',
		'project.updated',
		'project.deleted',
		'integration.created',
		'integration.deleted',
		'pipeline.retried',
		'spec.generated',
		'generation.created'
	];

	const knownEntityTypes = [
		'organization',
		'member',
		'project',
		'integration',
		'pipeline',
		'spec',
		'generation',
		'signal',
		'candidate'
	];

	function formatDate(dateStr: string): string {
		return new Date(dateStr).toLocaleDateString('en-US', {
			month: 'short',
			day: 'numeric',
			year: 'numeric',
			hour: '2-digit',
			minute: '2-digit',
			second: '2-digit'
		});
	}

	function userInitials(name: string): string {
		const parts = name.trim().split(' ');
		return parts
			.slice(0, 2)
			.map((p) => p[0]?.toUpperCase() ?? '')
			.join('');
	}

	function actionLabel(action: string): string {
		return action
			.split('.')
			.map((p) => p.replace(/_/g, ' '))
			.join(' › ');
	}

	function actionClass(action: string): string {
		if (action.includes('delete') || action.includes('removed') || action.includes('failed')) {
			return 'bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400 border-transparent';
		}
		if (action.includes('create') || action.includes('invited') || action.includes('generated')) {
			return 'bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400 border-transparent';
		}
		if (action.includes('update') || action.includes('changed')) {
			return 'bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-400 border-transparent';
		}
		return 'bg-secondary text-secondary-foreground border-transparent';
	}

	function entityTypeClass(type: string): string {
		const colors: Record<string, string> = {
			organization: 'bg-amber-100 text-amber-800 dark:bg-amber-900/30 dark:text-amber-400 border-transparent',
			member: 'bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-400 border-transparent',
			project: 'bg-violet-100 text-violet-800 dark:bg-violet-900/30 dark:text-violet-400 border-transparent',
			integration: 'bg-cyan-100 text-cyan-800 dark:bg-cyan-900/30 dark:text-cyan-400 border-transparent',
			pipeline: 'bg-orange-100 text-orange-800 dark:bg-orange-900/30 dark:text-orange-400 border-transparent',
			spec: 'bg-emerald-100 text-emerald-800 dark:bg-emerald-900/30 dark:text-emerald-400 border-transparent',
			generation: 'bg-indigo-100 text-indigo-800 dark:bg-indigo-900/30 dark:text-indigo-400 border-transparent'
		};
		return colors[type] ?? 'bg-secondary text-secondary-foreground border-transparent';
	}

	function handleFilterChange() {
		currentPage = 1;
	}
</script>

<svelte:head>
	<title>Audit Log — Neuco</title>
</svelte:head>

<div class="py-6 space-y-6">
	<!-- Header -->
	<div>
		<h2 class="text-lg font-semibold">Audit Log</h2>
		<p class="text-muted-foreground text-sm mt-0.5">
			A record of all actions taken within your organization.
		</p>
	</div>

	<!-- Filter bar -->
	<div class="flex items-center gap-3 flex-wrap">
		<div class="flex items-center gap-1.5 text-sm text-muted-foreground">
			<Filter class="h-3.5 w-3.5" />
			<span>Filter:</span>
		</div>

		<select
			bind:value={filterAction}
			onchange={handleFilterChange}
			class="h-8 rounded-md border border-input bg-background px-3 py-1 text-sm ring-offset-background focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2"
		>
			<option value="">All Actions</option>
			{#each knownActions as action (action)}
				<option value={action}>{actionLabel(action)}</option>
			{/each}
		</select>

		<select
			bind:value={filterEntityType}
			onchange={handleFilterChange}
			class="h-8 rounded-md border border-input bg-background px-3 py-1 text-sm ring-offset-background focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2"
		>
			<option value="">All Resource Types</option>
			{#each knownEntityTypes as type (type)}
				<option value={type}>{type.charAt(0).toUpperCase() + type.slice(1)}</option>
			{/each}
		</select>

		{#if filterAction || filterEntityType}
			<Button
				variant="ghost"
				size="sm"
				onclick={() => {
					filterAction = '';
					filterEntityType = '';
					currentPage = 1;
				}}
				class="h-8 text-xs text-muted-foreground"
			>
				Clear filters
			</Button>
		{/if}

		<Button
			variant="ghost"
			size="sm"
			onclick={() => auditQuery.refetch()}
			class="h-8 ml-auto gap-1.5 text-xs"
			disabled={auditQuery.isFetching}
		>
			<RefreshCw class="h-3.5 w-3.5 {auditQuery.isFetching ? 'animate-spin' : ''}" />
			Refresh
		</Button>
	</div>

	<!-- Table -->
	{#if auditQuery.isLoading}
		<div class="rounded-lg border border-border overflow-hidden">
			<div class="bg-muted/30 px-4 py-3 flex gap-6">
				{#each Array(5) as _, i (i)}
					<Skeleton class="h-4 w-24"></Skeleton>
				{/each}
			</div>
			<div class="divide-y divide-border">
				{#each Array(8) as _, i (i)}
					<div class="px-4 py-4 flex items-center gap-4">
						<Skeleton class="h-4 w-36 shrink-0"></Skeleton>
						<div class="flex items-center gap-2 flex-1">
							<Skeleton class="h-7 w-7 rounded-full shrink-0"></Skeleton>
							<Skeleton class="h-4 w-32"></Skeleton>
						</div>
						<Skeleton class="h-5 w-24 rounded-full shrink-0"></Skeleton>
						<Skeleton class="h-5 w-20 rounded-full shrink-0"></Skeleton>
						<Skeleton class="h-4 w-28 shrink-0"></Skeleton>
					</div>
				{/each}
			</div>
		</div>
	{:else if auditQuery.isError}
		<Alert variant="destructive">
			<AlertCircle class="h-4 w-4" />
			<AlertTitle>Failed to load audit log</AlertTitle>
			<AlertDescription class="flex items-center justify-between">
				<span>{auditQuery.error?.message ?? 'An unexpected error occurred.'}</span>
				<Button
					variant="outline"
					size="sm"
					onclick={() => auditQuery.refetch()}
					class="ml-4 shrink-0"
				>
					<RefreshCw class="mr-1.5 h-3 w-3" />
					Retry
				</Button>
			</AlertDescription>
		</Alert>
	{:else if entries.length === 0}
		<div class="flex flex-col items-center justify-center rounded-xl border-2 border-dashed border-border py-16 text-center">
			<ScrollText class="h-10 w-10 text-muted-foreground mb-3" />
			<h3 class="text-base font-semibold">
				{auditQuery.data?.total === 0 ? 'No audit events yet' : 'No matching events'}
			</h3>
			<p class="text-sm text-muted-foreground mt-1 max-w-xs">
				{#if auditQuery.data?.total === 0}
					Actions taken by organization members will appear here.
				{:else}
					Try adjusting your filters to see more results.
				{/if}
			</p>
			{#if filterAction || filterEntityType}
				<Button
					variant="outline"
					size="sm"
					onclick={() => { filterAction = ''; filterEntityType = ''; currentPage = 1; }}
					class="mt-4"
				>
					Clear filters
				</Button>
			{/if}
		</div>
	{:else}
		<div class="rounded-lg border border-border overflow-hidden">
			<Table>
				<TableHeader>
					<TableRow class="bg-muted/30">
						<TableHead class="w-[180px]">Timestamp</TableHead>
						<TableHead class="w-[200px]">User</TableHead>
						<TableHead class="w-[200px]">Action</TableHead>
						<TableHead class="w-[140px]">Resource Type</TableHead>
						<TableHead>Resource ID</TableHead>
					</TableRow>
				</TableHeader>
				<TableBody>
					{#each entries as entry (entry.id)}
						<TableRow>
							<TableCell class="text-xs text-muted-foreground font-mono whitespace-nowrap">
								{formatDate(entry.createdAt)}
							</TableCell>

							<TableCell>
								<div class="flex items-center gap-2">
									<Avatar class="h-7 w-7 shrink-0">
										<AvatarImage
											src={entry.user?.avatarUrl}
											alt={entry.user?.name ?? 'Unknown'}
										/>
										<AvatarFallback class="text-xs bg-muted">
											{entry.user?.name ? userInitials(entry.user.name) : '?'}
										</AvatarFallback>
									</Avatar>
									<div class="min-w-0">
										<p class="text-sm font-medium truncate">
											{entry.user?.name ?? 'Unknown'}
										</p>
									</div>
								</div>
							</TableCell>

							<TableCell>
								<span class="inline-flex items-center rounded-full border px-2.5 py-0.5 text-xs font-semibold {actionClass(entry.action)}">
									{actionLabel(entry.action)}
								</span>
							</TableCell>

							<TableCell>
								<span class="inline-flex items-center rounded-full border px-2.5 py-0.5 text-xs font-semibold {entityTypeClass(entry.entityType)}">
									{entry.entityType}
								</span>
							</TableCell>

							<TableCell>
								<span class="font-mono text-xs text-muted-foreground">
									{entry.entityId ? entry.entityId.slice(0, 12) + (entry.entityId.length > 12 ? '...' : '') : '—'}
								</span>
							</TableCell>
						</TableRow>
					{/each}
				</TableBody>
			</Table>
		</div>

		<!-- Pagination -->
		{#if auditQuery.data && auditQuery.data.totalPages > 1}
			<div class="flex items-center justify-between">
				<p class="text-sm text-muted-foreground">
					Showing {(currentPage - 1) * PAGE_SIZE + 1}–{Math.min(
						currentPage * PAGE_SIZE,
						auditQuery.data.total
					)} of {auditQuery.data.total} events
				</p>
				<div class="flex items-center gap-2">
					<Button
						variant="outline"
						size="sm"
						class="h-8 gap-1"
						disabled={currentPage <= 1 || auditQuery.isFetching}
						onclick={() => currentPage--}
					>
						<ChevronLeft class="h-3.5 w-3.5" />
						Previous
					</Button>
					<span class="text-sm text-muted-foreground px-2">
						Page {currentPage} of {auditQuery.data.totalPages}
					</span>
					<Button
						variant="outline"
						size="sm"
						class="h-8 gap-1"
						disabled={currentPage >= auditQuery.data.totalPages || auditQuery.isFetching}
						onclick={() => currentPage++}
					>
						Next
						<ChevronRight class="h-3.5 w-3.5" />
					</Button>
				</div>
			</div>
		{/if}
	{/if}
</div>
