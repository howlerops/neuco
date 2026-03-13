<script lang="ts">
	import { page } from '$app/stores';
	import { authStore } from '$lib/stores/auth.svelte';
	import { useOrgAnalytics } from '$lib/api/queries/analytics';
	import { useProjects } from '$lib/api/queries/projects';
	import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '$lib/components/ui/card';
	import { Badge } from '$lib/components/ui/badge';
	import { Skeleton } from '$lib/components/ui/skeleton';
	import { Button } from '$lib/components/ui/button';
	import {
		Radio,
		Lightbulb,
		GitPullRequest,
		Activity,
		TrendingUp,
		Users,
		FolderKanban,
		BarChart3
	} from 'lucide-svelte';
	import type { OrgAnalytics, DailyCount, StatusCount, SourceCount, ProjectAnalytics, MemberActivity } from '$lib/api/types';

	const orgSlug = $derived($page.params.orgSlug);

	let orgId = $state(authStore.currentOrg?.id ?? '');
	$effect(() => {
		orgId = authStore.currentOrg?.id ?? '';
	});

	// Time range selector
	let selectedDays = $state(30);
	const timeRanges = [
		{ label: '7d', value: 7 },
		{ label: '30d', value: 30 },
		{ label: '90d', value: 90 }
	];

	const analyticsQuery = $derived.by(() => useOrgAnalytics(orgId, selectedDays));
	const analytics = $derived(analyticsQuery.data);

	const statCards = $derived([
		{
			label: 'Signals Ingested',
			value: analytics?.totalSignals ?? 0,
			icon: Radio,
			color: 'text-blue-500'
		},
		{
			label: 'Candidates Found',
			value: analytics?.totalCandidates ?? 0,
			icon: Lightbulb,
			color: 'text-amber-500'
		},
		{
			label: 'PRs Created',
			value: analytics?.totalPrs ?? 0,
			icon: GitPullRequest,
			color: 'text-green-500'
		},
		{
			label: 'Pipeline Success',
			value: analytics ? `${Math.round(analytics.pipelineSuccessRate * 100)}%` : '—',
			icon: Activity,
			color: 'text-purple-500'
		}
	]);

	// ── Chart helpers ──────────────────────────────────────────────────────────

	function sparklinePath(data: DailyCount[], width: number, height: number): string {
		if (!data || data.length === 0) return '';
		const max = Math.max(...data.map((d) => d.count), 1);
		const step = width / Math.max(data.length - 1, 1);
		return data
			.map((d, i) => {
				const x = i * step;
				const y = height - (d.count / max) * height;
				return `${i === 0 ? 'M' : 'L'} ${x.toFixed(1)} ${y.toFixed(1)}`;
			})
			.join(' ');
	}

	function areaPath(data: DailyCount[], width: number, height: number): string {
		if (!data || data.length === 0) return '';
		const line = sparklinePath(data, width, height);
		const step = width / Math.max(data.length - 1, 1);
		return `${line} L ${((data.length - 1) * step).toFixed(1)} ${height} L 0 ${height} Z`;
	}

	// Colors for pie/donut charts
	const statusColors: Record<string, string> = {
		completed: '#22c55e',
		failed: '#ef4444',
		running: '#3b82f6',
		pending: '#a3a3a3',
		new: '#3b82f6',
		specced: '#8b5cf6',
		in_progress: '#f59e0b',
		shipped: '#22c55e',
		rejected: '#ef4444'
	};

	const sourceColors = ['#3b82f6', '#8b5cf6', '#f59e0b', '#22c55e', '#ef4444', '#06b6d4', '#ec4899', '#f97316'];

	function donutSegments(
		items: { label: string; value: number; color: string }[],
		cx: number,
		cy: number,
		r: number,
		innerR: number
	): { d: string; color: string; label: string; value: number }[] {
		const total = items.reduce((s, i) => s + i.value, 0);
		if (total === 0) return [];
		let startAngle = -Math.PI / 2;
		return items.map((item) => {
			const angle = (item.value / total) * 2 * Math.PI;
			const endAngle = startAngle + angle;
			const largeArc = angle > Math.PI ? 1 : 0;
			const x1 = cx + r * Math.cos(startAngle);
			const y1 = cy + r * Math.sin(startAngle);
			const x2 = cx + r * Math.cos(endAngle);
			const y2 = cy + r * Math.sin(endAngle);
			const ix1 = cx + innerR * Math.cos(endAngle);
			const iy1 = cy + innerR * Math.sin(endAngle);
			const ix2 = cx + innerR * Math.cos(startAngle);
			const iy2 = cy + innerR * Math.sin(startAngle);
			const d = [
				`M ${x1.toFixed(2)} ${y1.toFixed(2)}`,
				`A ${r} ${r} 0 ${largeArc} 1 ${x2.toFixed(2)} ${y2.toFixed(2)}`,
				`L ${ix1.toFixed(2)} ${iy1.toFixed(2)}`,
				`A ${innerR} ${innerR} 0 ${largeArc} 0 ${ix2.toFixed(2)} ${iy2.toFixed(2)}`,
				'Z'
			].join(' ');
			startAngle = endAngle;
			return { d, color: item.color, label: item.label, value: item.value };
		});
	}

	const pipelineDonutData = $derived(
		(analytics?.pipelineBreakdown ?? []).map((b) => ({
			label: b.status,
			value: b.count,
			color: statusColors[b.status] ?? '#a3a3a3'
		}))
	);

	const candidateDonutData = $derived(
		(analytics?.candidateBreakdown ?? []).map((b) => ({
			label: b.status,
			value: b.count,
			color: statusColors[b.status] ?? '#a3a3a3'
		}))
	);

	const sourceBarData = $derived(analytics?.signalsBySource ?? []);
	const sourceBarMax = $derived(Math.max(...(sourceBarData.map((s) => s.count) ?? [0]), 1));

	function formatStatus(s: string): string {
		return s
			.split('_')
			.map((w) => w.charAt(0).toUpperCase() + w.slice(1))
			.join(' ');
	}
</script>

<svelte:head>
	<title>Dashboard — Neuco</title>
</svelte:head>

<div class="p-6 space-y-6 max-w-[1400px] mx-auto">
	<!-- Header + Time range -->
	<div class="flex items-center justify-between">
		<div>
			<h1 class="text-2xl font-bold tracking-tight">Dashboard</h1>
			<p class="text-muted-foreground mt-1">
				{#if authStore.currentOrg}
					{authStore.currentOrg.name} analytics
				{:else}
					Organization overview
				{/if}
			</p>
		</div>
		<div class="flex items-center gap-1 rounded-lg border border-border p-1">
			{#each timeRanges as range (range.value)}
				<button
					class="px-3 py-1.5 text-sm rounded-md transition-colors {selectedDays === range.value
						? 'bg-primary text-primary-foreground font-medium'
						: 'text-muted-foreground hover:text-foreground hover:bg-muted'}"
					onclick={() => (selectedDays = range.value)}
				>
					{range.label}
				</button>
			{/each}
		</div>
	</div>

	<!-- Summary stat cards -->
	<div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
		{#each statCards as stat, i (i)}
			<Card>
				<CardHeader class="flex flex-row items-center justify-between space-y-0 pb-2">
					<CardTitle class="text-sm font-medium text-muted-foreground">{stat.label}</CardTitle>
					<stat.icon class="h-4 w-4 {stat.color}" />
				</CardHeader>
				<CardContent>
					{#if analyticsQuery.isLoading}
						<Skeleton class="h-8 w-24"></Skeleton>
					{:else}
						<div class="text-3xl font-bold">{stat.value}</div>
					{/if}
				</CardContent>
			</Card>
		{/each}
	</div>

	<!-- Trend charts row -->
	<div class="grid gap-6 lg:grid-cols-2">
		<!-- Signal trend -->
		<Card>
			<CardHeader>
				<div class="flex items-center gap-2">
					<TrendingUp class="h-4 w-4 text-blue-500" />
					<CardTitle class="text-base">Signal Ingestion</CardTitle>
				</div>
				<CardDescription>Signals ingested per day</CardDescription>
			</CardHeader>
			<CardContent>
				{#if analyticsQuery.isLoading}
					<Skeleton class="h-[180px] w-full"></Skeleton>
				{:else if analytics?.signalTrend && analytics.signalTrend.length > 0}
					{@const data = analytics.signalTrend}
					{@const w = 500}
					{@const h = 160}
					{@const max = Math.max(...data.map((d) => d.count), 1)}
					<div class="w-full overflow-hidden">
						<svg viewBox="0 0 {w} {h + 24}" class="w-full h-auto">
							<!-- Grid lines -->
							{#each [0, 0.25, 0.5, 0.75, 1] as pct}
								<line
									x1="0"
									y1={h - pct * h}
									x2={w}
									y2={h - pct * h}
									stroke="currentColor"
									class="text-border"
									stroke-width="0.5"
								/>
							{/each}
							<!-- Area fill -->
							<path d={areaPath(data, w, h)} fill="url(#signalGrad)" opacity="0.3" />
							<!-- Line -->
							<path d={sparklinePath(data, w, h)} fill="none" stroke="#3b82f6" stroke-width="2" />
							<!-- X axis labels -->
							{#each data as d, i}
								{#if i % Math.max(Math.floor(data.length / 5), 1) === 0 || i === data.length - 1}
									<text
										x={i * (w / Math.max(data.length - 1, 1))}
										y={h + 16}
										text-anchor="middle"
										class="fill-muted-foreground"
										font-size="10"
									>
										{d.date.slice(5)}
									</text>
								{/if}
							{/each}
							<!-- Y max label -->
							<text x="4" y="12" class="fill-muted-foreground" font-size="10">{max}</text>
							<defs>
								<linearGradient id="signalGrad" x1="0" y1="0" x2="0" y2="1">
									<stop offset="0%" stop-color="#3b82f6" />
									<stop offset="100%" stop-color="#3b82f6" stop-opacity="0" />
								</linearGradient>
							</defs>
						</svg>
					</div>
				{:else}
					<div class="flex items-center justify-center h-[180px] text-muted-foreground text-sm">
						No signal data yet
					</div>
				{/if}
			</CardContent>
		</Card>

		<!-- Pipeline trend -->
		<Card>
			<CardHeader>
				<div class="flex items-center gap-2">
					<Activity class="h-4 w-4 text-purple-500" />
					<CardTitle class="text-base">Pipeline Runs</CardTitle>
				</div>
				<CardDescription>Pipeline executions per day</CardDescription>
			</CardHeader>
			<CardContent>
				{#if analyticsQuery.isLoading}
					<Skeleton class="h-[180px] w-full"></Skeleton>
				{:else if analytics?.pipelineTrend && analytics.pipelineTrend.length > 0}
					{@const data = analytics.pipelineTrend}
					{@const w = 500}
					{@const h = 160}
					{@const max = Math.max(...data.map((d) => d.count), 1)}
					<div class="w-full overflow-hidden">
						<svg viewBox="0 0 {w} {h + 24}" class="w-full h-auto">
							{#each [0, 0.25, 0.5, 0.75, 1] as pct}
								<line
									x1="0"
									y1={h - pct * h}
									x2={w}
									y2={h - pct * h}
									stroke="currentColor"
									class="text-border"
									stroke-width="0.5"
								/>
							{/each}
							<path d={areaPath(data, w, h)} fill="url(#pipelineGrad)" opacity="0.3" />
							<path d={sparklinePath(data, w, h)} fill="none" stroke="#8b5cf6" stroke-width="2" />
							{#each data as d, i}
								{#if i % Math.max(Math.floor(data.length / 5), 1) === 0 || i === data.length - 1}
									<text
										x={i * (w / Math.max(data.length - 1, 1))}
										y={h + 16}
										text-anchor="middle"
										class="fill-muted-foreground"
										font-size="10"
									>
										{d.date.slice(5)}
									</text>
								{/if}
							{/each}
							<text x="4" y="12" class="fill-muted-foreground" font-size="10">{max}</text>
							<defs>
								<linearGradient id="pipelineGrad" x1="0" y1="0" x2="0" y2="1">
									<stop offset="0%" stop-color="#8b5cf6" />
									<stop offset="100%" stop-color="#8b5cf6" stop-opacity="0" />
								</linearGradient>
							</defs>
						</svg>
					</div>
				{:else}
					<div class="flex items-center justify-center h-[180px] text-muted-foreground text-sm">
						No pipeline data yet
					</div>
				{/if}
			</CardContent>
		</Card>
	</div>

	<!-- Breakdown charts row -->
	<div class="grid gap-6 lg:grid-cols-3">
		<!-- Pipeline status breakdown -->
		<Card>
			<CardHeader>
				<CardTitle class="text-base">Pipeline Status</CardTitle>
				<CardDescription>Distribution of pipeline outcomes</CardDescription>
			</CardHeader>
			<CardContent>
				{#if analyticsQuery.isLoading}
					<Skeleton class="h-[200px] w-full"></Skeleton>
				{:else if pipelineDonutData.length > 0}
					{@const segments = donutSegments(pipelineDonutData, 80, 80, 70, 45)}
					<div class="flex items-center gap-4">
						<svg viewBox="0 0 160 160" class="w-32 h-32 shrink-0">
							{#each segments as seg}
								<path d={seg.d} fill={seg.color} />
							{/each}
						</svg>
						<div class="space-y-1.5 text-sm">
							{#each pipelineDonutData as item}
								<div class="flex items-center gap-2">
									<span class="inline-block w-2.5 h-2.5 rounded-full" style="background:{item.color}"></span>
									<span class="text-muted-foreground">{formatStatus(item.label)}</span>
									<span class="font-medium ml-auto">{item.value}</span>
								</div>
							{/each}
						</div>
					</div>
				{:else}
					<div class="flex items-center justify-center h-[200px] text-muted-foreground text-sm">
						No pipeline data
					</div>
				{/if}
			</CardContent>
		</Card>

		<!-- Candidate status breakdown -->
		<Card>
			<CardHeader>
				<CardTitle class="text-base">Candidate Status</CardTitle>
				<CardDescription>Feature candidate workflow stages</CardDescription>
			</CardHeader>
			<CardContent>
				{#if analyticsQuery.isLoading}
					<Skeleton class="h-[200px] w-full"></Skeleton>
				{:else if candidateDonutData.length > 0}
					{@const segments = donutSegments(candidateDonutData, 80, 80, 70, 45)}
					<div class="flex items-center gap-4">
						<svg viewBox="0 0 160 160" class="w-32 h-32 shrink-0">
							{#each segments as seg}
								<path d={seg.d} fill={seg.color} />
							{/each}
						</svg>
						<div class="space-y-1.5 text-sm">
							{#each candidateDonutData as item}
								<div class="flex items-center gap-2">
									<span class="inline-block w-2.5 h-2.5 rounded-full" style="background:{item.color}"></span>
									<span class="text-muted-foreground">{formatStatus(item.label)}</span>
									<span class="font-medium ml-auto">{item.value}</span>
								</div>
							{/each}
						</div>
					</div>
				{:else}
					<div class="flex items-center justify-center h-[200px] text-muted-foreground text-sm">
						No candidate data
					</div>
				{/if}
			</CardContent>
		</Card>

		<!-- Signals by source -->
		<Card>
			<CardHeader>
				<div class="flex items-center gap-2">
					<BarChart3 class="h-4 w-4 text-blue-500" />
					<CardTitle class="text-base">Signals by Source</CardTitle>
				</div>
				<CardDescription>Where your signals come from</CardDescription>
			</CardHeader>
			<CardContent>
				{#if analyticsQuery.isLoading}
					<Skeleton class="h-[200px] w-full"></Skeleton>
				{:else if sourceBarData.length > 0}
					<div class="space-y-3">
						{#each sourceBarData as source, i (source.source)}
							<div>
								<div class="flex items-center justify-between text-sm mb-1">
									<span class="capitalize text-muted-foreground">{source.source.replace(/_/g, ' ')}</span>
									<span class="font-medium">{source.count}</span>
								</div>
								<div class="h-2 rounded-full bg-muted overflow-hidden">
									<div
										class="h-full rounded-full transition-all"
										style="width: {(source.count / sourceBarMax) * 100}%; background: {sourceColors[i % sourceColors.length]}"
									></div>
								</div>
							</div>
						{/each}
					</div>
				{:else}
					<div class="flex items-center justify-center h-[200px] text-muted-foreground text-sm">
						No signal data
					</div>
				{/if}
			</CardContent>
		</Card>
	</div>

	<!-- Project breakdown & Team activity -->
	<div class="grid gap-6 lg:grid-cols-2">
		<!-- Project breakdown -->
		<Card>
			<CardHeader>
				<div class="flex items-center gap-2">
					<FolderKanban class="h-4 w-4 text-amber-500" />
					<CardTitle class="text-base">Project Breakdown</CardTitle>
				</div>
				<CardDescription>Activity across your projects</CardDescription>
			</CardHeader>
			<CardContent>
				{#if analyticsQuery.isLoading}
					<div class="space-y-3">
						{#each Array(3) as _, i (i)}
							<Skeleton class="h-16 w-full rounded-lg"></Skeleton>
						{/each}
					</div>
				{:else if analytics?.projects && analytics.projects.length > 0}
					<div class="space-y-3">
						{#each analytics.projects as project (project.id)}
							<div class="rounded-lg border border-border p-3">
								<div class="flex items-center justify-between mb-2">
									<a
										href="/{orgSlug}/projects/{project.id}"
										class="font-medium text-sm hover:underline"
									>
										{project.name}
									</a>
								</div>
								<div class="grid grid-cols-4 gap-2 text-xs">
									<div>
										<span class="text-muted-foreground">Signals</span>
										<p class="font-semibold">{project.signalCount}</p>
									</div>
									<div>
										<span class="text-muted-foreground">Candidates</span>
										<p class="font-semibold">{project.candidateCount}</p>
									</div>
									<div>
										<span class="text-muted-foreground">PRs</span>
										<p class="font-semibold">{project.prCount}</p>
									</div>
									<div>
										<span class="text-muted-foreground">Pipelines</span>
										<p class="font-semibold">{project.pipelineCount}</p>
									</div>
								</div>
							</div>
						{/each}
					</div>
				{:else}
					<div class="flex flex-col items-center justify-center py-10 text-center">
						<FolderKanban class="h-10 w-10 text-muted-foreground mb-3" />
						<p class="text-sm font-medium">No projects yet</p>
						<p class="text-xs text-muted-foreground mt-1">
							Create a project to start tracking signals.
						</p>
					</div>
				{/if}
			</CardContent>
		</Card>

		<!-- Team activity -->
		<Card>
			<CardHeader>
				<div class="flex items-center gap-2">
					<Users class="h-4 w-4 text-green-500" />
					<CardTitle class="text-base">Team Activity</CardTitle>
				</div>
				<CardDescription>What your team has been up to</CardDescription>
			</CardHeader>
			<CardContent>
				{#if analyticsQuery.isLoading}
					<div class="space-y-3">
						{#each Array(3) as _, i (i)}
							<Skeleton class="h-12 w-full rounded-lg"></Skeleton>
						{/each}
					</div>
				{:else if analytics?.teamActivity && analytics.teamActivity.length > 0}
					<div class="overflow-x-auto">
						<table class="w-full text-sm">
							<thead>
								<tr class="border-b border-border">
									<th class="text-left py-2 font-medium text-muted-foreground">Member</th>
									<th class="text-right py-2 font-medium text-muted-foreground">Signals</th>
									<th class="text-right py-2 font-medium text-muted-foreground">Specs</th>
									<th class="text-right py-2 font-medium text-muted-foreground">PRs</th>
								</tr>
							</thead>
							<tbody>
								{#each analytics.teamActivity as member (member.userId)}
									<tr class="border-b border-border/50">
										<td class="py-2.5 font-medium">{member.displayName}</td>
										<td class="py-2.5 text-right tabular-nums">{member.signalsUploaded}</td>
										<td class="py-2.5 text-right tabular-nums">{member.specsGenerated}</td>
										<td class="py-2.5 text-right tabular-nums">{member.prsCreated}</td>
									</tr>
								{/each}
							</tbody>
						</table>
					</div>
				{:else}
					<div class="flex flex-col items-center justify-center py-10 text-center">
						<Users class="h-10 w-10 text-muted-foreground mb-3" />
						<p class="text-sm font-medium">No team activity yet</p>
						<p class="text-xs text-muted-foreground mt-1">
							Activity will appear as your team uses Neuco.
						</p>
					</div>
				{/if}
			</CardContent>
		</Card>
	</div>
</div>
