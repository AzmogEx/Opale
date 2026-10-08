import { test, expect } from '@playwright/test';
test('real API: calendar, decisions, investment flows, CCA, emergency revocation and alerts', async ({
	page,
	request
}) => {
	const pin = '72938461',
		name = `Advanced web ${crypto.randomUUID()}`,
		recipientName = `Recipient web ${crypto.randomUUID()}`;
	const makeProfile = async (name: string) => {
		const r = await request.post('/v1/profiles', { data: { name, pin } });
		expect(r.ok()).toBe(true);
		return r.json();
	};
	const profile = await makeProfile(name),
		recipient = await makeProfile(recipientName);
	const today = new Date().toISOString().slice(0, 10),
		yesterday = new Date(Date.now() - 86400000).toISOString().slice(0, 10);
	try {
		await page.goto('/login');
		await page.getByRole('button', { name: new RegExp(name + '$') }).click();
		await page.getByLabel('Code personnel').fill(pin);
		await page.getByRole('button', { name: 'Déverrouiller', exact: true }).click();
		await expect(page.getByRole('heading', { name: 'Vue d’ensemble' })).toBeVisible();
		const token = await page.evaluate(() => sessionStorage.getItem('opale.token') || '');
		const headers = { Authorization: `Bearer ${token}` };
		const send = async (path: string, data: any) => {
			const r = await request.post(path, { headers, data });
			expect(r.ok(), await r.text()).toBe(true);
			return r.json();
		};
		const get = async (path: string) => {
			const r = await request.get(path, { headers });
			expect(r.ok()).toBe(true);
			return r.json();
		};
		const cash = await send('/v1/assets/', {
			name: 'Compte modules',
			kind: 'checking',
			currency: 'EUR',
			initial_value_cents: 100000,
			initial_as_of: yesterday,
			client_request_id: crypto.randomUUID()
		});
		const investment = await send('/v1/assets/', {
			name: 'PEA modules',
			kind: 'pea',
			currency: 'EUR',
			initial_value_cents: 100000,
			initial_as_of: yesterday,
			client_request_id: crypto.randomUUID()
		});
		await send(`/v1/assets/${investment.id}/valuations`, { value_cents: 150000, as_of: today });
		const company = await send('/v1/assets/', {
			name: 'Société modules',
			kind: 'company_share',
			currency: 'EUR',
			initial_value_cents: 200000,
			initial_as_of: yesterday,
			client_request_id: crypto.randomUUID()
		});
		const document = await send('/v1/documents/', {
			name: 'Contrat modules',
			kind: 'contract',
			mime: 'text/plain',
			asset_id: cash.id,
			content_base64: Buffer.from('Contrat synthétique').toString('base64')
		});
		await page.getByRole('button', { name: 'Mes alertes', exact: true }).click();
		await page.getByRole('button', { name: 'Ajouter', exact: true }).click();
		await page.getByLabel('Déclencheur').selectOption('cash_below');
		await page.getByLabel('Seuil (EUR)').fill('500');
		await page.getByRole('button', { name: 'Enregistrer', exact: true }).click();
		await expect(page.getByText('Enregistré.', { exact: true })).toBeVisible();
		expect((await get('/v1/alerts/custom/')).alerts[0].threshold_cents).toBe(50000);
		await page.getByRole('link', { name: 'Flux', exact: true }).click();
		await page.getByRole('button', { name: 'Calendrier', exact: true }).click();
		await page.getByRole('button', { name: 'Ajouter une échéance' }).click();
		await page.getByLabel('Compte', { exact: false }).selectOption(cash.id);
		await page.getByLabel('Libellé').fill('Échéance modules');
		await page.getByLabel('Montant signé').fill('-12,50');
		await page.getByRole('button', { name: 'Enregistrer', exact: true }).click();
		await expect(page.getByRole('button', { name: 'Modifier cette occurrence' })).toBeVisible();
		await page.getByRole('button', { name: 'Modifier cette occurrence' }).click();
		await page.getByLabel('État', { exact: false }).selectOption('excluded');
		await page.getByRole('button', { name: 'Enregistrer', exact: true }).click();
		await expect(page.getByLabel('État', { exact: false })).toHaveCount(0);
		expect((await get(`/v1/calendar?from=${today}&until=${today}`)).occurrences[0].status).toBe(
			'excluded'
		);
		await page.getByRole('link', { name: 'Projection', exact: true }).click();
		await page.getByRole('button', { name: 'Décisions', exact: true }).click();
		await page.getByLabel('Décision', { exact: false }).selectOption('cash_credit');
		await page.getByLabel('Capital disponible').fill('20000');
		await page.getByLabel('Budget mensuel avant').fill('500');
		await page.getByLabel('Prix du bien').fill('10000');
		await page.getByLabel('Capital emprunté ou restant dû').fill('10000');
		await page.getByRole('button', { name: 'Calculer', exact: true }).click();
		await expect(page.getByRole('heading', { name: 'Résultat', exact: true })).toBeVisible();
		await expect(page.getByText('Patrimoine final nominal', { exact: true }).first()).toBeVisible();
		await page.screenshot({ path: 'qa-artifacts/decisions-desktop.png', fullPage: true });
		await page.getByRole('link', { name: 'Patrimoine', exact: true }).click();
		await page.getByRole('button', { name: 'Investissements', exact: true }).click();
		await page
			.getByRole('combobox', { name: 'Placement', exact: true })
			.selectOption(investment.id);
		await page.getByRole('button', { name: 'Ajouter un flux' }).click();
		await page.getByLabel('Nature').selectOption('contribution');
		await page.getByLabel('Montant positif').fill('500');
		await page.getByRole('button', { name: 'Enregistrer', exact: true }).click();
		await page.getByLabel('Je confirme avoir saisi tous les flux').check();
		await page.getByRole('button', { name: 'Enregistrer la couverture historique' }).click();
		await expect
			.poll(async () => (await get(`/v1/assets/${investment.id}/investment`)).performance.known)
			.toBe(true);
		expect((await get(`/v1/assets/${investment.id}/investment`)).performance.gain_cents).toBe(0);
		await page.getByRole('button', { name: 'Entrepreneur', exact: true }).click();
		await page.getByLabel('Actif concerné').selectOption(company.id);
		await page.getByLabel('Compte courant d’associé').fill('500');
		await page.getByRole('button', { name: 'Enregistrer', exact: true }).click();
		await expect
			.poll(async () => (await get('/v1/company')).companies[0].details.cca_cents)
			.toBe(50000);
		const companyData = (await get('/v1/company')).companies[0];
		expect(companyData.details.cca_asset_id).toBeTruthy();
		const cca = (await get('/v1/assets/')).assets.find(
			(a: any) => a.id === companyData.details.cca_asset_id
		);
		expect(cca.latest_value_cents).toBe(50000);
		await page.getByRole('button', { name: 'Transmission', exact: true }).click();
		const contacts = page
			.locator('section')
			.filter({ has: page.getByRole('heading', { name: 'Contacts de confiance', exact: true }) });
		await contacts.getByRole('button', { name: 'Ajouter', exact: true }).click();
		await contacts.getByLabel('Nom', { exact: false }).fill('Contact modules');
		await contacts.getByLabel('Rôle').selectOption('trusted');
		await contacts.getByRole('button', { name: 'Enregistrer', exact: true }).click();
		await expect(contacts.getByText('Contact modules', { exact: true })).toBeVisible();
		await page.getByRole('button', { name: 'Lier un bénéficiaire' }).click();
		const contact = (await get('/v1/contacts/')).contacts[0];
		await page.getByLabel('Bénéficiaire', { exact: false }).selectOption(contact.id);
		await page.getByLabel('Contrat du coffre').selectOption(document.id);
		await page.getByRole('button', { name: 'Enregistrer', exact: true }).click();
		await expect.poll(async () => (await get('/v1/beneficiaries')).beneficiaries.length).toBe(1);
		await page.getByLabel('Profil destinataire').selectOption(recipient.id);
		await page.getByLabel('Compte modules', { exact: true }).check();
		await page.getByLabel('Contrat modules', { exact: true }).check();
		await page.getByRole('button', { name: 'Créer un droit inactif' }).click();
		await expect(page.getByRole('button', { name: 'Activer cet accès' })).toBeVisible();
		const grant = (await get('/v1/emergency-grants')).grants[0];
		expect(grant.active).toBe(false);
		await page.getByRole('button', { name: 'Activer cet accès' }).click();
		await expect(page.getByRole('button', { name: 'Révoquer immédiatement' })).toBeVisible();
		const recipientLogin = await request.post('/v1/auth/login', {
			data: { profile_id: recipient.id, pin }
		});
		const rh = { Authorization: `Bearer ${(await recipientLogin.json()).token}` };
		const access = await request.get(`/v1/emergency/${grant.id}`, { headers: rh });
		expect(access.ok()).toBe(true);
		const allowed = await access.json();
		expect(allowed.assets).toHaveLength(1);
		expect(allowed.documents).toHaveLength(1);
		await page.getByRole('button', { name: 'Révoquer immédiatement' }).click();
		await expect(page.getByRole('button', { name: 'Activer cet accès' })).toBeVisible();
		expect((await request.get(`/v1/emergency/${grant.id}`, { headers: rh })).status()).toBe(404);
	} finally {
		for (const p of [profile, recipient]) {
			const l = await request.post('/v1/auth/login', { data: { profile_id: p.id, pin } });
			if (l.ok()) {
				const d = await request.delete(`/v1/me?confirm=${encodeURIComponent(p.name)}`, {
					headers: { Authorization: `Bearer ${(await l.json()).token}` }
				});
				expect(d.status()).toBe(204);
			}
		}
	}
});
