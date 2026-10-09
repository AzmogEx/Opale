import { test, expect } from '@playwright/test';

test('delayed company details cannot overwrite a CCA entered by the user', async ({ page }) => {
	const profile = { id: 'company-test', name: 'Société recette', privacy_default: 'N1' };
	const asset = {
		id: 'company',
		name: 'Participation test',
		kind: 'company_share',
		currency: 'EUR'
	};
	let release!: () => void;
	const detailsReady = new Promise<void>((resolve) => {
		release = resolve;
	});
	let saved: any = null;
	await page.route('**/v1/**', async (route) => {
		const path = new URL(route.request().url()).pathname;
		let data: any = {};
		if (path === '/v1/profiles') data = { profiles: [profile] };
		else if (path === '/v1/auth/login') data = { token: 'company-session', profile };
		else if (path === '/v1/assets/') data = { assets: [asset] };
		else if (path === '/v1/liabilities/') data = { liabilities: [] };
		else if (path === '/v1/company') {
			await detailsReady;
			data = { companies: [{ asset, details: { cca_cents: saved?.cca_cents ?? 0 } }] };
		} else if (path === '/v1/assets/company/company') {
			saved = route.request().postDataJSON();
			data = saved;
		}
		await route.fulfill({ json: data });
	});
	await page.goto('/login');
	await page.getByRole('button', { name: profile.name }).click();
	await page.getByLabel('Code personnel').fill('123456');
	await page.getByRole('button', { name: 'Déverrouiller', exact: true }).click();
	await page.getByRole('link', { name: 'Patrimoine', exact: true }).click();
	await page.getByRole('button', { name: 'Entrepreneur', exact: true }).click();
	await page.getByLabel('Actif concerné').selectOption(asset.id);
	await expect(page.getByText('Chargement des informations…', { exact: true })).toBeVisible();
	await expect(page.getByLabel('Compte courant d’associé')).toHaveCount(0);
	release();
	await page.getByLabel('Compte courant d’associé').fill('500,25');
	await page.getByRole('button', { name: 'Enregistrer', exact: true }).click();
	await expect.poll(() => saved?.cca_cents).toBe(50025);
	await expect(page.getByLabel('Compte courant d’associé')).toHaveValue('500.25');
});
