import { test as base, expect } from '@playwright/test';

/** The shipped client always targets Vaycode. Only the test browser proxies to a disposable API. */
export const test = base.extend<{ isolatedBackend: void }>({
	isolatedBackend: [
		async ({ context, baseURL }, use) => {
			const local = new URL(baseURL ?? '');
			if (!['127.0.0.1', 'localhost', '[::1]'].includes(local.hostname))
				throw new Error('Integration tests require a disposable loopback API, never production.');
			await context.route('https://opale.vaycode.com/**', async (route) => {
				const requested = new URL(route.request().url());
				if (!requested.pathname.startsWith('/v1/')) return route.abort();
				const cors = {
					'access-control-allow-origin': local.origin,
					'access-control-allow-methods': 'GET,POST,PUT,PATCH,DELETE,OPTIONS',
					'access-control-allow-headers': 'Authorization,Content-Type'
				};
				if (route.request().method() === 'OPTIONS')
					return route.fulfill({ status: 204, headers: cors });
				const response = await route.fetch({
					url: local.origin + requested.pathname + requested.search,
					maxRedirects: 0
				});
				await route.fulfill({ response, headers: { ...response.headers(), ...cors } });
			});
			await use();
		},
		{ auto: true }
	]
});
export { expect };
