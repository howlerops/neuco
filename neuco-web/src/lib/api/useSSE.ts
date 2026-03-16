import { browser } from '$app/environment';

interface SSEOptions {
	onMessage?: (event: MessageEvent<string>) => void;
	onError?: (event: Event) => void;
	onOpen?: (event: Event) => void;
}

export function connectSSE(url: string, options: SSEOptions = {}) {
	if (!browser) {
		return { close: () => {} };
	}

	const token = localStorage.getItem('access_token');
	const target = new URL(url, window.location.origin);

	if (token) {
		target.searchParams.set('access_token', token);
	}

	const source = new EventSource(target.toString());

	if (options.onMessage) {
		source.onmessage = options.onMessage;
	}

	if (options.onError) {
		source.onerror = options.onError;
	}

	if (options.onOpen) {
		source.onopen = options.onOpen;
	}

	return {
		close: () => source.close()
	};
}
