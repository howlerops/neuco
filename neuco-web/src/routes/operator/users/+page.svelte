<script lang="ts">
	import { useOperatorUsers } from '$lib/api/queries/operator';
	import { Badge } from '$lib/components/ui/badge';
	import { Card, CardContent } from '$lib/components/ui/card';
	import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '$lib/components/ui/table';
	import { Avatar, AvatarFallback, AvatarImage } from '$lib/components/ui/avatar';
	import { Skeleton } from '$lib/components/ui/skeleton';
	import { Alert, AlertDescription } from '$lib/components/ui/alert';

	const usersQuery = useOperatorUsers();
</script>

<div class="space-y-6">
	<div class="flex items-center justify-between">
		<div>
			<h2 class="text-xl font-semibold">All Users</h2>
			<p class="text-sm text-muted-foreground">All registered users across all organizations.</p>
		</div>
		{#if usersQuery.data}
			<Badge variant="secondary">{usersQuery.data.length} users</Badge>
		{/if}
	</div>

	{#if usersQuery.isPending}
		<Card>
			<CardContent class="p-6 space-y-3">
				{#each Array(5) as _}
					<Skeleton class="h-10 w-full" />
				{/each}
			</CardContent>
		</Card>
	{:else if usersQuery.isError}
		<Alert variant="destructive">
			<AlertDescription>Failed to load users.</AlertDescription>
		</Alert>
	{:else if usersQuery.data && usersQuery.data.length > 0}
		<Card>
			<Table>
				<TableHeader>
					<TableRow>
						<TableHead>User</TableHead>
						<TableHead>GitHub</TableHead>
						<TableHead>Email</TableHead>
						<TableHead>Created</TableHead>
					</TableRow>
				</TableHeader>
				<TableBody>
					{#each usersQuery.data as user}
						<TableRow>
							<TableCell>
								<div class="flex items-center gap-3">
									<Avatar class="h-8 w-8">
										{#if user.avatarUrl}
											<AvatarImage src={user.avatarUrl} alt={user.githubLogin} />
										{/if}
										<AvatarFallback>
											{user.githubLogin.slice(0, 2).toUpperCase()}
										</AvatarFallback>
									</Avatar>
									<span class="font-medium">{user.githubLogin}</span>
								</div>
							</TableCell>
							<TableCell>
								<a
									href="https://github.com/{user.githubLogin}"
									target="_blank"
									rel="noopener"
									class="text-sm text-muted-foreground hover:text-foreground hover:underline"
								>
									@{user.githubLogin}
								</a>
							</TableCell>
							<TableCell class="text-muted-foreground">{user.email ?? '—'}</TableCell>
							<TableCell class="text-muted-foreground text-sm">
								{new Date(user.createdAt).toLocaleDateString()}
							</TableCell>
						</TableRow>
					{/each}
				</TableBody>
			</Table>
		</Card>
	{:else}
		<Card>
			<CardContent class="py-12 text-center text-muted-foreground">
				No users yet.
			</CardContent>
		</Card>
	{/if}
</div>
