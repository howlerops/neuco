<script lang="ts">
	import { useOperatorFlags, useToggleFlag, type FeatureFlag } from '$lib/api/queries/operator';
	import { Badge } from '$lib/components/ui/badge';
	import { Card, CardContent } from '$lib/components/ui/card';
	import {
		Table,
		TableBody,
		TableCell,
		TableHead,
		TableHeader,
		TableRow
	} from '$lib/components/ui/table';
	import { Skeleton } from '$lib/components/ui/skeleton';
	import { Alert, AlertDescription } from '$lib/components/ui/alert';
	import { Switch } from '$lib/components/ui/switch';
	import { Button } from '$lib/components/ui/button';
	import { toast } from 'svelte-sonner';

	const flagsQuery = useOperatorFlags();
	const toggleMutation = useToggleFlag();

	// Track which flag is currently being toggled to show loading state.
	let togglingKey = $state<string | null>(null);

	async function handleToggle(flag: FeatureFlag, newEnabled: boolean) {
		togglingKey = flag.key;
		try {
			await toggleMutation.mutateAsync({ key: flag.key, enabled: newEnabled });
			toast.success(`Flag "${flag.key}" ${newEnabled ? 'enabled' : 'disabled'}`);
		} catch (err) {
			toast.error('Failed to update flag', {
				description: err instanceof Error ? err.message : 'An unexpected error occurred.'
			});
		} finally {
			togglingKey = null;
		}
	}

	function formatDate(dateStr: string | null | undefined): string {
		if (!dateStr) return 'Never';
		const date = new Date(dateStr);
		if (isNaN(date.getTime())) return 'Never';
		return date.toLocaleDateString('en-US', {
			month: 'short',
			day: 'numeric',
			year: 'numeric',
			hour: '2-digit',
			minute: '2-digit'
		});
	}

	function formatKey(key: string): string {
		return key.replace(/_/g, ' ').replace(/\b\w/g, (c) => c.toUpperCase());
	}
</script>

<svelte:head>
	<title>Feature Flags -- Neuco Operator</title>
</svelte:head>

<div class="space-y-6">
	<div class="flex items-center justify-between">
		<div>
			<h2 class="text-xl font-semibold">Feature Flags</h2>
			<p class="text-sm text-muted-foreground">
				Toggle feature flags to control system behaviour. Changes take effect immediately.
			</p>
		</div>
		{#if flagsQuery.data}
			<div class="flex items-center gap-2">
				<Badge variant="secondary">
					{flagsQuery.data.filter((f) => f.enabled).length} / {flagsQuery.data.length} enabled
				</Badge>
			</div>
		{/if}
	</div>

	{#if flagsQuery.isPending}
		<Card>
			<CardContent class="p-6 space-y-3">
				{#each Array(6) as _}
					<Skeleton class="h-12 w-full" />
				{/each}
			</CardContent>
		</Card>
	{:else if flagsQuery.isError}
		<Alert variant="destructive">
			<AlertDescription class="flex items-center justify-between">
				<span>Failed to load feature flags. Check your operator token.</span>
				<Button variant="outline" size="sm" onclick={() => flagsQuery.refetch()}>Retry</Button>
			</AlertDescription>
		</Alert>
	{:else if flagsQuery.data && flagsQuery.data.length > 0}
		<Card>
			<Table>
				<TableHeader>
					<TableRow>
						<TableHead class="w-[200px]">Key</TableHead>
						<TableHead>Description</TableHead>
						<TableHead class="w-[100px] text-center">Status</TableHead>
						<TableHead class="w-[180px]">Last Updated</TableHead>
						<TableHead class="w-[160px]">Updated By</TableHead>
					</TableRow>
				</TableHeader>
				<TableBody>
					{#each flagsQuery.data as flag (flag.key)}
						{@const isToggling = togglingKey === flag.key}
						<TableRow>
							<TableCell>
								<code
									class="text-sm font-mono bg-muted/60 rounded px-1.5 py-0.5 break-all"
								>
									{flag.key}
								</code>
							</TableCell>
							<TableCell class="text-sm text-muted-foreground">
								{flag.description || '---'}
							</TableCell>
							<TableCell class="text-center">
								<div class="flex items-center justify-center gap-2">
									<Switch
										checked={flag.enabled}
										disabled={isToggling}
										onCheckedChange={(checked) => handleToggle(flag, checked)}
									/>
									<Badge variant={flag.enabled ? 'default' : 'secondary'} class="text-[10px]">
										{flag.enabled ? 'ON' : 'OFF'}
									</Badge>
								</div>
							</TableCell>
							<TableCell class="text-sm text-muted-foreground">
								{formatDate(flag.updatedAt)}
							</TableCell>
							<TableCell class="text-sm text-muted-foreground">
								{#if flag.updatedBy && flag.updatedBy !== '00000000-0000-0000-0000-000000000000'}
									<code class="text-xs font-mono">{flag.updatedBy.slice(0, 8)}...</code>
								{:else}
									<span class="text-xs">System</span>
								{/if}
							</TableCell>
						</TableRow>
					{/each}
				</TableBody>
			</Table>
		</Card>
	{:else}
		<Card>
			<CardContent class="py-12 text-center text-muted-foreground">
				No feature flags configured.
			</CardContent>
		</Card>
	{/if}
</div>
