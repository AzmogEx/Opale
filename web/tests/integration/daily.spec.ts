import { test, expect } from '@playwright/test';
// Requires the disposable integration API behind Vite's /v1 proxy. Never use a personal instance.
test('real API: create profile, initial valuation, transaction import/edit/filter/page, export and expiry', async ({
	page,
	request
}) => {
	const name = `Recette web ${crypto.randomUUID()}`;
	const pin = '72938461';
	let profileID = '';
	let token = '';
	try {
		await page.goto('/login');
		await page.getByRole('button', { name: 'Créer un profil', exact: true }).click();
		await page.getByLabel('Prénom ou nom du profil').fill(name);
		await page.getByLabel('Code personnel', { exact: true }).fill(pin);
		await page.getByLabel('Confirmer le code').fill(pin);
		await page.getByRole('button', { name: 'Créer mon profil' }).click();
		await expect(page.getByRole('heading', { name: 'Vue d’ensemble' })).toBeVisible();
		const saved = await page.evaluate(() => ({
			profile: JSON.parse(sessionStorage.getItem('opale.profile') || 'null'),
			token: sessionStorage.getItem('opale.token')
		}));
		profileID = saved.profile.id;
		token = saved.token ?? '';
		await page.getByRole('link', { name: 'Patrimoine', exact: true }).click();
		await page.getByRole('button', { name: 'Ajouter un actif' }).click();
		await page.getByLabel('Nom', { exact: false }).first().fill('Compte recette');
		await page.getByLabel('Type', { exact: false }).first().selectOption('checking');
		await page.getByLabel(/^Valeur initiale/).fill('1000,25');
		await page.getByRole('button', { name: 'Enregistrer', exact: true }).click();
		await expect(page.getByRole('button', { name: 'Compte recette', exact: true })).toBeVisible();
		const a = await request.get('/v1/assets/', { headers: { Authorization: `Bearer ${token}` } });
		expect(a.ok()).toBe(true);
		const asset = (await a.json()).assets.find((x: any) => x.name === 'Compte recette');
		expect(asset.latest_value_cents).toBe(100025);
		await page.getByRole('link', { name: 'Flux', exact: true }).click();
		await page.getByRole('button', { name: 'Importer CSV / OFX' }).click();
		await page.getByLabel('Compte de destination').selectOption(asset.id);
		const today = new Date();
		const day = `${today.getFullYear()}-${String(today.getMonth() + 1).padStart(2, '0')}-08`;
		const csv = `date;libelle;montant\n${Array.from({ length: 55 }, (_, i) => `${day};Recette ${i};-1,25`).join('\n')}`;
		await page
			.getByLabel('Fichier CSV ou OFX')
			.setInputFiles({ name: 'recette.csv', mimeType: 'text/csv', buffer: Buffer.from(csv) });
		await expect(page.getByText('Aperçu du fichier avant import')).toBeVisible();
		await page.getByRole('button', { name: 'Importer ce fichier', exact: true }).click();
		await expect(page.getByRole('button', { name: 'Page suivante', exact: true })).toBeEnabled();
		await page.getByRole('button', { name: 'Page suivante', exact: true }).click();
		await expect(page.getByText('Page 2', { exact: true })).toBeVisible();
		await page.getByLabel('Rechercher', { exact: true }).fill('Recette 54');
		await page.getByRole('button', { name: 'Appliquer les filtres' }).click();
		await expect(page.locator('tbody tr')).toHaveCount(1);
		await page.locator('tbody summary').click();
		await page.locator('tbody').getByRole('button', { name: 'Modifier', exact: true }).click();
		await page.getByLabel('Libellé', { exact: false }).fill('Opération corrigée');
		await page.getByRole('button', { name: 'Enregistrer', exact: true }).click();
		await page.getByLabel('Rechercher', { exact: true }).fill('corrigée');
		await page.getByRole('button', { name: 'Appliquer les filtres' }).click();
		await expect(page.locator('tbody')).toContainText('Opération corrigée');
		await page.getByRole('button', { name: 'Paramètres', exact: true }).click();
		const downloadPromise = page.waitForEvent('download');
		await page.getByRole('button', { name: 'Télécharger mon export' }).click();
		const download = await downloadPromise;
		expect(download.suggestedFilename()).toContain('.zip');
		expect(await download.failure()).toBeNull();
		await page.reload();
		await expect(page.getByRole('heading', { name: `Déverrouiller ${name}` })).toBeVisible();
		await page.getByLabel('Code personnel').fill(pin);
		await page.getByRole('button', { name: 'Déverrouiller', exact: true }).click();
		await expect(page.getByRole('heading', { name: 'Vue d’ensemble' })).toBeVisible();
		token = await page.evaluate(() => sessionStorage.getItem('opale.token') || '');
		await request.post('/v1/auth/logout', { headers: { Authorization: `Bearer ${token}` } });
		await page.getByRole('button', { name: 'Actualiser', exact: true }).click();
		await expect(page).toHaveURL(/login/);
	} finally {
		if (profileID) {
			const login = await request.post('/v1/auth/login', { data: { profile_id: profileID, pin } });
			if (login.ok()) {
				const result = await login.json();
				await request.delete(`/v1/me?confirm=${encodeURIComponent(name)}`, {
					headers: { Authorization: `Bearer ${result.token}` }
				});
			}
		}
	}
});
