<script lang="ts">
	import { page } from '$app/stores';
	import { authStore } from '$lib/stores/auth.svelte';
	import {
		useMembers,
		useInviteMember,
		useUpdateMemberRole,
		useRemoveMember
	} from '$lib/api/queries/members';
	import {
		Table,
		TableBody,
		TableCell,
		TableHead,
		TableHeader,
		TableRow
	} from '$lib/components/ui/table';
	import { Avatar, AvatarImage, AvatarFallback } from '$lib/components/ui/avatar';
	import { Badge } from '$lib/components/ui/badge';
	import { Button } from '$lib/components/ui/button';
	import { Input } from '$lib/components/ui/input';
	import { Label } from '$lib/components/ui/label';
	import { Skeleton } from '$lib/components/ui/skeleton';
	import { Alert, AlertTitle, AlertDescription } from '$lib/components/ui/alert';
	import {
		Dialog,
		DialogContent,
		DialogHeader,
		DialogTitle,
		DialogDescription,
		DialogFooter
	} from '$lib/components/ui/dialog';
	import {
		DropdownMenu,
		DropdownMenuContent,
		DropdownMenuItem,
		DropdownMenuTrigger
	} from '$lib/components/ui/dropdown-menu';
	import { toast } from 'svelte-sonner';
	import {
		UserPlus,
		AlertCircle,
		RefreshCw,
		Users,
		ChevronDown,
		Trash2,
		Loader2,
		ShieldCheck,
		Crown
	} from 'lucide-svelte';
	import type { OrgRole, OrgMember } from '$lib/api/types';

	let orgId = $state(authStore.currentOrg?.id ?? '');
	$effect(() => {
		orgId = authStore.currentOrg?.id ?? '';
	});

	const membersQuery = $derived.by(() => useMembers(orgId));
	const inviteMutation = $derived.by(() => useInviteMember(orgId));
	const removeMutation = $derived.by(() => useRemoveMember(orgId));

	// Invite dialog state
	let inviteOpen = $state(false);
	let inviteEmail = $state('');
	let inviteRole = $state<OrgRole>('member');
	let inviteError = $state('');

	// Delete confirmation
	let removeConfirmId = $state<string | null>(null);

	// Role update mutations keyed by member id (created on demand)
	const members = $derived(membersQuery.data ?? []);

	const currentUserMember = $derived(
		members.find((m) => m.userId === authStore.currentUser?.id)
	);

	const canManageRoles = $derived(
		currentUserMember?.role === 'owner' || currentUserMember?.role === 'admin'
	);

	const roleLevels: Record<OrgRole, number> = {
		owner: 4,
		admin: 3,
		member: 2,
		viewer: 1
	};

	function canModifyMember(member: OrgMember): boolean {
		if (member.role === 'owner') return false;
		if (!currentUserMember) return false;
		return roleLevels[currentUserMember.role] > roleLevels[member.role];
	}

	function roleLabel(role: OrgRole): string {
		switch (role) {
			case 'owner': return 'Owner';
			case 'admin': return 'Admin';
			case 'member': return 'Member';
			case 'viewer': return 'Viewer';
			default: return role;
		}
	}

	function roleVariant(role: OrgRole): 'default' | 'secondary' | 'outline' {
		switch (role) {
			case 'owner': return 'default';
			case 'admin': return 'secondary';
			default: return 'outline';
		}
	}

	function roleClass(role: OrgRole): string {
		switch (role) {
			case 'owner': return 'bg-amber-100 text-amber-800 dark:bg-amber-900/30 dark:text-amber-400 border-transparent';
			case 'admin': return 'bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-400 border-transparent';
			case 'member': return 'bg-secondary text-secondary-foreground border-transparent';
			case 'viewer': return 'border-border text-muted-foreground';
			default: return 'border-border text-muted-foreground';
		}
	}

	function userInitials(name: string): string {
		const parts = name.trim().split(' ');
		return parts
			.slice(0, 2)
			.map((p) => p[0]?.toUpperCase() ?? '')
			.join('');
	}

	function formatDate(dateStr: string): string {
		return new Date(dateStr).toLocaleDateString('en-US', {
			month: 'short',
			day: 'numeric',
			year: 'numeric'
		});
	}

	function validateInviteEmail(): boolean {
		if (!inviteEmail.trim()) {
			inviteError = 'Email is required.';
			return false;
		}
		if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(inviteEmail.trim())) {
			inviteError = 'Please enter a valid email address.';
			return false;
		}
		inviteError = '';
		return true;
	}

	async function handleInvite() {
		if (!validateInviteEmail()) return;

		try {
			await inviteMutation.mutateAsync({
				email: inviteEmail.trim(),
				role: inviteRole
			});
			toast.success(`Invitation sent to ${inviteEmail.trim()}`);
			inviteOpen = false;
			inviteEmail = '';
			inviteRole = 'member';
		} catch (err) {
			toast.error('Failed to send invitation', {
				description: err instanceof Error ? err.message : 'An unexpected error occurred.'
			});
		}
	}

	async function handleRemove(memberId: string) {
		try {
			await removeMutation.mutateAsync(memberId);
			toast.success('Member removed');
			removeConfirmId = null;
		} catch (err) {
			toast.error('Failed to remove member', {
				description: err instanceof Error ? err.message : 'An unexpected error occurred.'
			});
		}
	}

	// Track in-flight role changes per member
	let pendingRoleChange = $state<Record<string, boolean>>({});

	async function handleRoleChange(member: OrgMember, newRole: OrgRole) {
		pendingRoleChange = { ...pendingRoleChange, [member.userId]: true };
		// Create a fresh mutation for this specific member
		const mutation = useUpdateMemberRole(orgId, member.userId);
		try {
			await mutation.mutateAsync({ role: newRole });
			toast.success(`${member.githubLogin}'s role updated to ${roleLabel(newRole)}`);
		} catch (err) {
			toast.error('Failed to update role', {
				description: err instanceof Error ? err.message : 'An unexpected error occurred.'
			});
		} finally {
			pendingRoleChange = { ...pendingRoleChange, [member.userId]: false };
		}
	}

	const assignableRoles: OrgRole[] = ['admin', 'member', 'viewer'];
</script>

<svelte:head>
	<title>Members — Neuco</title>
</svelte:head>

<div class="py-6 space-y-6">
	<!-- Header -->
	<div class="flex items-center justify-between">
		<div>
			<h2 class="text-lg font-semibold">Members</h2>
			<p class="text-muted-foreground text-sm mt-0.5">
				Manage who has access to your organization.
			</p>
		</div>
		<Button onclick={() => (inviteOpen = true)} class="gap-2" size="sm">
			<UserPlus class="h-4 w-4" />
			Invite Member
		</Button>
	</div>

	<!-- Members table -->
	{#if membersQuery.isLoading}
		<div class="rounded-lg border border-border overflow-hidden">
			<div class="bg-muted/30 px-4 py-3 flex gap-6">
				{#each Array(4) as _, i (i)}
					<Skeleton class="h-4 w-24"></Skeleton>
				{/each}
			</div>
			<div class="divide-y divide-border">
				{#each Array(4) as _, i (i)}
					<div class="px-4 py-4 flex items-center gap-4">
						<Skeleton class="h-9 w-9 rounded-full shrink-0"></Skeleton>
						<div class="flex-1 space-y-1.5">
							<Skeleton class="h-4 w-40"></Skeleton>
							<Skeleton class="h-3 w-32"></Skeleton>
						</div>
						<Skeleton class="h-5 w-16 rounded-full"></Skeleton>
						<Skeleton class="h-4 w-24"></Skeleton>
					</div>
				{/each}
			</div>
		</div>
	{:else if membersQuery.isError}
		<Alert variant="destructive">
			<AlertCircle class="h-4 w-4" />
			<AlertTitle>Failed to load members</AlertTitle>
			<AlertDescription class="flex items-center justify-between">
				<span>{membersQuery.error?.message ?? 'An unexpected error occurred.'}</span>
				<Button
					variant="outline"
					size="sm"
					onclick={() => membersQuery.refetch()}
					class="ml-4 shrink-0"
				>
					<RefreshCw class="mr-1.5 h-3 w-3" />
					Retry
				</Button>
			</AlertDescription>
		</Alert>
	{:else if members.length === 0}
		<div class="flex flex-col items-center justify-center rounded-xl border-2 border-dashed border-border py-16 text-center">
			<Users class="h-10 w-10 text-muted-foreground mb-3" />
			<h3 class="text-base font-semibold">No members yet</h3>
			<p class="text-sm text-muted-foreground mt-1 max-w-sm">
				Invite your team to collaborate on this organization.
			</p>
		</div>
	{:else}
		<div class="rounded-lg border border-border overflow-hidden">
			<Table>
				<TableHeader>
					<TableRow class="bg-muted/30">
						<TableHead>Member</TableHead>
						<TableHead class="w-[140px]">Role</TableHead>
						<TableHead class="w-[150px]">Joined</TableHead>
						<TableHead class="w-[80px]"></TableHead>
					</TableRow>
				</TableHeader>
				<TableBody>
					{#each members as member (member.userId)}
						<TableRow>
							<!-- Avatar + name -->
							<TableCell>
								<div class="flex items-center gap-3">
									<Avatar class="h-9 w-9 shrink-0">
										<AvatarImage
											src={member.avatarUrl}
											alt={member.githubLogin}
										/>
										<AvatarFallback class="text-xs bg-muted">
											{userInitials(member.githubLogin)}
										</AvatarFallback>
									</Avatar>
									<div class="min-w-0">
										<p class="text-sm font-medium truncate flex items-center gap-1.5">
											{member.githubLogin}
											{#if member.role === 'owner'}
												<Crown class="h-3 w-3 text-amber-500 shrink-0" />
											{/if}
										</p>
										<p class="text-xs text-muted-foreground truncate">
											{member.email}
										</p>
									</div>
								</div>
							</TableCell>

							<!-- Role -->
							<TableCell>
								{#if canManageRoles && canModifyMember(member)}
									<DropdownMenu>
										<DropdownMenuTrigger>
											<button
												type="button"
												class="inline-flex items-center gap-1 rounded-full border px-2.5 py-0.5 text-xs font-semibold transition-colors hover:opacity-80 {roleClass(member.role)}"
												disabled={pendingRoleChange[member.userId]}
											>
												{#if pendingRoleChange[member.userId]}
													<Loader2 class="h-3 w-3 animate-spin" />
												{:else}
													{roleLabel(member.role)}
													<ChevronDown class="h-3 w-3" />
												{/if}
											</button>
										</DropdownMenuTrigger>
										<DropdownMenuContent align="start">
											{#each assignableRoles as role (role)}
												<DropdownMenuItem
													onSelect={() => handleRoleChange(member, role)}
													class={member.role === role ? 'font-semibold' : ''}
												>
													{roleLabel(role)}
												</DropdownMenuItem>
											{/each}
										</DropdownMenuContent>
									</DropdownMenu>
								{:else}
									<span class="inline-flex items-center rounded-full border px-2.5 py-0.5 text-xs font-semibold {roleClass(member.role)}">
										{roleLabel(member.role)}
									</span>
								{/if}
							</TableCell>

							<!-- Joined date -->
							<TableCell class="text-sm text-muted-foreground">
								{formatDate(member.joinedAt ?? member.invitedAt)}
							</TableCell>

							<!-- Actions -->
							<TableCell>
								{#if canModifyMember(member) && member.userId !== authStore.currentUser?.id}
									{#if removeConfirmId === member.userId}
										<div class="flex items-center gap-1">
											<Button
												variant="destructive"
												size="sm"
												class="h-7 text-xs px-2"
												onclick={() => handleRemove(member.userId)}
												disabled={removeMutation.isPending}
											>
												{#if removeMutation.isPending}
													<Loader2 class="h-3 w-3 animate-spin" />
												{:else}
													Remove
												{/if}
											</Button>
											<Button
												variant="ghost"
												size="sm"
												class="h-7 text-xs px-2"
												onclick={() => (removeConfirmId = null)}
											>
												Cancel
											</Button>
										</div>
									{:else}
										<button
											type="button"
											class="inline-flex items-center justify-center h-7 w-7 p-0 rounded-md text-muted-foreground hover:text-destructive hover:bg-accent transition-colors"
											onclick={() => (removeConfirmId = member.userId)}
											title="Remove member"
										>
											<Trash2 class="h-3.5 w-3.5" />
										</button>
									{/if}
								{/if}
							</TableCell>
						</TableRow>
					{/each}
				</TableBody>
			</Table>
		</div>

		<p class="text-xs text-muted-foreground">
			{members.length} member{members.length === 1 ? '' : 's'} total
		</p>
	{/if}
</div>

<!-- Invite Member Dialog -->
<Dialog bind:open={inviteOpen}>
	<DialogContent class="sm:max-w-md">
		<DialogHeader>
			<DialogTitle>Invite Member</DialogTitle>
			<DialogDescription>
				Send an invitation to join your organization. They will receive an email with instructions.
			</DialogDescription>
		</DialogHeader>

		<div class="space-y-4 py-2">
			<div class="space-y-2">
				<Label for="invite-email">Email Address</Label>
				<Input
					id="invite-email"
					type="email"
					bind:value={inviteEmail}
					placeholder="teammate@company.com"
					class={inviteError ? 'border-destructive' : ''}
					onblur={validateInviteEmail}
					disabled={inviteMutation.isPending}
				/>
				{#if inviteError}
					<p class="text-xs text-destructive">{inviteError}</p>
				{/if}
			</div>

			<div class="space-y-2">
				<Label for="invite-role">Role</Label>
				<select
					id="invite-role"
					bind:value={inviteRole}
					class="w-full h-10 rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2"
					disabled={inviteMutation.isPending}
				>
					<option value="admin">Admin — Can manage project settings and members</option>
					<option value="member">Member — Can view and interact with projects</option>
					<option value="viewer">Viewer — Read-only access</option>
				</select>
			</div>
		</div>

		<DialogFooter>
			<Button
				variant="outline"
				onclick={() => { inviteOpen = false; inviteEmail = ''; inviteRole = 'member'; inviteError = ''; }}
				disabled={inviteMutation.isPending}
			>
				Cancel
			</Button>
			<Button
				onclick={handleInvite}
				disabled={inviteMutation.isPending}
				class="gap-2"
			>
				{#if inviteMutation.isPending}
					<Loader2 class="h-3.5 w-3.5 animate-spin" />
					Sending...
				{:else}
					<UserPlus class="h-3.5 w-3.5" />
					Send Invitation
				{/if}
			</Button>
		</DialogFooter>
	</DialogContent>
</Dialog>
