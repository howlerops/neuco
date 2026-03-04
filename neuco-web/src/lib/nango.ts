import Nango from '@nangohq/frontend';
import { apiClient } from '$lib/api/client';

const NANGO_HOST = import.meta.env.VITE_NANGO_URL ?? 'http://localhost:3003';

export async function createNangoSession(): Promise<Nango> {
	const resp = await apiClient.post<{ token: string }>('/api/v1/auth/nango/connect-session');
	return new Nango({ connectSessionToken: resp.token, host: NANGO_HOST });
}
