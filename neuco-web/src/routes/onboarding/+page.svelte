<script lang="ts">
	import { goto } from '$app/navigation';
	import { authStore } from '$lib/stores/auth.svelte';
	import { useOnboardingStatus, useCompleteStep, useSkipOnboarding } from '$lib/api/queries/onboarding';
	import { useOrgs } from '$lib/api/queries/organizations';
	import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '$lib/components/ui/card';
	import { Button } from '$lib/components/ui/button';
	import { Skeleton } from '$lib/components/ui/skeleton';
	import { toast } from 'svelte-sonner';
	import {
		Loader2,
		Sparkles,
		Building2,
		FolderKanban,
		Upload,
		Zap,
		PartyPopper,
		ChevronRight,
		SkipForward,
		Check
	} from 'lucide-svelte';
	import type { OnboardingStep } from '$lib/api/types-compat';

	// Auth guard
	$effect(() => {
		if (!authStore.isAuthenticated) {
			goto('/login');
		}
	});

	const statusQuery = useOnboardingStatus();
	const orgsQuery = useOrgs();
	const completeStepMutation = useCompleteStep();
	const skipMutation = useSkipOnboarding();

	// Redirect if onboarding already complete
	$effect(() => {
		if (statusQuery.data?.isComplete && orgsQuery.data?.length) {
			const org = authStore.currentOrg ?? orgsQuery.data[0];
			goto(`/${org.slug}/dashboard`);
		}
	});

	// Set org from data if not already set
	$effect(() => {
		if (orgsQuery.data?.length && !authStore.currentOrg) {
			authStore.setOrg(orgsQuery.data[0]);
		}
	});

	const steps: {
		key: OnboardingStep;
		title: string;
		description: string;
		icon: typeof Sparkles;
		actionLabel: string;
	}[] = [
		{
			key: 'welcome',
			title: 'Welcome to Neuco',
			description:
				"Neuco turns product signals — bugs, feature requests, user feedback — into generated code and pull requests. Let's get you set up.",
			icon: Sparkles,
			actionLabel: 'Get Started'
		},
		{
			key: 'org',
			title: 'Your Organization',
			description:
				"We've created an organization for you. This is where your team's projects, signals, and generated code live. You can rename it or invite teammates later.",
			icon: Building2,
			actionLabel: 'Continue'
		},
		{
			key: 'project',
			title: 'Create a Project',
			description:
				'Projects connect to a GitHub repo and collect signals. Head to your dashboard to create your first project, or continue the tour.',
			icon: FolderKanban,
			actionLabel: 'Continue'
		},
		{
			key: 'signal',
			title: 'Upload Signals',
			description:
				'Signals are the raw inputs — bug reports, feature requests, or user feedback. Upload a CSV, connect an integration, or use the API. You can do this from any project page.',
			icon: Upload,
			actionLabel: 'Continue'
		},
		{
			key: 'synthesis',
			title: 'AI Synthesis',
			description:
				'Once you have signals, Neuco synthesizes them into feature candidates, generates specs, and creates pull requests — all automatically.',
			icon: Zap,
			actionLabel: 'Continue'
		},
		{
			key: 'done',
			title: "You're All Set!",
			description:
				"You're ready to start using Neuco. Create a project, upload some signals, and watch the magic happen.",
			icon: PartyPopper,
			actionLabel: 'Go to Dashboard'
		}
	];

	// Determine current step index based on completed steps
	const currentStepIndex = $derived.by(() => {
		const completed = statusQuery.data?.completedSteps ?? [];
		for (let i = 0; i < steps.length; i++) {
			if (!completed.includes(steps[i].key)) {
				return i;
			}
		}
		return steps.length - 1;
	});

	const currentStep = $derived(steps[currentStepIndex]);
	const progress = $derived(Math.round(((currentStepIndex) / steps.length) * 100));

	async function handleNext() {
		try {
			await completeStepMutation.mutateAsync({ step: currentStep.key });

			// If this was the last step, mark onboarding complete and redirect
			if (currentStep.key === 'done') {
				await skipMutation.mutateAsync(); // marks as complete
				const org = authStore.currentOrg ?? orgsQuery.data?.[0];
				if (org) {
					goto(`/${org.slug}/dashboard`);
				}
			}
		} catch (err) {
			toast.error('Failed to save progress', {
				description: err instanceof Error ? err.message : 'An unexpected error occurred.'
			});
		}
	}

	async function handleSkip() {
		try {
			await skipMutation.mutateAsync();
			const org = authStore.currentOrg ?? orgsQuery.data?.[0];
			if (org) {
				goto(`/${org.slug}/dashboard`);
			}
		} catch (err) {
			toast.error('Failed to skip onboarding');
		}
	}
</script>

<svelte:head>
	<title>Getting Started — Neuco</title>
</svelte:head>

<div class="flex min-h-screen flex-col items-center justify-center bg-background p-6">
	{#if statusQuery.isLoading}
		<Card class="w-full max-w-lg">
			<CardHeader>
				<Skeleton class="h-6 w-48" />
				<Skeleton class="h-4 w-64 mt-2" />
			</CardHeader>
			<CardContent class="space-y-4">
				<Skeleton class="h-32 w-full" />
			</CardContent>
		</Card>
	{:else}
		<div class="w-full max-w-lg space-y-6">
			<!-- Progress bar -->
			<div class="space-y-2">
				<div class="flex items-center justify-between text-xs text-muted-foreground">
					<span>Step {currentStepIndex + 1} of {steps.length}</span>
					<span>{progress}%</span>
				</div>
				<div class="h-1.5 rounded-full bg-muted overflow-hidden">
					<div
						class="h-full rounded-full bg-primary transition-all duration-500"
						style="width: {progress}%"
					></div>
				</div>
			</div>

			<!-- Step card -->
			<Card class="relative overflow-hidden">
				<div class="absolute top-0 left-0 right-0 h-1 bg-primary"></div>
				<CardHeader class="text-center pt-8">
					<div class="mx-auto mb-4 flex h-16 w-16 items-center justify-center rounded-full bg-primary/10">
						<currentStep.icon class="h-8 w-8 text-primary" />
					</div>
					<CardTitle class="text-xl">{currentStep.title}</CardTitle>
					<CardDescription class="mt-2 text-sm leading-relaxed">
						{currentStep.description}
					</CardDescription>
				</CardHeader>
				<CardContent class="flex flex-col items-center gap-3 pb-8">
					<Button
						onclick={handleNext}
						disabled={completeStepMutation.isPending || skipMutation.isPending}
						class="gap-2 min-w-[180px]"
					>
						{#if completeStepMutation.isPending}
							<Loader2 class="h-4 w-4 animate-spin" />
						{:else if currentStep.key === 'done'}
							<Check class="h-4 w-4" />
						{:else}
							<ChevronRight class="h-4 w-4" />
						{/if}
						{currentStep.actionLabel}
					</Button>
					{#if currentStep.key !== 'done'}
						<Button
							variant="ghost"
							size="sm"
							onclick={handleSkip}
							disabled={skipMutation.isPending}
							class="text-muted-foreground"
						>
							{#if skipMutation.isPending}
								<Loader2 class="mr-1.5 h-3 w-3 animate-spin" />
							{:else}
								<SkipForward class="mr-1.5 h-3 w-3" />
							{/if}
							Skip tour
						</Button>
					{/if}
				</CardContent>
			</Card>

			<!-- Step indicators -->
			<div class="flex justify-center gap-2">
				{#each steps as step, i (step.key)}
					<div
						class="h-2 w-2 rounded-full transition-colors {i < currentStepIndex
							? 'bg-primary'
							: i === currentStepIndex
								? 'bg-primary/70'
								: 'bg-muted'}"
					></div>
				{/each}
			</div>
		</div>
	{/if}
</div>
