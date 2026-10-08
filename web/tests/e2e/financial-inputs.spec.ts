import { test, expect } from '@playwright/test';
test('quarterly detection preserves frequency, requests native JPY amount and keeps beneficiary decimals', async ({
	page
}) => {
	const profile = { id: 'input-profile', name: 'Saisie de recette', privacy_default: 'N1' };
	let calendar: any = null;
	let beneficiary: any = null;
	await page.route('**/v1/**', async (route) => {
		const path = new URL(route.request().url()).pathname;
		let data: any = {};
		if (path === '/v1/profiles') data = { profiles: [profile] };
		else if (path === '/v1/auth/login') data = { token: 'input-session', profile };
		else if (path === '/v1/assets/')
			data = {
				assets: [{ id: 'yen-account', name: 'Compte yen', currency: 'JPY', kind: 'checking' }]
			};
		else if (path === '/v1/recurring')
			data = {
				recurring: [
					{
						merchant_key: 'marchand',
						label: 'Facture trimestre',
						periodicity: 'quarterly',
						amount_cents: -1250,
						next_date: '2026-11-30'
					}
				]
			};
		else if (path === '/v1/calendar' && route.request().method() === 'POST') {
			calendar = route.request().postDataJSON();
			data = calendar;
		} else if (path === '/v1/contacts/')
			data = { contacts: [{ id: 'contact', name: 'Contact test' }] };
		else if (path === '/v1/documents/')
			data = { documents: [{ id: 'contract', name: 'Contrat test' }] };
		else if (path === '/v1/beneficiaries' && route.request().method() === 'PUT') {
			beneficiary = route.request().postDataJSON();
			data = beneficiary;
		}
		await route.fulfill({ json: data });
	});
	await page.goto('/login');
	await page.getByRole('button', { name: profile.name }).click();
	await page.getByLabel('Code personnel').fill('123456');
	await page.getByRole('button', { name: 'Déverrouiller', exact: true }).click();
	await page.getByRole('link', { name: 'Flux', exact: true }).click();
	await page.getByRole('button', { name: 'Calendrier', exact: true }).click();
	await page.getByText('Récurrences détectées à confirmer', { exact: true }).click();
	await page.getByRole('button', { name: 'Confirmer et planifier' }).click();
	await expect(page.getByLabel('Répétition')).toHaveValue('quarterly');
	await expect(page.getByLabel('Montant signé')).toHaveValue('');
	await page.getByLabel('Compte', { exact: false }).selectOption('yen-account');
	await page.getByLabel('Montant signé (JPY)').fill('-1275');
	await page.getByRole('button', { name: 'Enregistrer', exact: true }).click();
	await expect.poll(() => calendar).not.toBeNull();
	expect(calendar.amount_cents).toBe(-1275);
	expect(calendar.frequency).toBe('quarterly');
	await page.getByRole('link', { name: 'Patrimoine', exact: true }).click();
	await page.getByRole('button', { name: 'Transmission', exact: true }).click();
	await page.getByRole('button', { name: 'Lier un bénéficiaire' }).click();
	await page.getByLabel('Bénéficiaire', { exact: false }).selectOption('contact');
	await page.getByLabel('Contrat du coffre').selectOption('contract');
	await page.getByLabel('Part (%)').fill('33,33');
	await page.getByRole('button', { name: 'Enregistrer', exact: true }).click();
	await expect.poll(() => beneficiary).not.toBeNull();
	expect(beneficiary.share_bps).toBe(3333);
});
