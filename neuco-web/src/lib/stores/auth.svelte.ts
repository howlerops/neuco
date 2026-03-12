import { browser } from '$app/environment';
import type { Organization, User } from '$lib/api/types';

// ─── Persisted Token State ────────────────────────────────────────────────────

function readFromStorage(key: string): string | null {
	if (!browser) return null;
	return localStorage.getItem(key);
}

function writeToStorage(key: string, value: string | null): void {
	if (!browser) return;
	if (value === null) {
		localStorage.removeItem(key);
	} else {
		localStorage.setItem(key, value);
	}
}

// ─── Auth Store (Svelte 5 Runes) ─────────────────────────────────────────────

function createAuthStore() {
	let accessToken = $state<string | null>(readFromStorage('access_token'));
	let currentUser = $state<User | null>(null);
	let currentOrg = $state<Organization | null>(null);

	const isAuthenticated = $derived(Boolean(accessToken && accessToken.length > 0));

	function setTokens(access: string, expiresIn?: number): void {
		accessToken = access;
		writeToStorage('access_token', access);
		if (expiresIn) {
			const expiresAt = Date.now() + expiresIn * 1000;
			writeToStorage('access_token_expires_at', String(expiresAt));
		}
	}

	function clearAuth(): void {
		accessToken = null;
		currentUser = null;
		currentOrg = null;
		writeToStorage('access_token', null);
		writeToStorage('access_token_expires_at', null);
		writeToStorage('current_org_id', null);
	}

	function setUser(user: User): void {
		currentUser = user;
	}

	function setOrg(org: Organization): void {
		currentOrg = org;
		writeToStorage('current_org_id', org.id);
	}

	function switchOrg(org: Organization): void {
		setOrg(org);
	}

	function getStoredOrgId(): string | null {
		return readFromStorage('current_org_id');
	}

	return {
		get accessToken() {
			return accessToken;
		},
		get currentUser() {
			return currentUser;
		},
		get currentOrg() {
			return currentOrg;
		},
		get isAuthenticated() {
			return isAuthenticated;
		},
		setTokens,
		clearAuth,
		setUser,
		setOrg,
		switchOrg,
		getStoredOrgId
	};
}

export const authStore = createAuthStore();
