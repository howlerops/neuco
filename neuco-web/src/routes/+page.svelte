<script lang="ts">
	import { browser } from '$app/environment';
	import { goto } from '$app/navigation';
	import { authStore } from '$lib/stores/auth.svelte';
	import { apiClient } from '$lib/api/client';
	import { Button } from '$lib/components/ui/button';
	import { Badge } from '$lib/components/ui/badge';
	import {
		ArrowRight,
		Zap,
		GitPullRequest,
		Brain,
		FileText,
		MessageSquare,
		ChevronRight,
		Check,
		Star,
		Menu,
		X
	} from 'lucide-svelte';
	import IntegrationIcon from '$lib/components/icons/integrations.svelte';
	import type { IntegrationIconName } from '$lib/components/icons/integrations.svelte';

	import type { OnboardingStatus } from '$lib/api/types-compat';

	interface MeResponse {
		user: { id: string; githubLogin: string; email: string; avatarUrl: string };
		orgs: { id: string; name: string; slug: string; plan: string }[];
	}

	let redirecting = $state(false);
	let mobileMenuOpen = $state(false);

	// Redirect authenticated users to their dashboard
	$effect(() => {
		if (!browser) return;
		if (!authStore.isAuthenticated) return;

		redirecting = true;
		Promise.all([
			apiClient.get<MeResponse>('/api/v1/auth/me'),
			apiClient.get<OnboardingStatus>('/api/v1/onboarding/status').catch(() => null)
		])
			.then(([data, onboarding]) => {
				if (data.orgs && data.orgs.length > 0) {
					const org = data.orgs[0];
					authStore.setOrg(org as any);
					if (onboarding && !onboarding.isComplete) {
						goto('/onboarding');
					} else {
						goto(`/${org.slug}/dashboard`);
					}
				} else {
					redirecting = false;
				}
			})
			.catch(() => {
				authStore.clearAuth();
				redirecting = false;
			});
	});

	const pipelineSteps = [
		{
			step: '01',
			icon: MessageSquare,
			title: 'Collect signals',
			description: 'Ingest feedback from Slack, Intercom, Jira, Linear, Gong, and CSV uploads. Every voice gets heard.'
		},
		{
			step: '02',
			icon: Brain,
			title: 'AI synthesis',
			description: 'Cluster related signals, identify themes, and surface actionable insights with context-aware AI.'
		},
		{
			step: '03',
			icon: FileText,
			title: 'Generate specs',
			description: 'Turn insights into detailed technical specifications ready for engineering — automatically.'
		},
		{
			step: '04',
			icon: GitPullRequest,
			title: 'Ship code',
			description: 'Generate implementation code and open a GitHub PR. Review, merge, deploy.'
		}
	];

	const integrations: { name: string; color: string; icon: IntegrationIconName }[] = [
		{ name: 'Slack', color: '#4A154B', icon: 'slack' },
		{ name: 'Intercom', color: '#1F8DED', icon: 'intercom' },
		{ name: 'Linear', color: '#5E6AD2', icon: 'linear' },
		{ name: 'Jira', color: '#0052CC', icon: 'jira' },
		{ name: 'Gong', color: '#7A57D1', icon: 'gong' },
		{ name: 'HubSpot', color: '#FF7A59', icon: 'hubspot' },
		{ name: 'CSV', color: '#059669', icon: 'csv' }
	];

	const plans = [
		{
			name: 'Starter',
			price: 49,
			description: 'For small teams getting started with product intelligence.',
			features: [
				'Up to 500 signals/mo',
				'2 integrations',
				'AI synthesis & clustering',
				'Spec generation',
				'Email support'
			],
			cta: 'Start free trial',
			highlighted: false
		},
		{
			name: 'Builder',
			price: 149,
			description: 'For growing teams that ship fast.',
			features: [
				'Up to 5,000 signals/mo',
				'All integrations',
				'AI synthesis & clustering',
				'Spec + code generation',
				'GitHub PR automation',
				'Priority support'
			],
			cta: 'Start free trial',
			highlighted: true
		},
		{
			name: 'Team',
			price: 399,
			description: 'For teams that need full pipeline control.',
			features: [
				'Unlimited signals',
				'All integrations',
				'AI synthesis & clustering',
				'Spec + code generation',
				'GitHub PR automation',
				'Custom AI models',
				'SSO & RBAC',
				'Dedicated support'
			],
			cta: 'Contact sales',
			highlighted: false
		}
	];

	const testimonials = [
		{
			quote: "Neuco turned weeks of feedback triage into hours. We shipped 3x more features last quarter.",
			author: 'Sarah Chen',
			role: 'VP Product',
			company: 'Series B SaaS'
		},
		{
			quote: "The signal-to-spec pipeline is magic. Our PMs finally have time for strategy instead of ticket grooming.",
			author: 'Marcus Rivera',
			role: 'Head of Engineering',
			company: 'Growth-stage Fintech'
		},
		{
			quote: "We plugged in Intercom and Slack on Monday. By Friday we had specs and PRs for our top 5 customer requests.",
			author: 'Aisha Patel',
			role: 'CPO',
			company: 'Enterprise Platform'
		}
	];
</script>

<svelte:head>
	<title>Neuco — Turn customer feedback into shipped code</title>
	<meta name="description" content="AI-native product intelligence platform. Ingest signals from Slack, Intercom, Jira, Linear, and more. Synthesize insights. Generate specs and code. Ship faster." />
</svelte:head>

{#if redirecting}
	<div class="flex min-h-screen items-center justify-center">
		<div class="h-8 w-8 animate-spin rounded-full border-4 border-muted border-t-primary"></div>
	</div>
{:else}
	<div class="min-h-screen bg-background text-foreground">
		<!-- Navigation -->
		<nav class="sticky top-0 z-50 border-b border-border/50 bg-background/80 backdrop-blur-lg">
			<div class="mx-auto flex h-16 max-w-6xl items-center justify-between px-6">
				<a href="/" class="text-xl font-bold tracking-tight">Neuco</a>

				<!-- Desktop nav -->
				<div class="hidden items-center gap-8 md:flex">
					<a href="#how-it-works" class="text-sm text-muted-foreground transition-colors hover:text-foreground">How it works</a>
					<a href="#integrations" class="text-sm text-muted-foreground transition-colors hover:text-foreground">Integrations</a>
					<a href="#pricing" class="text-sm text-muted-foreground transition-colors hover:text-foreground">Pricing</a>
					<a href="/login" class="text-sm text-muted-foreground transition-colors hover:text-foreground">Sign in</a>
					<Button href="/login" size="sm">Get started <ArrowRight class="ml-1 h-3.5 w-3.5" /></Button>
				</div>

				<!-- Mobile menu toggle -->
				<button
					class="md:hidden p-2 text-muted-foreground hover:text-foreground"
					onclick={() => mobileMenuOpen = !mobileMenuOpen}
					aria-label="Toggle menu"
				>
					{#if mobileMenuOpen}
						<X class="h-5 w-5" />
					{:else}
						<Menu class="h-5 w-5" />
					{/if}
				</button>
			</div>

			<!-- Mobile nav -->
			{#if mobileMenuOpen}
				<div class="border-t border-border/50 bg-background px-6 py-4 md:hidden">
					<div class="flex flex-col gap-4">
						<a href="#how-it-works" class="text-sm text-muted-foreground" onclick={() => mobileMenuOpen = false}>How it works</a>
						<a href="#integrations" class="text-sm text-muted-foreground" onclick={() => mobileMenuOpen = false}>Integrations</a>
						<a href="#pricing" class="text-sm text-muted-foreground" onclick={() => mobileMenuOpen = false}>Pricing</a>
						<a href="/login" class="text-sm text-muted-foreground">Sign in</a>
						<Button href="/login" size="sm" class="w-fit">Get started</Button>
					</div>
				</div>
			{/if}
		</nav>

		<!-- Hero -->
		<section class="relative overflow-hidden">
			<!-- Subtle gradient background -->
			<div class="pointer-events-none absolute inset-0 bg-gradient-to-b from-muted/30 via-transparent to-transparent"></div>

			<div class="relative mx-auto max-w-6xl px-6 pb-24 pt-20 sm:pb-32 sm:pt-28">
				<div class="mx-auto max-w-3xl text-center">
					<Badge variant="secondary" class="mb-6 px-3 py-1 text-xs font-medium">
						AI-native product intelligence
					</Badge>

					<h1 class="text-4xl font-bold tracking-tight sm:text-5xl lg:text-6xl">
						Turn customer feedback into
						<span class="bg-gradient-to-r from-foreground via-foreground/80 to-foreground bg-clip-text">shipped code</span>
					</h1>

					<p class="mx-auto mt-6 max-w-2xl text-lg text-muted-foreground sm:text-xl">
						Neuco ingests signals from every channel, synthesizes them with AI, generates specs, and opens pull requests — so your team ships what customers actually need.
					</p>

					<div class="mt-10 flex flex-col items-center gap-4 sm:flex-row sm:justify-center">
						<Button href="/login" size="lg" class="w-full sm:w-auto">
							Start free trial
							<ArrowRight class="ml-2 h-4 w-4" />
						</Button>
						<Button href="#how-it-works" variant="outline" size="lg" class="w-full sm:w-auto">
							See how it works
						</Button>
					</div>

					<p class="mt-4 text-sm text-muted-foreground">No credit card required. 14-day free trial.</p>
				</div>

				<!-- Pipeline visualization -->
				<div class="mx-auto mt-20 max-w-4xl">
					<div class="rounded-xl border border-border bg-card p-1 shadow-lg">
						<div class="rounded-lg bg-muted/50 p-6 sm:p-10">
							<div class="grid grid-cols-2 gap-3 sm:grid-cols-4 sm:gap-4">
								{#each ['Signals', 'Insights', 'Specs', 'Code'] as label, i}
									<div class="flex flex-col items-center gap-2">
										<div class="flex h-12 w-12 items-center justify-center rounded-lg bg-background shadow-sm border border-border sm:h-14 sm:w-14">
											{#if i === 0}<MessageSquare class="h-5 w-5 text-muted-foreground sm:h-6 sm:w-6" />
											{:else if i === 1}<Brain class="h-5 w-5 text-muted-foreground sm:h-6 sm:w-6" />
											{:else if i === 2}<FileText class="h-5 w-5 text-muted-foreground sm:h-6 sm:w-6" />
											{:else}<GitPullRequest class="h-5 w-5 text-muted-foreground sm:h-6 sm:w-6" />
											{/if}
										</div>
										<span class="text-xs font-medium text-muted-foreground sm:text-sm">{label}</span>
									</div>
									{#if i < 3}
										<div class="hidden items-center sm:flex absolute" style="display: none;">
											<ChevronRight class="h-4 w-4 text-muted-foreground/50" />
										</div>
									{/if}
								{/each}
							</div>
							<div class="mt-4 flex items-center justify-center gap-1">
								{#each Array(3) as _, i}
									<div class="hidden sm:block h-px w-full bg-gradient-to-r from-border via-muted-foreground/20 to-border"></div>
								{/each}
							</div>
						</div>
					</div>
				</div>
			</div>
		</section>

		<!-- How it works -->
		<section id="how-it-works" class="border-t border-border/50 bg-muted/20 py-24 sm:py-32">
			<div class="mx-auto max-w-6xl px-6">
				<div class="mx-auto max-w-2xl text-center">
					<h2 class="text-3xl font-bold tracking-tight sm:text-4xl">From signal to shipped in four steps</h2>
					<p class="mt-4 text-lg text-muted-foreground">
						No more lost feedback. No more stale backlogs. Neuco closes the loop from customer voice to production code.
					</p>
				</div>

				<div class="mx-auto mt-16 grid max-w-5xl gap-8 sm:grid-cols-2 lg:gap-12">
					{#each pipelineSteps as step}
						<div class="group relative">
							<div class="flex gap-4">
								<div class="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg border border-border bg-background text-sm font-bold text-muted-foreground shadow-sm transition-colors group-hover:border-foreground/20 group-hover:text-foreground">
									{step.step}
								</div>
								<div>
									<h3 class="text-lg font-semibold">{step.title}</h3>
									<p class="mt-1.5 text-sm leading-relaxed text-muted-foreground">{step.description}</p>
								</div>
							</div>
						</div>
					{/each}
				</div>
			</div>
		</section>

		<!-- Integrations -->
		<section id="integrations" class="border-t border-border/50 py-24 sm:py-32">
			<div class="mx-auto max-w-6xl px-6">
				<div class="mx-auto max-w-2xl text-center">
					<h2 class="text-3xl font-bold tracking-tight sm:text-4xl">Connect your tools</h2>
					<p class="mt-4 text-lg text-muted-foreground">
						Pull signals from the tools your team already uses. Set up in minutes, not days.
					</p>
				</div>

				<div class="mx-auto mt-16 grid max-w-4xl grid-cols-2 gap-4 sm:grid-cols-3 lg:grid-cols-4">
					{#each integrations as integration}
						<div class="flex flex-col items-center gap-3 rounded-xl border border-border bg-card p-6 transition-colors hover:border-foreground/20 hover:bg-accent/50">
							<div class="flex h-12 w-12 items-center justify-center rounded-lg bg-muted/60">
								<IntegrationIcon name={integration.icon} class="h-7 w-7" />
							</div>
							<span class="text-sm font-medium">{integration.name}</span>
						</div>
					{/each}
				</div>

				<p class="mt-8 text-center text-sm text-muted-foreground">
					More integrations coming soon — GitHub, Zendesk, and more.
				</p>
			</div>
		</section>

		<!-- Social proof -->
		<section class="border-t border-border/50 bg-muted/20 py-24 sm:py-32">
			<div class="mx-auto max-w-6xl px-6">
				<div class="mx-auto max-w-2xl text-center">
					<h2 class="text-3xl font-bold tracking-tight sm:text-4xl">Trusted by product teams</h2>
					<p class="mt-4 text-lg text-muted-foreground">
						Teams use Neuco to close the gap between what customers say and what gets shipped.
					</p>
				</div>

				<div class="mx-auto mt-16 grid max-w-5xl gap-8 md:grid-cols-3">
					{#each testimonials as testimonial}
						<div class="rounded-xl border border-border bg-card p-6">
							<div class="flex gap-1">
								{#each Array(5) as _}
									<Star class="h-4 w-4 fill-foreground text-foreground" />
								{/each}
							</div>
							<blockquote class="mt-4 text-sm leading-relaxed text-muted-foreground">
								"{testimonial.quote}"
							</blockquote>
							<div class="mt-4 border-t border-border pt-4">
								<p class="text-sm font-medium">{testimonial.author}</p>
								<p class="text-xs text-muted-foreground">{testimonial.role}, {testimonial.company}</p>
							</div>
						</div>
					{/each}
				</div>
			</div>
		</section>

		<!-- Pricing -->
		<section id="pricing" class="border-t border-border/50 py-24 sm:py-32">
			<div class="mx-auto max-w-6xl px-6">
				<div class="mx-auto max-w-2xl text-center">
					<h2 class="text-3xl font-bold tracking-tight sm:text-4xl">Simple, transparent pricing</h2>
					<p class="mt-4 text-lg text-muted-foreground">
						Start free. Scale as your team grows. No hidden fees.
					</p>
				</div>

				<div class="mx-auto mt-16 grid max-w-5xl gap-8 lg:grid-cols-3">
					{#each plans as plan}
						<div
							class="relative flex flex-col rounded-xl border p-8 {plan.highlighted
								? 'border-foreground/20 bg-card shadow-lg ring-1 ring-foreground/5'
								: 'border-border bg-card'}"
						>
							{#if plan.highlighted}
								<Badge class="absolute -top-3 left-1/2 -translate-x-1/2">Most popular</Badge>
							{/if}
							<div>
								<h3 class="text-lg font-semibold">{plan.name}</h3>
								<p class="mt-1 text-sm text-muted-foreground">{plan.description}</p>
							</div>
							<div class="mt-6">
								<span class="text-4xl font-bold">${plan.price}</span>
								<span class="text-sm text-muted-foreground">/mo</span>
							</div>
							<ul class="mt-8 flex-1 space-y-3">
								{#each plan.features as feature}
									<li class="flex items-start gap-2 text-sm">
										<Check class="mt-0.5 h-4 w-4 shrink-0 text-foreground" />
										<span class="text-muted-foreground">{feature}</span>
									</li>
								{/each}
							</ul>
							<div class="mt-8">
								<Button
									href="/login"
									variant={plan.highlighted ? 'default' : 'outline'}
									class="w-full"
								>
									{plan.cta}
								</Button>
							</div>
						</div>
					{/each}
				</div>
			</div>
		</section>

		<!-- CTA -->
		<section class="border-t border-border/50 bg-muted/20 py-24 sm:py-32">
			<div class="mx-auto max-w-6xl px-6">
				<div class="mx-auto max-w-2xl text-center">
					<h2 class="text-3xl font-bold tracking-tight sm:text-4xl">Ready to ship what customers need?</h2>
					<p class="mt-4 text-lg text-muted-foreground">
						Start your free trial today. Go from customer feedback to pull requests in minutes.
					</p>
					<div class="mt-10 flex flex-col items-center gap-4 sm:flex-row sm:justify-center">
						<Button href="/login" size="lg">
							Get started free
							<ArrowRight class="ml-2 h-4 w-4" />
						</Button>
						<Button href="#how-it-works" variant="outline" size="lg">
							Learn more
						</Button>
					</div>
				</div>
			</div>
		</section>

		<!-- Footer -->
		<footer class="border-t border-border/50 py-12">
			<div class="mx-auto max-w-6xl px-6">
				<div class="flex flex-col items-center justify-between gap-8 sm:flex-row">
					<div>
						<span class="text-lg font-bold tracking-tight">Neuco</span>
						<p class="mt-1 text-sm text-muted-foreground">AI-native product intelligence.</p>
					</div>

					<div class="flex flex-wrap items-center justify-center gap-6 text-sm text-muted-foreground">
						<a href="#how-it-works" class="transition-colors hover:text-foreground">How it works</a>
						<a href="#pricing" class="transition-colors hover:text-foreground">Pricing</a>
						<a href="/login" class="transition-colors hover:text-foreground">Sign in</a>
						<a href="/terms" class="transition-colors hover:text-foreground">Terms</a>
						<a href="/privacy" class="transition-colors hover:text-foreground">Privacy</a>
					</div>
				</div>

				<div class="mt-8 border-t border-border/50 pt-8 text-center text-xs text-muted-foreground">
					&copy; {new Date().getFullYear()} Neuco. All rights reserved.
				</div>
			</div>
		</footer>
	</div>
{/if}
