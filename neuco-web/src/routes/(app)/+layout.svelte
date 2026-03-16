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
		FolderKanban,
		Bell,
		Menu,
		X
	} from 'lucide-svelte';
	import { cn } from '$lib/utils';
	import { identifyUser, resetUser, setGroup } from '$lib/analytics';
	import type { Organization } from '$lib/api/types-compat';
	import {
		useNotifications,
		useUnreadCount,
		useMarkNotificationRead,
		useMarkAllRead
	} from '$lib/api/queries/notifications';

	let { children } = $props();

	// Mobile sidebar state
	let mobileOpen = $state(false);

	// Close mobile sidebar on navigation
	$effect(() => {
		$page.url.pathname;
		mobileOpen = false;
	});

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

	// Notifications
	const getOrgId = () => authStore.currentOrg?.id ?? '';
	const unreadCountQuery = useUnreadCount(getOrgId);
	const notificationsQuery = useNotifications(getOrgId);
	const markReadMutation = useMarkNotificationRead(getOrgId);
	const markAllReadMutation = useMarkAllRead(getOrgId);
	const unreadCount = $derived(unreadCountQuery.data ?? 0);

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
		setGroup(org.id, { name: org.name, slug: org.slug, plan: org.plan });
		goto(`/${org.slug}/dashboard`);
	}

	// Derive user initials for avatar fallback
	const userInitials = $derived.by(() => {
		const user = meQuery.data?.user;
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

	// Sync user to auth store and identify for analytics when loaded
	$effect(() => {
		if (meQuery.data?.user) {
			const user = meQuery.data.user;
			authStore.setUser(user);
			identifyUser(user.id, {
				email: user.email,
				name: user.name || user.githubLogin
			});
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

	// Associate all events with the current organization for group analytics
	$effect(() => {
		if (authStore.currentOrg) {
			setGroup(authStore.currentOrg.id, {
				name: authStore.currentOrg.name,
				slug: authStore.currentOrg.slug,
				plan: authStore.currentOrg.plan
			});
		}
	});
</script>

{#snippet sidebarContent()}
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
{/snippet}

<div class="flex h-screen overflow-hidden bg-background">
	<!-- Mobile sidebar overlay -->
	{#if mobileOpen}
		<!-- Backdrop -->
		<div
			class="fixed inset-0 z-40 bg-black/50 md:hidden"
			role="button"
			tabindex="-1"
			onclick={() => (mobileOpen = false)}
			onkeydown={(e) => e.key === 'Escape' && (mobileOpen = false)}
		></div>
		<!-- Drawer -->
		<aside
			class="fixed inset-y-0 left-0 z-50 flex w-64 flex-col bg-sidebar-background shadow-xl md:hidden"
		>
			<button
				class="absolute right-3 top-4 rounded-md p-1 text-sidebar-foreground hover:bg-sidebar-accent"
				onclick={() => (mobileOpen = false)}
				aria-label="Close sidebar"
			>
				<X class="h-5 w-5" />
			</button>
			{@render sidebarContent()}
		</aside>
	{/if}

	<!-- Desktop sidebar -->
	<aside
		class="hidden w-64 flex-col border-r border-sidebar-border bg-sidebar-background md:flex"
	>
		{@render sidebarContent()}
	</aside>

	<!-- Main area -->
	<div class="flex flex-1 flex-col overflow-hidden">
		<!-- Top bar -->
		<header
			class="flex h-16 items-center justify-between border-b border-border bg-background px-4 md:px-6"
		>
			<div class="flex items-center gap-2">
				<!-- Mobile hamburger -->
				<button
					class="inline-flex h-9 w-9 items-center justify-center rounded-md text-sm font-medium hover:bg-accent hover:text-accent-foreground md:hidden"
					onclick={() => (mobileOpen = true)}
					aria-label="Open sidebar"
				>
					<Menu class="h-5 w-5" />
				</button>

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
			</div>

			<div class="flex items-center gap-2">
			<!-- Notifications -->
			<DropdownMenu>
				<DropdownMenuTrigger>
					<Button variant="ghost" class="relative h-9 w-9 p-0">
						<Bell class="h-5 w-5" />
						{#if unreadCount > 0}
							<span
								class="absolute -right-0.5 -top-0.5 flex h-4 min-w-4 items-center justify-center rounded-full bg-destructive px-1 text-[10px] font-medium text-destructive-foreground"
							>
								{unreadCount > 99 ? '99+' : unreadCount}
							</span>
						{/if}
					</Button>
				</DropdownMenuTrigger>
				<DropdownMenuContent align="end" class="w-80">
					<div class="flex items-center justify-between px-2 py-1.5">
						<DropdownMenuLabel class="p-0">Notifications</DropdownMenuLabel>
						{#if unreadCount > 0}
							<button
								class="text-xs text-muted-foreground hover:text-foreground transition-colors"
								onclick={() => markAllReadMutation.mutate()}
							>
								Mark all read
							</button>
						{/if}
					</div>
					<DropdownMenuSeparator />
					{#if notificationsQuery.data && notificationsQuery.data.length > 0}
						{#each notificationsQuery.data.slice(0, 10) as notif (notif.id)}
							<DropdownMenuItem
								class="flex flex-col items-start gap-1 py-2"
								onSelect={() => {
									if (!notif.readAt) {
										markReadMutation.mutate(notif.id);
									}
									if (notif.link) {
										goto(`/${orgSlug}${notif.link}`);
									}
								}}
							>
								<div class="flex w-full items-start gap-2">
									{#if !notif.readAt}
										<div class="mt-1.5 h-2 w-2 shrink-0 rounded-full bg-primary"></div>
									{:else}
										<div class="mt-1.5 h-2 w-2 shrink-0"></div>
									{/if}
									<div class="flex-1 min-w-0">
										<p class="text-sm font-medium truncate">{notif.title}</p>
										<p class="text-xs text-muted-foreground line-clamp-2">{notif.body}</p>
										<p class="text-xs text-muted-foreground mt-0.5">
											{new Date(notif.createdAt).toLocaleDateString()}
										</p>
									</div>
								</div>
							</DropdownMenuItem>
						{/each}
					{:else}
						<div class="px-4 py-6 text-center text-sm text-muted-foreground">
							No notifications yet
						</div>
					{/if}
				</DropdownMenuContent>
			</DropdownMenu>

			<!-- User menu -->
			<DropdownMenu>
				<DropdownMenuTrigger>
					<Button variant="ghost" class="h-9 w-9 rounded-full p-0">
						<Avatar class="h-8 w-8">
							<AvatarImage
								src={meQuery.data?.user?.avatarUrl ?? ''}
								alt={meQuery.data?.user?.name ?? 'User'}
							/>
							<AvatarFallback class="text-xs">{userInitials}</AvatarFallback>
						</Avatar>
					</Button>
				</DropdownMenuTrigger>
				<DropdownMenuContent align="end">
					<DropdownMenuLabel>
						<div class="flex flex-col space-y-1">
							<p class="text-sm font-medium">{meQuery.data?.user?.name ?? 'Loading...'}</p>
							<p class="text-xs text-muted-foreground">{meQuery.data?.user?.email ?? ''}</p>
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
		</div>
		</header>

		<!-- Page content -->
		<main class="flex-1 overflow-y-auto">
			{@render children()}
		</main>
	</div>
</div>
