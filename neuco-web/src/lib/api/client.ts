import { browser } from '$app/environment';
import { goto } from '$app/navigation';

// ─── Error Types ──────────────────────────────────────────────────────────────

export class ApiError extends Error {
	constructor(
		public readonly status: number,
		public readonly statusText: string,
		public readonly body: unknown
	) {
		super(`API Error ${status}: ${statusText}`);
		this.name = 'ApiError';
	}
}

// ─── Config ───────────────────────────────────────────────────────────────────

function getBaseUrl(): string {
	if (typeof import.meta !== 'undefined' && import.meta.env?.VITE_API_BASE_URL) {
		return import.meta.env.VITE_API_BASE_URL as string;
	}
	return 'http://localhost:8080';
}

// ─── Token Management ─────────────────────────────────────────────────────────

function getAccessToken(): string | null {
	if (!browser) return null;
	return localStorage.getItem('access_token');
}

// ─── snake_case → camelCase transformer ───────────────────────────────────────
// The Go backend sends snake_case JSON keys. The frontend uses camelCase types.
// This transformer converts all object keys recursively on every API response.

function snakeToCamel(str: string): string {
	return str.replace(/_([a-z0-9])/g, (_, letter) => letter.toUpperCase());
}

export function transformKeys(obj: unknown): unknown {
	if (obj === null || obj === undefined) return obj;
	if (Array.isArray(obj)) return obj.map(transformKeys);
	if (typeof obj === 'object') {
		const result: Record<string, unknown> = {};
		for (const [key, value] of Object.entries(obj as Record<string, unknown>)) {
			result[snakeToCamel(key)] = transformKeys(value);
		}
		return result;
	}
	return obj;
}

// Also transform camelCase request bodies → snake_case for the backend
function camelToSnake(str: string): string {
	return str.replace(/[A-Z]/g, (letter) => `_${letter.toLowerCase()}`);
}

function transformKeysToSnake(obj: unknown): unknown {
	if (obj === null || obj === undefined) return obj;
	if (Array.isArray(obj)) return obj.map(transformKeysToSnake);
	if (typeof obj === 'object') {
		const result: Record<string, unknown> = {};
		for (const [key, value] of Object.entries(obj as Record<string, unknown>)) {
			result[camelToSnake(key)] = transformKeysToSnake(value);
		}
		return result;
	}
	return obj;
}

// ─── Core Fetch ───────────────────────────────────────────────────────────────

interface RequestOptions {
	headers?: Record<string, string>;
	signal?: AbortSignal;
}

async function request<T>(
	method: string,
	path: string,
	body?: unknown,
	options: RequestOptions = {}
): Promise<T> {
	const baseUrl = getBaseUrl();
	const url = `${baseUrl}${path}`;

	const headers: Record<string, string> = {
		'Content-Type': 'application/json',
		...options.headers
	};

	const token = getAccessToken();
	if (token) {
		headers['Authorization'] = `Bearer ${token}`;
	}

	const init: RequestInit = {
		method,
		headers,
		signal: options.signal
	};

	if (body !== undefined) {
		// Transform outgoing camelCase keys to snake_case for Go backend
		init.body = JSON.stringify(transformKeysToSnake(body));
	}

	let response: Response;
	try {
		response = await fetch(url, init);
	} catch (err) {
		throw new ApiError(0, 'Network error', err);
	}

	if (response.status === 401) {
		if (browser) {
			localStorage.removeItem('access_token');
			localStorage.removeItem('refresh_token');
			await goto('/login');
		}
		throw new ApiError(401, 'Unauthorized', null);
	}

	if (!response.ok) {
		let errorBody: unknown;
		try {
			errorBody = transformKeys(await response.json());
		} catch {
			errorBody = await response.text().catch(() => null);
		}
		throw new ApiError(response.status, response.statusText, errorBody);
	}

	// 204 No Content or empty body
	if (response.status === 204 || response.headers.get('content-length') === '0') {
		return undefined as T;
	}

	const contentType = response.headers.get('content-type') ?? '';
	if (!contentType.includes('application/json')) {
		return (await response.text()) as unknown as T;
	}

	// Transform incoming snake_case keys to camelCase
	const raw = await response.json();
	return transformKeys(raw) as T;
}

// ─── HTTP Method Shortcuts ────────────────────────────────────────────────────

export const apiClient = {
	get<T>(path: string, options?: RequestOptions): Promise<T> {
		return request<T>('GET', path, undefined, options);
	},

	post<T>(path: string, body?: unknown, options?: RequestOptions): Promise<T> {
		return request<T>('POST', path, body, options);
	},

	patch<T>(path: string, body?: unknown, options?: RequestOptions): Promise<T> {
		return request<T>('PATCH', path, body, options);
	},

	put<T>(path: string, body?: unknown, options?: RequestOptions): Promise<T> {
		return request<T>('PUT', path, body, options);
	},

	delete<T>(path: string, options?: RequestOptions): Promise<T> {
		return request<T>('DELETE', path, undefined, options);
	}
};

export default apiClient;
