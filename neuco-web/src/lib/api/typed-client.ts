/**
 * Fully-typed API client powered by openapi-fetch + openapi-typescript.
 *
 * Usage:
 *   import { api } from '$lib/api/typed-client';
 *   const { data, error } = await api.GET('/api/v1/projects/{projectId}/agent-config', {
 *     params: { path: { projectId: '...' } }
 *   });
 *   // data is fully typed as components["schemas"]["AgentConfig"]
 *
 * Run `pnpm generate:api` after modifying openapi.yaml to regenerate types.
 */
import createClient from 'openapi-fetch';
import type { paths } from './v1';
import { browser } from '$app/environment';
import { goto } from '$app/navigation';

function getBaseUrl(): string {
  if (typeof import.meta !== 'undefined' && import.meta.env?.VITE_API_BASE_URL) {
    return import.meta.env.VITE_API_BASE_URL as string;
  }
  return 'http://localhost:8080';
}

function getAccessToken(): string | null {
  if (!browser) return null;
  return localStorage.getItem('access_token');
}

function getTokenExpiry(): number | null {
  if (!browser) return null;
  const val = localStorage.getItem('access_token_expires_at');
  return val ? Number(val) : null;
}

function isTokenExpiringSoon(): boolean {
  const expiresAt = getTokenExpiry();
  if (!expiresAt) return false;
  return Date.now() > expiresAt - 5 * 60 * 1000;
}

let refreshPromise: Promise<boolean> | null = null;

async function silentRefresh(): Promise<boolean> {
  if (refreshPromise) return refreshPromise;
  refreshPromise = doRefresh();
  try {
    return await refreshPromise;
  } finally {
    refreshPromise = null;
  }
}

async function doRefresh(): Promise<boolean> {
  try {
    const baseUrl = getBaseUrl();
    const response = await fetch(`${baseUrl}/api/v1/auth/refresh`, {
      method: 'POST',
      credentials: 'include',
    });
    if (!response.ok) return false;
    const data = await response.json();
    if (data.access_token) {
      localStorage.setItem('access_token', data.access_token);
      if (data.expires_in) {
        const expiresAt = Date.now() + data.expires_in * 1000;
        localStorage.setItem('access_token_expires_at', String(expiresAt));
      }
      return true;
    }
    return false;
  } catch {
    return false;
  }
}

/**
 * Typed API client. All paths, params, request bodies, and responses
 * are inferred from the OpenAPI spec at compile time.
 */
export const api = createClient<paths>({
  baseUrl: getBaseUrl(),
  credentials: 'include',
});

// Middleware: inject auth token and handle 401 refresh
api.use({
  async onRequest({ request }) {
    // Proactive refresh
    if (browser && isTokenExpiringSoon() && getAccessToken()) {
      await silentRefresh();
    }

    const token = getAccessToken();
    if (token) {
      request.headers.set('Authorization', `Bearer ${token}`);
    }
    return request;
  },

  async onResponse({ response, request }) {
    if (response.status === 401 && browser) {
      const refreshed = await silentRefresh();
      if (refreshed) {
        // Retry with new token
        const token = getAccessToken();
        if (token) {
          request.headers.set('Authorization', `Bearer ${token}`);
        }
        return fetch(request);
      }
      // Still 401 after refresh — redirect to login
      localStorage.removeItem('access_token');
      localStorage.removeItem('access_token_expires_at');
      await goto('/login');
    }
    return response;
  },
});

// Re-export schema types for convenient use
export type { paths, components } from './v1';
