import { test, expect } from './fixtures';
// Creates and deletes only its own profile on the disposable integration API.
test('real API: transfer, goals, vault, fiscal source and module rendering', async ({
	page,
	request
}) => {
	const pin = '72938461',
		name = `Modules ${crypto.randomUUID()}`;
	const created = await request.post('/v1/profiles', { data: { name, pin } });
	expect(created.ok()).toBe(true);
	const profile = await created.json();
	const errors: string[] = [];
	page.on('pageerror', (e) => errors.push(e.message));
	try {
		await page.goto('/login');
		await page.getByRole('button', { name: new RegExp(name + '$') }).click();
		await page.getByLabel('Code personnel').fill(pin);
		await page.getByRole('button', { name: 'Déverrouiller', exact: true }).click();
		await expect(page.getByRole('heading', { name: 'Vue d’ensemble' })).toBeVisible();
		const token = await page.evaluate(() => sessionStorage.getItem('opale.token') || '');
		const headers = { Authorization: `Bearer ${token}` };
		const accounts: any[] = [];
		for (const n of ['Courant test', 'Épargne test']) {
			const r = await request.post('/v1/assets/', {
				headers,
				data: {
					name: n,
					kind: 'checking',
					currency: 'EUR',
					initial_value_cents: 100000,
					initial_as_of: '2026-10-01',
					client_request_id: crypto.randomUUID()
				}
			});
			expect(r.ok()).toBe(true);
			accounts.push(await r.json());
		}
		await page.getByRole('link', { name: 'Flux', exact: true }).click();
		await page.getByRole('button', { name: 'Virement entre comptes' }).click();
		await page.getByLabel('Compte débité').selectOption(accounts[0].id);
		await page.getByLabel('Compte crédité').selectOption(accounts[1].id);
		await page.getByLabel(/^Montant débité/).fill('125,55');
		await page.getByLabel(/^Montant crédité/).fill('125,55');
		await page.getByLabel(/^Frais dans/).fill('1,20');
		await page.getByRole('button', { name: 'Enregistrer', exact: true }).click();
		await expect(page.getByRole('heading', { name: 'Virement entre tes comptes' })).toHaveCount(0);
		const tx = await (await request.get('/v1/transactions/', { headers })).json();
		expect(tx.transactions.filter((x: any) => x.transfer_id).length).toBe(3);
		expect(tx.transactions.reduce((s: number, x: any) => s + x.amount_cents, 0)).toBe(-120);
		const income = await request.post('/v1/transactions/', {
			headers,
			data: {
				asset_id: accounts[0].id,
				label: 'Revenu synthétique',
				amount_cents: 300000,
				occurred_on: new Date().toISOString().slice(0, 10),
				flow_kind: 'expense_income'
			}
		});
		expect(income.ok()).toBe(true);
		await page.getByRole('link', { name: 'Projection', exact: true }).click();
		await page.getByRole('button', { name: 'Objectifs', exact: true }).click();
		await page.getByRole('button', { name: 'Ajouter', exact: true }).click();
		await page.getByLabel('Nom de l’objectif').fill('Voyage test');
		await page.getByLabel('Montant cible').fill('5000');
		await page.getByLabel('Épargne mensuelle affectée').fill('200');
		await page.getByRole('button', { name: 'Enregistrer', exact: true }).click();
		await expect(page.getByText('Voyage test', { exact: true })).toBeVisible();
		await page.getByRole('button', { name: 'Modifier', exact: true }).click();
		await page.getByLabel('Montant cible').fill('6000');
		await page.getByRole('button', { name: 'Enregistrer', exact: true }).click();
		// Le formulaire se ferme après la réponse PATCH ; attendre sa persistance
		// avant la lecture indépendante de l'API (plus lente derrière nginx en CI).
		await expect(page.getByLabel('Montant cible')).toHaveCount(0);
		const goals = await (await request.get('/v1/goals/', { headers })).json();
		expect(goals.goals[0].target_cents).toBe(600000);
		await page.getByRole('button', { name: 'Fiscalité et PER', exact: true }).click();
		await page.getByLabel('Revenu annuel imposable').fill('50000');
		await page.getByRole('button', { name: 'Calculer', exact: true }).click();
		await expect(page.getByRole('link', { name: 'Consulter la source officielle' })).toBeVisible();
		await page.getByRole('link', { name: 'Patrimoine', exact: true }).click();
		await page.getByRole('button', { name: 'Coffre-fort', exact: true }).click();
		const content = 'Document synthétique recette Web';
		await page
			.getByLabel('Fichier (10 Mo maximum)')
			.setInputFiles({ name: 'recette.txt', mimeType: 'text/plain', buffer: Buffer.from(content) });
		await page.getByRole('button', { name: 'Ajouter au coffre', exact: true }).click();
		await expect(page.getByText('recette.txt', { exact: true })).toBeVisible();
		const docs = await (await request.get('/v1/documents/', { headers })).json();
		const bytes = await request.get(`/v1/documents/${docs.documents[0].id}/content`, { headers });
		expect(await bytes.text()).toBe(content);
		for (const tab of [
			'Actifs',
			'Dettes',
			'Immobilier',
			'Investissements',
			'Objets',
			'Entrepreneur',
			'Transmission',
			'Devises',
			'Banques',
			'Cours'
		]) {
			await page.getByRole('button', { name: tab, exact: true }).click();
			await expect(page.locator('.stack')).toBeVisible();
			await expect(page.locator('.notice.error')).toHaveCount(0);
		}
		expect(errors).toEqual([]);
	} finally {
		const login = await request.post('/v1/auth/login', { data: { profile_id: profile.id, pin } });
		if (login.ok())
			await request.delete(`/v1/me?confirm=${encodeURIComponent(name)}`, {
				headers: { Authorization: `Bearer ${(await login.json()).token}` }
			});
	}
});
