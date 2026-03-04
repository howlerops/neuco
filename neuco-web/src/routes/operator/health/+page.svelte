<script lang="ts">
	import { useOperatorHealth } from '$lib/api/queries/operator';
	import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '$lib/components/ui/card';
	import { Badge } from '$lib/components/ui/badge';
	import { Skeleton } from '$lib/components/ui/skeleton';
	import { Alert, AlertDescription } from '$lib/components/ui/alert';

	const healthQuery = useOperatorHealth();

	function statusColor(status: string): 'default' | 'secondary' | 'destructive' {
		if (status === 'ok' || status === 'healthy') return 'default';
		if (status === 'degraded') return 'secondary';
		return 'destructive';
	}
</script>

<div class="space-y-6">
	<div>
		<h2 class="text-xl font-semibold">System Health</h2>
		<p class="text-sm text-muted-foreground">Real-time status of Neuco's backend services. Auto-refreshes every 30 seconds.</p>
	</div>

	{#if healthQuery.isPending}
		<div class="grid gap-4 md:grid-cols-3">
			{#each Array(3) as _}
				<Card>
					<CardContent class="p-6">
						<Skeleton class="h-6 w-32 mb-2" />
						<Skeleton class="h-8 w-20" />
					</CardContent>
				</Card>
			{/each}
		</div>
	{:else if healthQuery.isError}
		<Alert variant="destructive">
			<AlertDescription>
				Cannot reach the backend. The API server may be down.
			</AlertDescription>
		</Alert>
	{:else if healthQuery.data}
		<div class="grid gap-4 md:grid-cols-3">
			<Card>
				<CardHeader class="pb-2">
					<CardDescription>Database</CardDescription>
					<CardTitle class="flex items-center gap-2">
						<Badge variant={statusColor(healthQuery.data.checks?.database ?? 'unknown')}>
							{healthQuery.data.checks?.database ?? 'unknown'}
						</Badge>
					</CardTitle>
				</CardHeader>
				<CardContent>
					<p class="text-sm text-muted-foreground">PostgreSQL connection pool status.</p>
				</CardContent>
			</Card>

			<Card>
				<CardHeader class="pb-2">
					<CardDescription>Queue Depths</CardDescription>
					<CardTitle class="text-sm font-mono">
						{healthQuery.data.checks?.queue ?? 'unknown'}
					</CardTitle>
				</CardHeader>
				<CardContent>
					<p class="text-sm text-muted-foreground">River job queue state distribution.</p>
				</CardContent>
			</Card>

			<Card>
				<CardHeader class="pb-2">
					<CardDescription>Version</CardDescription>
					<CardTitle class="text-sm font-mono">
						{healthQuery.data.status ?? 'unknown'}
					</CardTitle>
				</CardHeader>
				<CardContent>
					<p class="text-sm text-muted-foreground">Currently deployed API version.</p>
				</CardContent>
			</Card>
		</div>

		<Card>
			<CardHeader>
				<CardTitle>Raw Health Response</CardTitle>
			</CardHeader>
			<CardContent>
				<pre class="bg-muted p-4 rounded-lg text-sm font-mono overflow-auto">{JSON.stringify(healthQuery.data, null, 2)}</pre>
			</CardContent>
		</Card>
	{/if}
</div>
