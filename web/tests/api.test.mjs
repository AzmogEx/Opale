import test, { afterEach } from 'node:test';
import assert from 'node:assert/strict';
import { API, APIError } from '../.test-build/api.js';
const originalFetch = globalThis.fetch;
afterEach(() => (globalThis.fetch = originalFetch));
test('expired session invokes recovery only for authenticated 401, not a 403 or network outage', async () => {
	let expired = 0;
	const api = new API(
		'',
		() => 'token',
		() => expired++
	);
	globalThis.fetch = async () =>
		new Response(JSON.stringify({ error: { message: 'Expirée' } }), { status: 401 });
	await assert.rejects(api.netWorth(), (e) => e instanceof APIError && e.status === 401);
	assert.equal(expired, 1);
	globalThis.fetch = async () => new Response('{}', { status: 403 });
	await assert.rejects(api.netWorth());
	assert.equal(expired, 1);
	globalThis.fetch = async () => {
		throw new TypeError('Network unavailable');
	};
	await assert.rejects(api.netWorth());
	assert.equal(expired, 1);
});
test('a wrong unlock PIN does not invalidate an existing locked session', async () => {
	let expired = 0;
	const api = new API(
		'',
		() => 'old',
		() => expired++
	);
	globalThis.fetch = async () => new Response('{}', { status: 401 });
	await assert.rejects(api.login('profile', 'bad'));
	assert.equal(expired, 0);
});
test('a delayed response from a previous profile is discarded', async () => {
	let token = 'profile-a';
	let finish;
	globalThis.fetch = () => new Promise((resolve) => (finish = resolve));
	const api = new API('', () => token);
	const pending = api.netWorth();
	token = 'profile-b';
	finish(new Response(JSON.stringify({ net_cents: 10000 }), { status: 200 }));
	await assert.rejects(pending, (e) => e.status === 409);
});
test('pagination and filters are encoded and the bearer request disables browser caching', async () => {
	let requested;
	globalThis.fetch = async (url, options) => {
		requested = { url, options };
		return new Response('{"transactions":[]}', { status: 200 });
	};
	const api = new API('', () => 'secret');
	await api.listTransactions('2026-10-01', '2026-10-31', { offset: 50, limit: 51, q: 'a&b' });
	const url = new URL(requested.url, 'http://test');
	assert.equal(url.searchParams.get('offset'), '50');
	assert.equal(url.searchParams.get('q'), 'a&b');
	assert.equal(requested.options.cache, 'no-store');
	assert.equal(requested.options.headers.Authorization, 'Bearer secret');
});
test('assistant sends explicit opt-in and bounded caller history without enabling cloud implicitly', async () => {
	let payload;
	globalThis.fetch = async (_url, options) => {
		payload = JSON.parse(options.body);
		return new Response('{"answer":"Réponse","tier":"data"}', { status: 200 });
	};
	const api = new API('', () => 't');
	await api.ask('Épargne ?', false, [{ role: 'user', text: 'Question précédente' }]);
	assert.equal(payload.allow_cloud, false);
	assert.equal(payload.history.length, 1);
});
test('unsafe monetary JSON is rejected instead of displaying silently rounded wealth', async () => {
	globalThis.fetch = async () => new Response('{"nested":{"value_cents":9007199254740993}}');
	await assert.rejects(new API('', () => null).netWorth(), (e) => e.status === 422);
});
test('a document that finishes downloading after profile change cannot open', async () => {
	let token = 'a',
		finish;
	globalThis.fetch = async () => ({
		ok: true,
		status: 200,
		blob: () => new Promise((resolve) => (finish = resolve))
	});
	const pending = new API('', () => token).download('/v1/documents/id/content', 'test.txt');
	await Promise.resolve();
	token = 'b';
	finish(new Blob(['private']));
	await assert.rejects(pending, (e) => e.status === 409);
});
