import { test, expect } from '@playwright/test';
test('real API: three decision comparisons at 0/5/10 years and unavailable private label suggestion', async ({
	page,
	request
}) => {
	const name = `Final web ${crypto.randomUUID()}`,
		pin = '72938461';
	const create = await request.post('/v1/profiles', { data: { name, pin } });
	expect(create.ok()).toBe(true);
	const profile = await create.json();
	try {
		await page.goto('/login');
		await page.getByRole('button', { name: new RegExp(name + '$') }).click();
		await page.getByLabel('Code personnel').fill(pin);
		await page.getByRole('button', { name: 'Déverrouiller', exact: true }).click();
		await expect(page.getByRole('heading', { name: 'Vue d’ensemble' })).toBeVisible();
		const token = await page.evaluate(() => sessionStorage.getItem('opale.token') || '');
		const headers = { Authorization: `Bearer ${token}` };
		await page.getByRole('link', { name: 'Projection', exact: true }).click();
		await page.getByRole('button', { name: 'Décisions', exact: true }).click();
		await page.getByLabel('Capital disponible').fill('200000');
		await page.getByLabel('Budget mensuel avant').fill('3000');
		await page.getByLabel('Prix du bien').fill('100000');
		await page.getByLabel('Apport (EUR)').fill('10000');
		await page.getByLabel('Loyer initial').fill('500');
		await page.getByLabel('Capital emprunté ou restant dû').fill('90000');
		await page.getByLabel('Taux annuel du crédit').fill('3');
		await page.getByLabel('Remboursement anticipé').fill('20000');
		for (const kind of ['buy_rent', 'cash_credit', 'repay_invest']) {
			await page.getByLabel('Décision', { exact: false }).selectOption(kind);
			const response = page.waitForResponse(
				(r) => r.url().endsWith('/v1/decisions/compare') && r.request().method() === 'POST'
			);
			await page.getByRole('button', { name: 'Calculer', exact: true }).click();
			const r = await response;
			expect(r.ok()).toBe(true);
			const data = await r.json();
			expect(data.timeline.map((p: any) => p.months)).toEqual([0, 60, 120]);
			expect(data.scenarios.map((p: any) => p.name)).toEqual(['prudent', 'normal', 'ambitieux']);
			await expect(page.getByRole('heading', { name: 'Impact à 0, 5 et 10 ans' })).toBeVisible();
			await expect(
				page
					.getByRole('table', { name: 'Patrimoine et liquidités en euros aux trois dates' })
					.getByRole('row')
			).toHaveCount(4);
			await expect(
				page
					.getByRole('table', { name: 'Comparaison prudent, normal et ambitieux' })
					.getByRole('row')
			).toHaveCount(4);
			await expect(page.getByText(data.recommendation.message, { exact: true })).toBeVisible();
			await expect(page.getByRole('heading', { name: 'Risques et limites' })).toBeVisible();
		}
		await page.setViewportSize({ width: 390, height: 844 });
		expect(
			await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)
		).toBe(true);
		await page.screenshot({ path: 'qa-artifacts/decisions-mobile.png', fullPage: true });
		const a = await request.post('/v1/assets/', {
			headers,
			data: {
				name: 'Compte libellé',
				kind: 'checking',
				currency: 'EUR',
				initial_value_cents: 100000,
				initial_as_of: new Date().toISOString().slice(0, 10),
				client_request_id: crypto.randomUUID()
			}
		});
		expect(a.ok()).toBe(true);
		const tx = await request.post('/v1/transactions/', {
			headers,
			data: {
				asset_id: (await a.json()).id,
				label: 'CB CAFE TEST',
				amount_cents: -1250,
				occurred_on: new Date().toISOString().slice(0, 10),
				flow_kind: 'expense_income'
			}
		});
		expect(tx.ok()).toBe(true);
		await page.getByRole('link', { name: 'Flux', exact: true }).click();
		await page
			.locator('summary')
			.filter({ hasText: /^Actions$/ })
			.click();
		await page.getByRole('button', { name: 'Nettoyer le libellé', exact: true }).click();
		const suggestion = page.waitForResponse((r) => r.url().endsWith('/label-suggestion'));
		await page.getByRole('button', { name: 'Suggérer via le homelab' }).click();
		const result = await (await suggestion).json();
		expect(result.available).toBe(false);
		await expect(page.getByRole('status')).toContainText(result.reason);
		await expect(page.getByRole('button', { name: 'Accepter ce libellé' })).toHaveCount(0);
		const rows = await request.get('/v1/transactions/', { headers });
		expect((await rows.json()).transactions[0].label).toBe('CB CAFE TEST');
	} finally {
		const login = await request.post('/v1/auth/login', { data: { profile_id: profile.id, pin } });
		if (login.ok())
			expect(
				(
					await request.delete(`/v1/me?confirm=${encodeURIComponent(name)}`, {
						headers: { Authorization: `Bearer ${(await login.json()).token}` }
					})
				).status()
			).toBe(204);
	}
});
