import { test, expect } from '@playwright/test';
test('label suggestion requires preview acceptance, patches only label and explains unavailability', async ({
	page
}) => {
	const profile = { id: 'label-profile', name: 'Libellés de recette', privacy_default: 'N1' };
	const tx = {
		id: 'label-tx',
		asset_id: 'cash-account',
		label: 'CB 0810 CAFE CENTRAL',
		amount_cents: -1250,
		occurred_on: '2026-10-08',
		currency: 'EUR'
	};
	let patch: unknown = null;
	let available = true;
	await page.route('**/v1/**', async (route) => {
		const path = new URL(route.request().url()).pathname;
		let data: unknown = {};
		if (path === '/v1/profiles') data = { profiles: [profile] };
		else if (path === '/v1/auth/login') data = { token: 'label-session', profile };
		else if (path === '/v1/assets/')
			data = {
				assets: [{ id: tx.asset_id, name: 'Compte recette', currency: 'EUR', kind: 'checking' }]
			};
		else if (path === '/v1/transactions/') data = { transactions: [tx] };
		else if (path.endsWith('/label-suggestion'))
			data = available
				? {
						available: true,
						source: 'homelab',
						state: 'available',
						original_label: tx.label,
						suggested_label: 'Café Central'
					}
				: {
						available: false,
						original_label: tx.label,
						source: 'homelab',
						state: 'unavailable',
						reason: 'Homelab non configuré. Le libellé reste inchangé.'
					};
		else if (route.request().method() === 'PATCH') {
			patch = route.request().postDataJSON();
			tx.label = (patch as any).label;
			data = tx;
		}
		await route.fulfill({ json: data });
	});
	await page.goto('/login');
	await page.getByRole('button', { name: profile.name }).click();
	await page.getByLabel('Code personnel').fill('123456');
	await page.getByRole('button', { name: 'Déverrouiller', exact: true }).click();
	await page.getByRole('link', { name: 'Flux', exact: true }).click();
	await page
		.locator('summary')
		.filter({ hasText: /^Actions$/ })
		.click();
	await page.getByRole('button', { name: 'Nettoyer le libellé', exact: true }).click();
	await expect(page.getByText('Seul ce libellé sera envoyé', { exact: false })).toBeVisible();
	await page.getByRole('button', { name: 'Suggérer via le homelab' }).click();
	await expect(page.getByRole('heading', { name: 'Suggestion du homelab privé' })).toBeVisible();
	expect(patch).toBeNull();
	expect(tx.label).toBe('CB 0810 CAFE CENTRAL');
	await page.getByRole('button', { name: 'Conserver le libellé actuel' }).click();
	expect(patch).toBeNull();
	await page.getByRole('button', { name: 'Nettoyer le libellé', exact: true }).click();
	await page.getByRole('button', { name: 'Suggérer via le homelab' }).click();
	await page.getByRole('button', { name: 'Accepter ce libellé' }).click();
	await expect(page.getByRole('cell', { name: 'Café Central Sans catégorie' })).toBeVisible();
	expect(patch).toEqual({ label: 'Café Central' });
	available = false;
	await page.getByRole('button', { name: 'Nettoyer le libellé', exact: true }).click();
	await page.getByRole('button', { name: 'Suggérer via le homelab' }).click();
	await expect(page.getByRole('status')).toContainText('Homelab non configuré');
	await expect(page.getByRole('button', { name: 'Accepter ce libellé' })).toHaveCount(0);
	expect(tx.label).toBe('Café Central');
});
