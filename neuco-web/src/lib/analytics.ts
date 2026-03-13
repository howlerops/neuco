import posthog from 'posthog-js';
import { browser } from '$app/environment';

const POSTHOG_KEY = import.meta.env.VITE_POSTHOG_KEY as string | undefined;
const POSTHOG_HOST = (import.meta.env.VITE_POSTHOG_HOST as string) || 'https://us.i.posthog.com';

let initialized = false;

export function initPostHog(): void {
	if (!browser || !POSTHOG_KEY || initialized) return;
	posthog.init(POSTHOG_KEY, {
		api_host: POSTHOG_HOST,
		capture_pageview: true,
		capture_pageleave: true,
		persistence: 'localStorage+cookie'
	});
	initialized = true;
}

export function identifyUser(userId: string, properties?: Record<string, unknown>): void {
	if (!browser || !initialized) return;
	posthog.identify(userId, properties);
}

export function setGroup(orgId: string, properties?: Record<string, unknown>): void {
	if (!browser || !initialized) return;
	posthog.group('organization', orgId, properties);
}

export function resetUser(): void {
	if (!browser || !initialized) return;
	posthog.reset();
}

export function capture(event: string, properties?: Record<string, unknown>): void {
	if (!browser || !initialized) return;
	posthog.capture(event, properties);
}

// ─── Typed event helpers ──────────────────────────────────────────────────────

export function trackSignup(provider: 'github' | 'google'): void {
	capture('user_signed_up', { provider });
}

export function trackLogin(provider: 'github' | 'google'): void {
	capture('user_logged_in', { provider });
}

export function trackProjectCreated(projectId: string, name: string): void {
	capture('project_created', { projectId, name });
}

export function trackSignalUploaded(projectId: string, count: number): void {
	capture('signal_uploaded', { projectId, signalCount: count });
}

export function trackSynthesisRun(projectId: string): void {
	capture('synthesis_run', { projectId });
}

export function trackSpecGenerated(projectId: string, candidateId: string): void {
	capture('spec_generated', { projectId, candidateId });
}

export function trackCodegenStarted(projectId: string, candidateId: string): void {
	capture('codegen_started', { projectId, candidateId });
}

export function trackPrCreated(projectId: string, generationId: string): void {
	capture('pr_created', { projectId, generationId });
}
