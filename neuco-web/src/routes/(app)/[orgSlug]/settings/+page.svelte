<script lang="ts">
	import { page } from '$app/stores';
	import { authStore } from '$lib/stores/auth.svelte';
	import { useOrg, useUpdateOrg } from '$lib/api/queries/organizations';
	import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '$lib/components/ui/card';
	import { Badge } from '$lib/components/ui/badge';
	import { Button } from '$lib/components/ui/button';
	import { Input } from '$lib/components/ui/input';
	import { Label } from '$lib/components/ui/label';
	import { Skeleton } from '$lib/components/ui/skeleton';
	import { Alert, AlertTitle, AlertDescription } from '$lib/components/ui/alert';
	import { Separator } from '$lib/components/ui/separator';
	import { toast } from 'svelte-sonner';
	import {
		AlertCircle,
		RefreshCw,
		Save,
		Loader2,
		Building2,
		BadgeCheck
	} from 'lucide-svelte';

	const orgSlug = $derived($page.params.orgSlug ?? '');

	let orgId = $state(authStore.currentOrg?.id ?? '');
	$effect(() => {
		orgId = authStore.currentOrg?.id ?? '';
	});

	const orgQuery = $derived.by(() => useOrg(orgSlug));
	const updateMutation = $derived.by(() => useUpdateOrg(orgId));

	// Local form state
	let orgName = $state('');
	let isDirty = $state(false);

	// Populate form when org data loads
	$effect(() => {
		const org = orgQuery.data;
		if (org && !isDirty) {
			orgName = org.name;
		}
	});

	function handleNameInput(e: Event) {
		orgName = (e.target as HTMLInputElement).value;
		isDirty = orgName !== (orgQuery.data?.name ?? '');
	}

	async function handleSave() {
		if (!orgName.trim()) {
			toast.error('Organization name cannot be empty');
			return;
		}
		if (!orgId) {
			toast.error('Organization not loaded');
			return;
		}

		try {
			const updated = await updateMutation.mutateAsync({ name: orgName.trim() });
			authStore.setOrg(updated);
			isDirty = false;
			toast.success('Organization name updated');
		} catch (err) {
			toast.error('Failed to update organization', {
				description: err instanceof Error ? err.message : 'An unexpected error occurred.'
			});
		}
	}

	// Plan badge helper (extend if plan info comes from API)
	function planLabel(plan?: string): string {
		if (!plan) return 'Free';
		return plan.charAt(0).toUpperCase() + plan.slice(1);
	}

	function planVariant(plan?: string): 'default' | 'secondary' | 'outline' {
		switch (plan) {
			case 'pro': return 'default';
			case 'enterprise': return 'default';
			default: return 'secondary';
		}
	}
</script>

<svelte:head>
	<title>General Settings — Neuco</title>
</svelte:head>

<div class="py-6 space-y-8 max-w-2xl">
	{#if orgQuery.isLoading}
		<Card>
			<CardHeader>
				<Skeleton class="h-5 w-40"></Skeleton>
				<Skeleton class="h-4 w-64 mt-1"></Skeleton>
			</CardHeader>
			<CardContent class="space-y-4">
				<div class="space-y-2">
					<Skeleton class="h-4 w-24"></Skeleton>
					<Skeleton class="h-10 w-full"></Skeleton>
				</div>
				<div class="space-y-2">
					<Skeleton class="h-4 w-24"></Skeleton>
					<Skeleton class="h-10 w-full"></Skeleton>
				</div>
			</CardContent>
		</Card>
	{:else if orgQuery.isError}
		<Alert variant="destructive">
			<AlertCircle class="h-4 w-4" />
			<AlertTitle>Failed to load organization</AlertTitle>
			<AlertDescription class="flex items-center justify-between">
				<span>{orgQuery.error?.message ?? 'An unexpected error occurred.'}</span>
				<Button
					variant="outline"
					size="sm"
					onclick={() => orgQuery.refetch()}
					class="ml-4 shrink-0"
				>
					<RefreshCw class="mr-1.5 h-3 w-3" />
					Retry
				</Button>
			</AlertDescription>
		</Alert>
	{:else if orgQuery.data}
		{@const org = orgQuery.data}

		<!-- Organization Details -->
		<Card>
			<CardHeader>
				<CardTitle class="flex items-center gap-2">
					<Building2 class="h-4 w-4" />
					Organization
				</CardTitle>
				<CardDescription>
					Basic information about your organization.
				</CardDescription>
			</CardHeader>
			<CardContent class="space-y-5">
				<!-- Name field -->
				<div class="space-y-2">
					<Label for="org-name">Organization Name</Label>
					<div class="flex items-center gap-2">
						<Input
							id="org-name"
							value={orgName}
							oninput={handleNameInput}
							placeholder="Acme Corp"
							class="flex-1"
							disabled={updateMutation.isPending}
						/>
						<Button
							onclick={handleSave}
							disabled={!isDirty || updateMutation.isPending || !orgName.trim()}
							size="sm"
							class="gap-1.5 shrink-0"
						>
							{#if updateMutation.isPending}
								<Loader2 class="h-3.5 w-3.5 animate-spin" />
								Saving...
							{:else}
								<Save class="h-3.5 w-3.5" />
								Save
							{/if}
						</Button>
					</div>
				</div>

				<Separator />

				<!-- Slug field (read-only) -->
				<div class="space-y-2">
					<Label for="org-slug">
						Organization Slug
						<span class="text-muted-foreground font-normal ml-1">(read-only)</span>
					</Label>
					<Input
						id="org-slug"
						value={org.slug}
						disabled
						class="font-mono bg-muted/40 text-muted-foreground"
					/>
					<p class="text-xs text-muted-foreground">
						The slug is used in URLs and cannot be changed after creation.
					</p>
				</div>

				<Separator />

				<!-- Plan -->
				<div class="space-y-2">
					<Label>Plan</Label>
					<div class="flex items-center gap-3">
						<Badge variant={planVariant((org as unknown as { plan?: string }).plan)}>
							<BadgeCheck class="mr-1 h-3 w-3" />
							{planLabel((org as unknown as { plan?: string }).plan)}
						</Badge>
						<span class="text-xs text-muted-foreground">
							{(org as unknown as { plan?: string }).plan === 'pro' || (org as unknown as { plan?: string }).plan === 'enterprise'
								? 'All features unlocked'
								: 'Upgrade for unlimited pipelines & members'}
						</span>
					</div>
				</div>
			</CardContent>
		</Card>

		<!-- Org metadata -->
		<Card>
			<CardHeader>
				<CardTitle class="text-sm font-medium text-muted-foreground">
					Organization Details
				</CardTitle>
			</CardHeader>
			<CardContent>
				<dl class="space-y-3 text-sm">
					<div class="flex justify-between">
						<dt class="text-muted-foreground">Organization ID</dt>
						<dd class="font-mono text-xs text-foreground">{org.id}</dd>
					</div>
					<Separator />
					<div class="flex justify-between">
						<dt class="text-muted-foreground">Created</dt>
						<dd class="text-foreground">
							{new Date(org.createdAt).toLocaleDateString('en-US', {
								month: 'long',
								day: 'numeric',
								year: 'numeric'
							})}
						</dd>
					</div>
					<Separator />
					<div class="flex justify-between">
						<dt class="text-muted-foreground">Last updated</dt>
						<dd class="text-foreground">
							{new Date(org.updatedAt).toLocaleDateString('en-US', {
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
