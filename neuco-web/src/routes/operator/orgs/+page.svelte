<script lang="ts">
	import { useOperatorOrgs } from '$lib/api/queries/operator';
	import { Badge } from '$lib/components/ui/badge';
	import { Card, CardContent, CardHeader, CardTitle } from '$lib/components/ui/card';
	import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '$lib/components/ui/table';
	import { Skeleton } from '$lib/components/ui/skeleton';
	import { Alert, AlertDescription } from '$lib/components/ui/alert';

	const orgsQuery = useOperatorOrgs();
</script>

<div class="space-y-6">
	<div class="flex items-center justify-between">
		<div>
			<h2 class="text-xl font-semibold">All Organizations</h2>
			<p class="text-sm text-muted-foreground">Manage all tenant organizations.</p>
		</div>
		{#if orgsQuery.data}
			<Badge variant="secondary">{orgsQuery.data.length} orgs</Badge>
		{/if}
	</div>

	{#if orgsQuery.isPending}
		<Card>
			<CardContent class="p-6 space-y-3">
				{#each Array(5) as _}
					<Skeleton class="h-10 w-full" />
				{/each}
			</CardContent>
		</Card>
	{:else if orgsQuery.isError}
		<Alert variant="destructive">
			<AlertDescription>Failed to load organizations. Check your operator token.</AlertDescription>
		</Alert>
	{:else if orgsQuery.data && orgsQuery.data.length > 0}
		<Card>
			<Table>
				<TableHeader>
					<TableRow>
						<TableHead>Name</TableHead>
						<TableHead>Slug</TableHead>
						<TableHead>Plan</TableHead>
						<TableHead class="text-right">Members</TableHead>
						<TableHead class="text-right">Projects</TableHead>
						<TableHead>Created</TableHead>
					</TableRow>
				</TableHeader>
				<TableBody>
					{#each orgsQuery.data as org}
						<TableRow>
							<TableCell class="font-medium">
								<a href="/operator/orgs/{org.id}" class="hover:underline">
									{org.name}
								</a>
							</TableCell>
							<TableCell class="text-muted-foreground">{org.slug}</TableCell>
							<TableCell>
								<Badge variant={org.plan === 'enterprise' ? 'default' : 'secondary'}>
									{org.plan}
								</Badge>
							</TableCell>
							<TableCell class="text-right">{org.memberCount ?? 0}</TableCell>
							<TableCell class="text-right">{org.projectCount ?? 0}</TableCell>
							<TableCell class="text-muted-foreground text-sm">
								{new Date(org.createdAt).toLocaleDateString()}
							</TableCell>
						</TableRow>
					{/each}
				</TableBody>
			</Table>
		</Card>
	{:else}
		<Card>
			<CardContent class="py-12 text-center text-muted-foreground">
				No organizations yet.
			</CardContent>
		</Card>
	{/if}
</div>
