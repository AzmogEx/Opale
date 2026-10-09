import { test, expect } from '@playwright/test';

test('legacy server overrides cannot redirect the client or reuse a foreign session', async ({
	page
}) => {
	await page.addInitScript(() => {
		localStorage.setItem('opale.baseURL', 'https://other.invalid');
		sessionStorage.setItem('opale.token', 'foreign-token');
		sessionStorage.setItem('opale.profile', JSON.stringify({ id: 'foreign', name: 'Foreign' }));
	});
	const requests: string[] = [];
	await page.route('**/v1/**', async (route) => {
		requests.push(route.request().url());
		expect(route.request().headers().authorization).toBeUndefined();
		await route.fulfill({ json: { profiles: [] } });
	});
	await page.goto('/login');
	await expect(page.getByText('Bienvenue ! Crée ton premier profil pour commencer.')).toBeVisible();
	expect(requests).toEqual(['https://opale.vaycode.com/v1/profiles']);
	await expect(page.locator('input[type="url"]')).toHaveCount(0);
	await expect(page.getByRole('button', { name: 'Utiliser ce serveur' })).toHaveCount(0);
	expect(await page.evaluate(() => localStorage.getItem('opale.baseURL'))).toBeNull();
	expect(await page.evaluate(() => sessionStorage.getItem('opale.token'))).toBeNull();
});
