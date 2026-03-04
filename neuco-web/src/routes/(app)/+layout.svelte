<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/stores';
	import { authStore } from '$lib/stores/auth.svelte';
	import { useOrgs } from '$lib/api/queries/organizations';
	import { useMe, useLogout } from '$lib/api/queries/auth.svelte';
	import { Button } from '$lib/components/ui/button';
	import { Avatar, AvatarImage, AvatarFallback } from '$lib/components/ui/avatar';
	import {
		DropdownMenu,
		DropdownMenuTrigger,
		DropdownMenuContent,
		DropdownMenuItem,
		DropdownMenuSeparator,
		DropdownMenuLabel
	} from '$lib/components/ui/dropdown-menu';
	import {
		LayoutDashboard,
		Settings,
		ChevronDown,
		LogOut,
		User,
		FolderKanban
	} from 'lucide-svelte';
	import { cn } from '$lib/utils';
	import type { Organization } from '$lib/api/types';

	let { children } = $props();

	// Auth guard
	$effect(() => {
		if (!authStore.isAuthenticated) {
			goto('/login');
		}
	});

	const meQuery = useMe();
	const orgsQuery = useOrgs();
	const logoutMutation = useLogout();

	// Derive current org slug from URL
	const orgSlug = $derived($page.params.orgSlug ?? '');

	// Nav items — org-level pages in sidebar, project-level pages in project layout tabs
	const navItems = $derived([
		{
			label: 'Dashboard',
			href: `/${orgSlug}/dashboard`,
			icon: LayoutDashboard,
			match: 'dashboard'
		},
		{
			label: 'Projects',
			href: `/${orgSlug}/projects`,
			icon: FolderKanban,
			match: 'projects'
		},
		{
			label: 'Settings',
			href: `/${orgSlug}/settings`,
			icon: Settings,
			match: 'settings'
		}
	]);

	function isActive(match: string): boolean {
		return $page.url.pathname.includes(`/${match}`);
	}

	function handleLogout() {
		logoutMutation.mutate();
	}

	function switchOrg(org: Organization) {
		authStore.switchOrg(org);
		goto(`/${org.slug}/dashboard`);
	}

	// Derive user initials for avatar fallback
	const userInitials = $derived.by(() => {
		const user = meQuery.data;
		if (!user) return 'U';
		// After the snake→camel transformer, githubLogin is already camelCase
		const displayName = user.githubLogin ?? user.name ?? user.email ?? 'U';
		if (!displayName) return 'U';
		const parts = displayName.split(/[\s_-]+/);
		return parts
			.slice(0, 2)
			.map((p: string) => p[0] ?? '')
			.join('')
			.toUpperCase() || 'U';
	});

	// Sync user to auth store when loaded
	$effect(() => {
		if (meQuery.data) {
			authStore.setUser(meQuery.data);
		}
	});

	$effect(() => {
		if (orgsQuery.data && orgsQuery.data.length > 0 && !authStore.currentOrg) {
			const storedOrgId = authStore.getStoredOrgId();
			const org =
				orgsQuery.data.find((o) => o.id === storedOrgId || o.slug === orgSlug) ??
				orgsQuery.data[0];
			authStore.setOrg(org);
		}
	});
</script>

<div class="flex h-screen overflow-hidden bg-background">
	<!-- Sidebar -->
	<aside
		class="flex w-64 flex-col border-r border-sidebar-border bg-sidebar-background"
	>
		<!-- Logo / Brand -->
		<div class="flex h-16 items-center border-b border-sidebar-border px-6">
			<span class="text-xl font-bold tracking-tight text-sidebar-foreground">Neuco</span>
		</div>

		<!-- Navigation -->
		<nav class="flex-1 overflow-y-auto px-3 py-4">
			<ul class="space-y-1">
				{#each navItems as item (item.match)}
					<li>
						<a
							href={item.href}
							class={cn(
								'flex items-center gap-3 rounded-md px-3 py-2 text-sm font-medium transition-colors',
								isActive(item.match)
									? 'bg-sidebar-accent text-sidebar-accent-foreground'
									: 'text-sidebar-foreground hover:bg-sidebar-accent hover:text-sidebar-accent-foreground'
							)}
						>
							<item.icon class="h-4 w-4 shrink-0" />
							{item.label}
						</a>
					</li>
				{/each}
			</ul>
		</nav>

		<!-- Org & User footer -->
		<div class="border-t border-sidebar-border p-3">
			<div class="flex items-center gap-2 rounded-md px-2 py-1.5 text-sm text-sidebar-foreground">
				<div class="h-2 w-2 rounded-full bg-green-500"></div>
				<span class="truncate font-medium">{authStore.currentOrg?.name ?? 'No org'}</span>
			</div>
		</div>
	</aside>

	<!-- Main area -->
	<div class="flex flex-1 flex-col overflow-hidden">
		<!-- Top bar -->
		<header
			class="flex h-16 items-center justify-between border-b border-border bg-background px-6"
		>
			<!-- Org switcher -->
			<DropdownMenu>
				<DropdownMenuTrigger>
					<Button variant="ghost" class="gap-2 font-medium">
						{authStore.currentOrg?.name ?? 'Select org'}
						<ChevronDown class="h-4 w-4 opacity-70" />
					</Button>
				</DropdownMenuTrigger>
				<DropdownMenuContent align="start">
					<DropdownMenuLabel>Organizations</DropdownMenuLabel>
					<DropdownMenuSeparator />
					{#if orgsQuery.data}
						{#each orgsQuery.data as org (org.id)}
							<DropdownMenuItem onSelect={() => switchOrg(org)}>
								<span
									class={cn(
										'flex items-center gap-2',
										authStore.currentOrg?.id === org.id && 'font-semibold'
									)}
								>
									{org.name}
								</span>
							</DropdownMenuItem>
						{/each}
					{/if}
				</DropdownMenuContent>
			</DropdownMenu>

			<!-- User menu -->
			<DropdownMenu>
				<DropdownMenuTrigger>
					<Button variant="ghost" class="h-9 w-9 rounded-full p-0">
						<Avatar class="h-8 w-8">
							<AvatarImage
								src={meQuery.data?.avatarUrl ?? ''}
								alt={meQuery.data?.name ?? 'User'}
							/>
							<AvatarFallback class="text-xs">{userInitials}</AvatarFallback>
						</Avatar>
					</Button>
				</DropdownMenuTrigger>
				<DropdownMenuContent align="end">
					<DropdownMenuLabel>
						<div class="flex flex-col space-y-1">
							<p class="text-sm font-medium">{meQuery.data?.name ?? 'Loading...'}</p>
							<p class="text-xs text-muted-foreground">{meQuery.data?.email ?? ''}</p>
						</div>
					</DropdownMenuLabel>
					<DropdownMenuSeparator />
					<DropdownMenuItem onSelect={() => goto(`/${orgSlug}/settings/profile`)}>
						<User class="mr-2 h-4 w-4" />
						Profile
					</DropdownMenuItem>
					<DropdownMenuItem onSelect={() => goto(`/${orgSlug}/settings`)}>
						<Settings class="mr-2 h-4 w-4" />
						Settings
					</DropdownMenuItem>
					<DropdownMenuSeparator />
					<DropdownMenuItem
						onSelect={handleLogout}
						class="text-destructive focus:text-destructive"
					>
						<LogOut class="mr-2 h-4 w-4" />
						Sign out
					</DropdownMenuItem>
				</DropdownMenuContent>
			</DropdownMenu>
		</header>

		<!-- Page content -->
		<main class="flex-1 overflow-y-auto">
			{@render children()}
		</main>
	</div>
</div>
