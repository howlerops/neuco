import { createQuery, createMutation, useQueryClient } from '@tanstack/svelte-query';
import { apiClient } from '$lib/api/client';
import type { OrgMember, InviteMemberPayload, UpdateMemberRolePayload } from '$lib/api/types';

// ─── Query Keys ───────────────────────────────────────────────────────────────

export const memberKeys = {
	all: (orgId: string) => ['organizations', orgId, 'members'] as const,
	lists: (orgId: string) => [...memberKeys.all(orgId), 'list'] as const,
	detail: (orgId: string, memberId: string) =>
		[...memberKeys.all(orgId), 'detail', memberId] as const
};

// ─── Hooks ────────────────────────────────────────────────────────────────────

export function useMembers(orgId: string) {
	return createQuery<OrgMember[]>(() => ({
		queryKey: memberKeys.lists(orgId),
		queryFn: () => apiClient.get<OrgMember[]>(`/api/v1/orgs/${orgId}/members`),
		enabled: !!orgId,
		staleTime: 60 * 1000
	}));
}

export function useInviteMember(orgId: string) {
	const queryClient = useQueryClient();
	return createMutation<OrgMember, Error, InviteMemberPayload>(() => ({
		mutationFn: (payload: InviteMemberPayload) =>
			apiClient.post<OrgMember>(`/api/v1/orgs/${orgId}/members`, payload),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: memberKeys.lists(orgId) });
		}
	}));
}

export function useUpdateMemberRole(orgId: string, memberId: string) {
	const queryClient = useQueryClient();
	return createMutation<OrgMember, Error, UpdateMemberRolePayload>(() => ({
		mutationFn: (payload: UpdateMemberRolePayload) =>
			apiClient.patch<OrgMember>(
				`/api/v1/orgs/${orgId}/members/${memberId}`,
				payload
			),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: memberKeys.lists(orgId) });
		}
	}));
}

export function useRemoveMember(orgId: string) {
	const queryClient = useQueryClient();
	return createMutation<void, Error, string>(() => ({
		mutationFn: (memberId: string) =>
			apiClient.delete<void>(`/api/v1/orgs/${orgId}/members/${memberId}`),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: memberKeys.lists(orgId) });
		}
	}));
}
