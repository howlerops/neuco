<script lang="ts">
	import { CheckCircle2, Circle, Loader2, XCircle } from 'lucide-svelte';
	import type { PipelineRun } from '$lib/api/types-compat';
	import { cn } from '$lib/utils';

	interface Props {
		pipeline: PipelineRun | null;
		provider: string;
		model: string;
	}

	let { pipeline, provider, model }: Props = $props();

	const steps = [
		'prepare_context',
		'provision_sandbox',
		'run_agent',
		'validate_output',
		'create_pr',
		'notify'
	] as const;

	type StepStatus = 'pending' | 'running' | 'completed' | 'failed';

	const taskMap = $derived(new Map((pipeline?.tasks ?? []).map((task) => [task.name, task])));

	function getStatus(step: string): StepStatus {
		const task = taskMap.get(step);
		if (!task) return 'pending';
		if (task.status === 'completed') return 'completed';
		if (task.status === 'running') return 'running';
		if (task.status === 'failed' || task.status === 'cancelled') return 'failed';
		return 'pending';
	}

	function statusClass(status: StepStatus) {
		return {
			completed: 'text-green-600',
			running: 'text-blue-600',
			failed: 'text-red-600',
			pending: 'text-muted-foreground'
		}[status];
	}
</script>

<div class="space-y-4">
	<div class="text-sm text-muted-foreground">
		Provider: <span class="font-medium text-foreground">{provider || '—'}</span>
		· Model: <span class="font-medium text-foreground">{model || '—'}</span>
	</div>

	<div class="flex items-center gap-2 overflow-x-auto pb-1">
		{#each steps as step, index}
			{@const status = getStatus(step)}
			<div class="flex items-center gap-2 min-w-max">
				<div class={cn('flex items-center gap-1.5 text-xs', statusClass(status))}>
					{#if status === 'completed'}
						<CheckCircle2 class="h-4 w-4" />
					{:else if status === 'running'}
						<Loader2 class="h-4 w-4 animate-spin" />
					{:else if status === 'failed'}
						<XCircle class="h-4 w-4" />
					{:else}
						<Circle class="h-4 w-4" />
					{/if}
					<span class="capitalize">{step.replaceAll('_', ' ')}</span>
				</div>

				{#if index < steps.length - 1}
					<div
						class={cn(
							'h-px w-8',
							status === 'completed' ? 'bg-green-500' : 'bg-border'
						)}
					></div>
				{/if}
			</div>
		{/each}
	</div>
</div>
