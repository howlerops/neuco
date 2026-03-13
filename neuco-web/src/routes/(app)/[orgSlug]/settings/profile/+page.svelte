<script lang="ts">
	import { useMe } from '$lib/api/queries/auth.svelte';
	import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '$lib/components/ui/card';
	import { Input } from '$lib/components/ui/input';
	import { Label } from '$lib/components/ui/label';
	import { Skeleton } from '$lib/components/ui/skeleton';
	import { Separator } from '$lib/components/ui/separator';
	import { Avatar, AvatarImage, AvatarFallback } from '$lib/components/ui/avatar';
	import { Alert, AlertTitle, AlertDescription } from '$lib/components/ui/alert';
	import { Button } from '$lib/components/ui/button';
	import { AlertCircle, RefreshCw, User } from 'lucide-svelte';

	const meQuery = $derived.by(() => useMe());

	const user = $derived(meQuery.data?.user);

	const initials = $derived(
		user?.name
			? user.name
					.split(' ')
					.map((n) => n[0])
					.join('')
					.toUpperCase()
					.slice(0, 2)
			: '?'
	);
</script>

<svelte:head>
	<title>Profile — Neuco</title>
</svelte:head>

<div class="py-6 space-y-8 max-w-2xl">
	{#if meQuery.isLoading}
		<Card>
			<CardHeader>
				<Skeleton class="h-5 w-40"></Skeleton>
				<Skeleton class="h-4 w-64 mt-1"></Skeleton>
			</CardHeader>
			<CardContent class="space-y-4">
				<div class="flex items-center gap-4">
					<Skeleton class="h-16 w-16 rounded-full"></Skeleton>
					<div class="space-y-2">
						<Skeleton class="h-4 w-32"></Skeleton>
						<Skeleton class="h-4 w-48"></Skeleton>
					</div>
				</div>
			</CardContent>
		</Card>
	{:else if meQuery.isError}
		<Alert variant="destructive">
			<AlertCircle class="h-4 w-4" />
			<AlertTitle>Failed to load profile</AlertTitle>
			<AlertDescription class="flex items-center justify-between">
				<span>{meQuery.error?.message ?? 'An unexpected error occurred.'}</span>
				<Button
					variant="outline"
					size="sm"
					onclick={() => meQuery.refetch()}
					class="ml-4 shrink-0"
				>
					<RefreshCw class="mr-1.5 h-3 w-3" />
					Retry
				</Button>
			</AlertDescription>
		</Alert>
	{:else if user}
		<Card>
			<CardHeader>
				<CardTitle class="flex items-center gap-2">
					<User class="h-4 w-4" />
					Profile
				</CardTitle>
				<CardDescription>Your personal account information.</CardDescription>
			</CardHeader>
			<CardContent class="space-y-6">
				<!-- Avatar -->
				<div class="flex items-center gap-4">
					<Avatar class="h-16 w-16">
						<AvatarImage src={user.avatarUrl} alt={user.name} />
						<AvatarFallback class="text-lg">{initials}</AvatarFallback>
					</Avatar>
					<div>
						<p class="font-medium">{user.name}</p>
						<p class="text-sm text-muted-foreground">{user.email}</p>
					</div>
				</div>

				<Separator />

				<!-- Name (read-only) -->
				<div class="space-y-2">
					<Label for="profile-name">
						Name
						<span class="text-muted-foreground font-normal ml-1">(from GitHub/Google)</span>
					</Label>
					<Input
						id="profile-name"
						value={user.name}
						disabled
						class="bg-muted/40 text-muted-foreground"
					/>
				</div>

				<!-- Email (read-only) -->
				<div class="space-y-2">
					<Label for="profile-email">
						Email
						<span class="text-muted-foreground font-normal ml-1">(from GitHub/Google)</span>
					</Label>
					<Input
						id="profile-email"
						value={user.email}
						disabled
						class="bg-muted/40 text-muted-foreground"
					/>
				</div>

				{#if user.githubLogin}
					<div class="space-y-2">
						<Label for="profile-github">GitHub</Label>
						<Input
							id="profile-github"
							value={user.githubLogin}
							disabled
							class="font-mono bg-muted/40 text-muted-foreground"
						/>
					</div>
				{/if}

				<Separator />

				<dl class="space-y-3 text-sm">
					<div class="flex justify-between">
						<dt class="text-muted-foreground">User ID</dt>
						<dd class="font-mono text-xs text-foreground">{user.id}</dd>
					</div>
					<Separator />
					<div class="flex justify-between">
						<dt class="text-muted-foreground">Account created</dt>
						<dd class="text-foreground">
							{new Date(user.createdAt).toLocaleDateString('en-US', {
								month: 'long',
								day: 'numeric',
								year: 'numeric'
							})}
						</dd>
					</div>
				</dl>
			</CardContent>
		</Card>
	{/if}
</div>
