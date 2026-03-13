<script lang="ts">
	import { page } from '$app/stores';
	import { authStore } from '$lib/stores/auth.svelte';
	import { useUsage, useSubscription, useCreateCheckout, useCreatePortalSession } from '$lib/api/queries/billing';
	import { useOrgLLMUsage } from '$lib/api/queries/llm-usage';
	import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '$lib/components/ui/card';
	import { Badge } from '$lib/components/ui/badge';
	import { Button } from '$lib/components/ui/button';
	import { Skeleton } from '$lib/components/ui/skeleton';
	import { Separator } from '$lib/components/ui/separator';
	import { Alert, AlertTitle, AlertDescription } from '$lib/components/ui/alert';
	import { toast } from 'svelte-sonner';
	import {
		AlertCircle,
		RefreshCw,
		Loader2,
		CreditCard,
		BarChart3,
		FolderKanban,
		Signal,
		GitPullRequest,
		ArrowUpRight,
		Zap,
		BrainCircuit,
		Clock,
		Hash,
		DollarSign
	} from 'lucide-svelte';

	const orgSlug = $derived($page.params.orgSlug ?? '');

	let orgId = $state(authStore.currentOrg?.id ?? '');
	$effect(() => {
		orgId = authStore.currentOrg?.id ?? '';
	});

	const usageQuery = $derived.by(() => useUsage(orgId));
	const subQuery = $derived.by(() => useSubscription(orgId));
	const llmUsageQuery = $derived.by(() => useOrgLLMUsage(orgId));
	const checkoutMutation = $derived.by(() => useCreateCheckout(orgId));
	const portalMutation = $derived.by(() => useCreatePortalSession(orgId));

	function tierLabel(tier?: string): string {
		if (!tier) return 'Free Trial';
		return tier.charAt(0).toUpperCase() + tier.slice(1);
	}

	function tierVariant(tier?: string): 'default' | 'secondary' | 'outline' {
		switch (tier) {
			case 'builder':
				return 'default';
			case 'starter':
				return 'secondary';
			default:
				return 'outline';
		}
	}

	function usagePercent(used: number, max: number): number {
		if (max <= 0) return 0;
		return Math.min(Math.round((used / max) * 100), 100);
	}

	function usageColor(percent: number): string {
		if (percent >= 90) return 'bg-red-500';
		if (percent >= 70) return 'bg-amber-500';
		return 'bg-primary';
	}

	function formatCost(usd: number): string {
		if (usd < 0.01) return '< $0.01';
		return `$${usd.toFixed(2)}`;
	}

	function formatTokens(n: number): string {
		if (n >= 1_000_000) return `${(n / 1_000_000).toFixed(1)}M`;
		if (n >= 1_000) return `${(n / 1_000).toFixed(1)}K`;
		return String(n);
	}

	function formatLatency(ms: number): string {
		if (ms >= 1000) return `${(ms / 1000).toFixed(1)}s`;
		return `${Math.round(ms)}ms`;
	}

	async function handleCheckout(planTier: string) {
		try {
			const result = await checkoutMutation.mutateAsync({ planTier });
			window.location.href = result.url;
		} catch (err) {
			toast.error('Failed to start checkout', {
				description: err instanceof Error ? err.message : 'An unexpected error occurred.'
			});
		}
	}

	async function handlePortal() {
		try {
			const result = await portalMutation.mutateAsync();
			window.location.href = result.url;
		} catch (err) {
			toast.error('Failed to open billing portal', {
				description: err instanceof Error ? err.message : 'An unexpected error occurred.'
			});
		}
	}
</script>

<svelte:head>
	<title>Billing & Usage — Neuco</title>
</svelte:head>

<div class="py-6 space-y-8 max-w-2xl">
	{#if usageQuery.isLoading || subQuery.isLoading}
		<Card>
			<CardHeader>
				<Skeleton class="h-5 w-40" />
				<Skeleton class="h-4 w-64 mt-1" />
			</CardHeader>
			<CardContent class="space-y-4">
				<Skeleton class="h-20 w-full" />
				<Skeleton class="h-20 w-full" />
				<Skeleton class="h-20 w-full" />
			</CardContent>
		</Card>
	{:else if usageQuery.isError}
		<Alert variant="destructive">
			<AlertCircle class="h-4 w-4" />
			<AlertTitle>Failed to load usage data</AlertTitle>
			<AlertDescription class="flex items-center justify-between">
				<span>{usageQuery.error?.message ?? 'An unexpected error occurred.'}</span>
				<Button
					variant="outline"
					size="sm"
					onclick={() => usageQuery.refetch()}
					class="ml-4 shrink-0"
				>
					<RefreshCw class="mr-1.5 h-3 w-3" />
					Retry
				</Button>
			</AlertDescription>
		</Alert>
	{:else if usageQuery.data}
		{@const usage = usageQuery.data}
		{@const sub = subQuery.data?.subscription}
		{@const signalPct = usagePercent(usage.signalsUsed, usage.limits.maxSignalsPerMonth)}
		{@const prPct = usagePercent(usage.prsUsed, usage.limits.maxPrsPerMonth)}
		{@const projectPct = usagePercent(usage.projectCount, usage.limits.maxProjects)}

		<!-- Current Plan -->
		<Card>
			<CardHeader>
				<CardTitle class="flex items-center gap-2">
					<CreditCard class="h-4 w-4" />
					Current Plan
				</CardTitle>
				<CardDescription>
					Your subscription and billing details.
				</CardDescription>
			</CardHeader>
			<CardContent class="space-y-4">
				<div class="flex items-center justify-between">
					<div class="flex items-center gap-3">
						<Badge variant={tierVariant(usage.planTier)}>
							{tierLabel(usage.planTier)}
						</Badge>
						{#if sub}
							<span class="text-xs text-muted-foreground capitalize">
								{sub.status === 'active' ? 'Active' : sub.status}
							</span>
						{:else}
							<span class="text-xs text-muted-foreground">
								14-day trial
							</span>
						{/if}
					</div>
					<div class="flex gap-2">
						{#if sub}
							<Button
								variant="outline"
								size="sm"
								onclick={handlePortal}
								disabled={portalMutation.isPending}
							>
								{#if portalMutation.isPending}
									<Loader2 class="mr-1.5 h-3 w-3 animate-spin" />
								{:else}
									<ArrowUpRight class="mr-1.5 h-3 w-3" />
								{/if}
								Manage Billing
							</Button>
						{/if}
						{#if !usage.planTier || usage.planTier === 'starter'}
							<Button
								size="sm"
								onclick={() => handleCheckout(usage.planTier ? 'builder' : 'starter')}
								disabled={checkoutMutation.isPending}
							>
								{#if checkoutMutation.isPending}
									<Loader2 class="mr-1.5 h-3 w-3 animate-spin" />
								{:else}
									<Zap class="mr-1.5 h-3 w-3" />
								{/if}
								{usage.planTier ? 'Upgrade to Builder' : 'Upgrade'}
							</Button>
						{/if}
					</div>
				</div>

				{#if sub?.currentPeriodEnd}
					<p class="text-xs text-muted-foreground">
						Current period ends {new Date(sub.currentPeriodEnd).toLocaleDateString('en-US', {
							month: 'long',
							day: 'numeric',
							year: 'numeric'
						})}
					</p>
				{/if}
			</CardContent>
		</Card>

		<!-- Usage Dashboard -->
		<Card>
			<CardHeader>
				<CardTitle class="flex items-center gap-2">
					<BarChart3 class="h-4 w-4" />
					Usage This Period
				</CardTitle>
				<CardDescription>
					Your current usage against plan limits. Counters reset each billing period.
				</CardDescription>
			</CardHeader>
			<CardContent class="space-y-6">
				<!-- Projects -->
				<div class="space-y-2">
					<div class="flex items-center justify-between text-sm">
						<div class="flex items-center gap-2 font-medium">
							<FolderKanban class="h-4 w-4 text-muted-foreground" />
							Projects
						</div>
						<span class="tabular-nums text-muted-foreground">
							{usage.projectCount} / {usage.limits.maxProjects}
						</span>
					</div>
					<div class="h-2 rounded-full bg-muted overflow-hidden">
						<div
							class="h-full rounded-full transition-all {usageColor(projectPct)}"
							style="width: {projectPct}%"
						></div>
					</div>
				</div>

				<Separator />

				<!-- Signals -->
				<div class="space-y-2">
					<div class="flex items-center justify-between text-sm">
						<div class="flex items-center gap-2 font-medium">
							<Signal class="h-4 w-4 text-muted-foreground" />
							Signals Ingested
						</div>
						<span class="tabular-nums text-muted-foreground">
							{usage.signalsUsed} / {usage.limits.maxSignalsPerMonth}
						</span>
					</div>
					<div class="h-2 rounded-full bg-muted overflow-hidden">
						<div
							class="h-full rounded-full transition-all {usageColor(signalPct)}"
							style="width: {signalPct}%"
						></div>
					</div>
				</div>

				<Separator />

				<!-- PRs -->
				<div class="space-y-2">
					<div class="flex items-center justify-between text-sm">
						<div class="flex items-center gap-2 font-medium">
							<GitPullRequest class="h-4 w-4 text-muted-foreground" />
							PRs Generated
						</div>
						<span class="tabular-nums text-muted-foreground">
							{usage.prsUsed} / {usage.limits.maxPrsPerMonth}
						</span>
					</div>
					<div class="h-2 rounded-full bg-muted overflow-hidden">
						<div
							class="h-full rounded-full transition-all {usageColor(prPct)}"
							style="width: {prPct}%"
						></div>
					</div>
				</div>
			</CardContent>
		</Card>

		<!-- AI Costs -->
		{#if llmUsageQuery.isLoading}
			<Card>
				<CardHeader>
					<Skeleton class="h-5 w-32" />
					<Skeleton class="h-4 w-56 mt-1" />
				</CardHeader>
				<CardContent class="space-y-4">
					<Skeleton class="h-16 w-full" />
				</CardContent>
			</Card>
		{:else if llmUsageQuery.data}
			{@const ai = llmUsageQuery.data}
			<Card>
				<CardHeader>
					<CardTitle class="flex items-center gap-2">
						<BrainCircuit class="h-4 w-4" />
						AI Costs
					</CardTitle>
					<CardDescription>
						Token usage, costs, and latency across all projects in your organization.
					</CardDescription>
				</CardHeader>
				<CardContent>
					<div class="grid grid-cols-2 gap-4">
						<div class="rounded-lg border p-4 space-y-1">
							<div class="flex items-center gap-1.5 text-xs text-muted-foreground">
								<DollarSign class="h-3 w-3" />
								Total Cost
							</div>
							<p class="text-2xl font-semibold tabular-nums">
								{formatCost(ai.total_cost_usd)}
							</p>
						</div>

						<div class="rounded-lg border p-4 space-y-1">
							<div class="flex items-center gap-1.5 text-xs text-muted-foreground">
								<Hash class="h-3 w-3" />
								API Calls
							</div>
							<p class="text-2xl font-semibold tabular-nums">
								{ai.total_calls.toLocaleString()}
							</p>
						</div>

						<div class="rounded-lg border p-4 space-y-1">
							<div class="flex items-center gap-1.5 text-xs text-muted-foreground">
								<BarChart3 class="h-3 w-3" />
								Tokens (in / out)
							</div>
							<p class="text-lg font-semibold tabular-nums">
								{formatTokens(ai.total_tokens_in)} / {formatTokens(ai.total_tokens_out)}
							</p>
						</div>

						<div class="rounded-lg border p-4 space-y-1">
							<div class="flex items-center gap-1.5 text-xs text-muted-foreground">
								<Clock class="h-3 w-3" />
								Latency (avg / p95)
							</div>
							<p class="text-lg font-semibold tabular-nums">
								{formatLatency(ai.avg_latency_ms)} / {formatLatency(ai.p95_latency_ms)}
							</p>
						</div>
					</div>
				</CardContent>
			</Card>
		{/if}

		<!-- Plan Comparison -->
		{#if !usage.planTier || usage.planTier === 'starter'}
			<Card>
				<CardHeader>
					<CardTitle class="text-sm font-medium text-muted-foreground">
						Plan Limits Comparison
					</CardTitle>
				</CardHeader>
				<CardContent>
					<div class="grid grid-cols-4 gap-4 text-sm">
						<div></div>
						<div class="text-center font-medium text-muted-foreground">Free</div>
						<div class="text-center font-medium">Starter</div>
						<div class="text-center font-medium">Builder</div>

						<div class="text-muted-foreground">Projects</div>
						<div class="text-center">1</div>
						<div class="text-center">3</div>
						<div class="text-center">10</div>

						<div class="text-muted-foreground">Signals/mo</div>
						<div class="text-center">20</div>
						<div class="text-center">100</div>
						<div class="text-center">500</div>

						<div class="text-muted-foreground">PRs/mo</div>
						<div class="text-center">3</div>
						<div class="text-center">10</div>
						<div class="text-center">50</div>
					</div>
				</CardContent>
			</Card>
		{/if}
	{/if}
</div>
