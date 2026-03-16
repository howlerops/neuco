<script lang="ts">
	import { Badge } from '$lib/components/ui/badge';
	import { Button } from '$lib/components/ui/button';
	import { Card, CardContent, CardHeader, CardTitle } from '$lib/components/ui/card';
	import { connectSSE } from '$lib/api/useSSE';
	import type { SandboxSession, SandboxSessionStatus } from '$lib/api/types-compat';
	import {
		CheckCircle2,
		Clock,
		Loader2,
		RotateCcw,
		Square,
		Terminal,
		XCircle
	} from 'lucide-svelte';
	import { cn } from '$lib/utils';

	interface Props {
		projectId: string;
		sessionId: string;
		session: SandboxSession;
		onStop: () => void;
		onRetry: () => void;
	}

	let { projectId, sessionId, session, onStop, onRetry }: Props = $props();

	let currentSession = $state<SandboxSession>({} as SandboxSession);
	let logs = $state<string[]>([]);
	let streamError = $state('');
	let logContainer = $state<HTMLDivElement | null>(null);

	$effect(() => {
		currentSession = { ...session };
		logs = session.agentLog?.split('\n').filter(Boolean) ?? [];
	});

	const isRunning = $derived(
		currentSession.status === 'running' ||
			currentSession.status === 'pending' ||
			currentSession.status === 'provisioning' ||
			currentSession.status === 'validating'
	);
	const isFailed = $derived(
		currentSession.status === 'failed' ||
			currentSession.status === 'cancelled' ||
			currentSession.status === 'timed_out'
	);

	$effect(() => {
		if (!projectId || !sessionId) return;

		const stream = connectSSE(`/api/v1/projects/${projectId}/sessions/${sessionId}/stream`, {
			onOpen: () => {
				streamError = '';
			},
			onError: () => {
				streamError = 'Live stream disconnected. Waiting for updates...';
			},
			onMessage: (event) => {
				try {
					const payload = JSON.parse(event.data) as {
						session?: SandboxSession;
						log?: string;
						agentLog?: string;
					};

					if (payload.session) {
						currentSession = { ...currentSession, ...payload.session };
					}
					if (payload.log) {
						logs = [...logs, payload.log];
					} else if (payload.agentLog) {
						logs = payload.agentLog.split('\n').filter(Boolean);
					}
				} catch {
					if (event.data) {
						logs = [...logs, event.data];
					}
				}
			}
		});

		return () => stream.close();
	});

	$effect(() => {
		if (logContainer) {
			logContainer.scrollTop = logContainer.scrollHeight;
		}
	});

	function statusClass(status: SandboxSessionStatus) {
		switch (status) {
			case 'completed':
				return 'bg-green-100 text-green-800 border-green-200';
			case 'running':
			case 'provisioning':
			case 'validating':
				return 'bg-blue-100 text-blue-800 border-blue-200';
			case 'failed':
			case 'cancelled':
			case 'timed_out':
				return 'bg-red-100 text-red-800 border-red-200';
			default:
				return 'bg-gray-100 text-gray-800 border-gray-200';
		}
	}
</script>

<Card>
	<CardHeader class="pb-3">
		<CardTitle class="flex items-center justify-between gap-3 text-base">
			<div class="flex items-center gap-2">
				<Terminal class="h-4 w-4" />
				Live Session
			</div>
			<Badge class={cn('capitalize border', statusClass(currentSession.status))}>
				{#if isRunning}
					<Loader2 class="mr-1 h-3 w-3 animate-spin" />
				{:else if currentSession.status === 'completed'}
					<CheckCircle2 class="mr-1 h-3 w-3" />
				{:else if isFailed}
					<XCircle class="mr-1 h-3 w-3" />
				{:else}
					<Clock class="mr-1 h-3 w-3" />
				{/if}
				{currentSession.status}
			</Badge>
		</CardTitle>
	</CardHeader>

	<CardContent class="space-y-4">
		<div class="grid grid-cols-3 gap-3 text-sm">
			<div>
				<p class="text-muted-foreground">Tokens Used</p>
				<p class="font-medium">{currentSession.tokensUsed ?? 0}</p>
			</div>
			<div>
				<p class="text-muted-foreground">Cost</p>
				<p class="font-medium">${(currentSession.costUsd ?? 0).toFixed(4)}</p>
			</div>
			<div>
				<p class="text-muted-foreground">Retry Count</p>
				<p class="font-medium">{currentSession.retryCount}</p>
			</div>
		</div>

		<div
			bind:this={logContainer}
			class="max-h-[400px] overflow-auto rounded-md border bg-zinc-950 p-3 font-mono text-xs text-zinc-100"
		>
			{#if logs.length === 0}
				<div class="text-zinc-400">Waiting for logs...</div>
			{:else}
				{#each logs as line}
					<div class="whitespace-pre-wrap break-words">{line}</div>
				{/each}
			{/if}
		</div>

		{#if currentSession.errorMessage}
			<p class="text-sm text-red-600">{currentSession.errorMessage}</p>
		{/if}
		{#if streamError}
			<p class="text-sm text-muted-foreground">{streamError}</p>
		{/if}

		<div class="flex items-center gap-2">
			{#if isRunning}
				<Button variant="destructive" size="sm" onclick={onStop}>
					<Square class="h-3.5 w-3.5" />
					Stop
				</Button>
			{/if}
			{#if isFailed}
				<Button variant="outline" size="sm" onclick={onRetry}>
					<RotateCcw class="h-3.5 w-3.5" />
					Retry
				</Button>
			{/if}
		</div>
	</CardContent>
</Card>
