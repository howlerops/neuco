import { createQuery } from '@tanstack/svelte-query';
import { apiClient } from '$lib/api/client';
import type { Framework, Styling } from '$lib/api/types-compat';

// ─── Types ────────────────────────────────────────────────────────────────────

export interface GitHubRepo {
	fullName: string;
	name: string;
	description: string | null;
	language: string | null;
	private: boolean;
	updatedAt: string;
}

export interface DetectFrameworkResult {
	name: string;
	framework: Framework;
	styling: Styling;
}

interface GitHubPackageJson {
	name?: string;
	dependencies?: Record<string, string>;
	devDependencies?: Record<string, string>;
	peerDependencies?: Record<string, string>;
}

interface GitHubContentsResponse {
	content?: string;
	encoding?: string;
}

// ─── URL Parsing ──────────────────────────────────────────────────────────────

/**
 * Parses a GitHub repo input into an "owner/repo" slug.
 * Accepts:
 *   - https://github.com/owner/repo
 *   - https://github.com/owner/repo.git
 *   - github.com/owner/repo
 *   - owner/repo
 */
export function parseGitHubRepo(input: string): string | null {
	const trimmed = input.trim();
	if (!trimmed) return null;

	// Full URL forms
	try {
		const url = new URL(trimmed.startsWith('http') ? trimmed : `https://${trimmed}`);
		if (url.hostname === 'github.com') {
			// pathname is like /owner/repo or /owner/repo.git
			const parts = url.pathname.replace(/^\//, '').replace(/\.git$/, '').split('/');
			if (parts.length >= 2 && parts[0] && parts[1]) {
				return `${parts[0]}/${parts[1]}`;
			}
		}
	} catch {
		// Not a URL — fall through to owner/repo check
	}

	// Plain "owner/repo" with no extra segments
	const slashParts = trimmed.replace(/\.git$/, '').split('/');
	if (slashParts.length === 2 && slashParts[0] && slashParts[1]) {
		return `${slashParts[0]}/${slashParts[1]}`;
	}

	return null;
}

/**
 * Converts a raw repo name into a human-readable project name.
 * e.g. "next.js" → "Next.js", "my-cool-app" → "My Cool App"
 */
export function repoNameToProjectName(repoName: string): string {
	return repoName
		.replace(/[-_]/g, ' ')
		.replace(/\b\w/g, (c) => c.toUpperCase());
}

// ─── Client-side detection from package.json ─────────────────────────────────

function detectFrameworkFromDeps(
	allDeps: Record<string, string>
): Framework {
	// Check in order of specificity — more specific frameworks first
	if ('next' in allDeps) return 'nextjs';
	if ('@nuxtjs/nuxt' in allDeps || 'nuxt' in allDeps) return 'nuxt';
	if ('@sveltejs/kit' in allDeps) return 'sveltekit';
	if ('svelte' in allDeps) return 'svelte';
	if ('@angular/core' in allDeps) return 'angular';
	if ('vue' in allDeps) return 'vue';
	if ('react' in allDeps || 'react-dom' in allDeps) return 'react';
	return 'other';
}

function detectStylingFromDeps(
	allDeps: Record<string, string>
): Styling {
	if ('tailwindcss' in allDeps) return 'tailwind';
	if ('styled-components' in allDeps) return 'styled_components';
	if ('@emotion/react' in allDeps || '@emotion/styled' in allDeps)
		return 'styled_components';
	if ('sass' in allDeps || 'node-sass' in allDeps) return 'sass';
	// CSS modules are typically zero-dep — infer from known bundler configs later
	return 'plain_css';
}

async function detectFromGitHubApi(ownerRepo: string): Promise<DetectFrameworkResult> {
	const [owner, repo] = ownerRepo.split('/');
	const repoName = repo;

	const apiUrl = `https://api.github.com/repos/${ownerRepo}/contents/package.json`;

	const response = await fetch(apiUrl, {
		headers: { Accept: 'application/vnd.github.v3+json' }
	});

	if (!response.ok) {
		// 404 means no package.json (non-JS project) or private repo
		throw new Error(
			response.status === 404
				? 'Repository not found or is private'
				: `GitHub API error: ${response.status}`
		);
	}

	const data: GitHubContentsResponse = await response.json();

	if (!data.content || data.encoding !== 'base64') {
		throw new Error('Could not read package.json content');
	}

	// GitHub returns base64 with newlines inserted
	const decoded = atob(data.content.replace(/\n/g, ''));
	const pkg: GitHubPackageJson = JSON.parse(decoded);

	const allDeps: Record<string, string> = {
		...pkg.dependencies,
		...pkg.devDependencies,
		...pkg.peerDependencies
	};

	const framework = detectFrameworkFromDeps(allDeps);
	const styling = detectStylingFromDeps(allDeps);
	const name = repoNameToProjectName(pkg.name ?? repoName);

	return { name, framework, styling };
}

// ─── Public API ───────────────────────────────────────────────────────────────

/**
 * Detects framework and styling for a GitHub repository.
 *
 * Strategy:
 *   1. Try the backend endpoint POST /api/v1/projects/detect-framework
 *   2. Fall back to direct GitHub API inspection of package.json
 *
 * Returns null if detection cannot be completed (private repo, no package.json, etc.)
 */
export async function detectFramework(
	ownerRepo: string
): Promise<DetectFrameworkResult | null> {
	// Detect framework via public GitHub API (client-side)
	try {
		return await detectFromGitHubApi(ownerRepo);
	} catch {
		return null;
	}
}

// ─── GitHub Repo Search ───────────────────────────────────────────────────────

export const githubRepoKeys = {
	search: (q: string) => ['github', 'repos', 'search', q] as const
};

/**
 * Searches the authenticated user's GitHub repositories via the backend.
 * Only enabled when `searchQuery` has 2 or more characters.
 * Falls back gracefully (returns an empty list) when no GitHub token is stored.
 */
export function useGitHubRepos(searchQuery: string) {
	return createQuery<GitHubRepo[]>(() => ({
		queryKey: githubRepoKeys.search(searchQuery),
		queryFn: () =>
			apiClient
				.get<GitHubRepo[]>(`/api/v1/auth/github/repos?q=${encodeURIComponent(searchQuery)}`)
				.catch(() => [] as GitHubRepo[]),
		enabled: searchQuery.length >= 2,
		staleTime: 30 * 1000,
		// Keep previous results visible while a new search is in-flight so the
		// dropdown doesn't flash empty between keystrokes.
		placeholderData: (prev) => prev
	}));
}
