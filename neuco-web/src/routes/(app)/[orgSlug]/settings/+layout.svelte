<script lang="ts">
	import { page } from '$app/stores';
	import { cn } from '$lib/utils';

	let { children } = $props();

	const orgSlug = $derived($page.params.orgSlug);

	const tabs = $derived([
		{
			label: 'General',
			href: `/${orgSlug}/settings`
		},
		{
			label: 'Profile',
			href: `/${orgSlug}/settings/profile`
		},
		{
			label: 'Members',
			href: `/${orgSlug}/settings/members`
		},
		{
			label: 'Billing & Usage',
			href: `/${orgSlug}/settings/billing`
		},
		{
			label: 'Audit Log',
			href: `/${orgSlug}/settings/audit-log`
		}
	]);

	const currentPath = $derived($page.url.pathname);

	function isActive(href: string): boolean {
		// Exact match for the general settings tab to avoid it matching all settings pages
		if (href === `/${orgSlug}/settings`) {
			return currentPath === `/${orgSlug}/settings` || currentPath === `/${orgSlug}/settings/`;
		}
		return currentPath.startsWith(href);
	}
</script>

<div class="p-6 space-y-6">
	<div>
		<h1 class="text-2xl font-semibold tracking-tight">Settings</h1>
		<p class="text-muted-foreground mt-1">
			Manage your organization settings, members, and view activity.
		</p>
	</div>

	<nav class="flex gap-1 border-b border-border" aria-label="Settings navigation">
		{#each tabs as tab (tab.href)}
			<a
				href={tab.href}
				class={cn(
					'px-4 py-2 text-sm font-medium border-b-2 -mb-px transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 rounded-t-sm',
					isActive(tab.href)
						? 'border-primary text-primary'
						: 'border-transparent text-muted-foreground hover:text-foreground hover:border-border'
				)}
				aria-current={isActive(tab.href) ? 'page' : undefined}
			>
				{tab.label}
			</a>
		{/each}
	</nav>

	{@render children()}
</div>
