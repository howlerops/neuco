import * as Sentry from '@sentry/sveltekit';
import { initPostHog } from '$lib/analytics';

const dsn = import.meta.env.VITE_SENTRY_DSN;

if (dsn) {
	Sentry.init({
		dsn,
		environment: import.meta.env.VITE_APP_ENV ?? 'development',
		tracesSampleRate: 0.2
	});
}

initPostHog();

export const handleError = Sentry.handleErrorWithSentry();
