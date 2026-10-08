import { test, expect } from '@playwright/test';
const profile = { id: 'test-profile-a', name: 'Profil de recette', privacy_default: 'N1' };
async function mockAPI(page: any) {
	await page.route('**/v1/**', async (route: any) => {
		const path = new URL(route.request().url()).pathname;
		const payload =
			path === '/v1/profiles'
				? { profiles: [profile] }
				: path === '/v1/auth/login'
					? { token: 'test-session', profile, expires_at: '2099-01-01T00:00:00Z' }
					: path === '/v1/net-worth'
						? {
								net_cents: 12345600,
								assets_total_cents: 15000000,
								liabilities_total_cents: 2654400,
								currency: 'EUR'
							}
						: path.includes('history')
							? {
									points: [
										{ as_of: '2026-09-01', net_cents: 10000000 },
										{ as_of: '2026-10-01', net_cents: 12345600 }
									]
								}
							: path === '/v1/health-score'
								? { score: 75, components: [] }
								: path === '/v1/cashflow'
									? { start_cash_cents: 1234500, end_cash_cents: 1234500 }
									: path === '/v1/risks'
										? { risks: [] }
										: path === '/v1/alerts'
											? {
													alerts: [
														{
															kind: 'test',
															title: 'Alerte de recette',
															detail: 'Dépense privée : 777,77 €',
															severity: 'warning'
														}
													]
												}
											: {};
		await route.fulfill({
			status: 200,
			contentType: 'application/json',
			body: JSON.stringify(payload)
		});
	});
}
async function signIn(page: any) {
	await page.goto('/login');
	await page.getByRole('button', { name: 'Profil de recette' }).click();
	await page.getByLabel('Code personnel').fill('123456');
	await page.getByRole('button', { name: 'Déverrouiller', exact: true }).click();
	await expect(page.getByRole('heading', { name: 'Vue d’ensemble' })).toBeVisible();
}
test('privacy removes amounts and prose from the accessibility tree, then cold reload locks', async ({
	page
}) => {
	await mockAPI(page);
	await signIn(page);
	await expect(page.getByText('Dépense privée : 777,77 €')).toBeVisible();
	await page.getByRole('button', { name: 'Masquer les données' }).click();
	await expect(page.getByRole('heading', { name: 'Données masquées' })).toBeVisible();
	await expect(page.getByText('Dépense privée : 777,77 €')).toHaveCount(0);
	await page.getByRole('button', { name: 'Afficher mes données', exact: true }).click();
	await expect(page.getByRole('heading', { name: 'Vue d’ensemble' })).toBeVisible();
	await page.reload();
	await expect(
		page.getByRole('heading', { name: 'Déverrouiller Profil de recette' })
	).toBeVisible();
	await expect(page.getByText('Dépense privée : 777,77 €')).toHaveCount(0);
	expect(await page.evaluate(() => localStorage.getItem('opale.token'))).toBeNull();
});
test('mobile navigation and logout remain named and reachable without horizontal overflow', async ({
	page
}) => {
	await page.setViewportSize({ width: 390, height: 844 });
	await mockAPI(page);
	await signIn(page);
	for (const name of ['Accueil', 'Flux', 'Patrimoine', 'Projection', 'Assistant'])
		await expect(page.getByRole('link', { name, exact: true })).toBeVisible();
	await expect(page.getByRole('button', { name: 'Déconnexion', exact: true })).toBeVisible();
	expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(
		true
	);
	await expect(page.locator('.hero-amount')).toContainText('123');
	await page.screenshot({
		path: 'qa-artifacts/mobile-dashboard.png',
		fullPage: true,
		animations: 'disabled'
	});
	await page.getByRole('button', { name: 'Déconnexion', exact: true }).click();
	await expect(page).toHaveURL(/login/);
});
test('confirmed expiration redirects to sign-in and erases visible data', async ({ page }) => {
	await mockAPI(page);
	await signIn(page);
	await page.route('**/v1/net-worth', (route) =>
		route.fulfill({
			status: 401,
			contentType: 'application/json',
			body: '{"error":{"message":"session expirée"}}'
		})
	);
	await page.getByRole('button', { name: 'Actualiser', exact: true }).click();
	await expect(page).toHaveURL(/login/);
	await expect(page.getByText('Ta session a expiré. Reconnecte-toi.')).toBeVisible();
	expect(await page.evaluate(() => sessionStorage.getItem('opale.token'))).toBeNull();
});

test('five minutes of inactivity locks even directly after sign-in', async ({ page }) => {
	await page.clock.install();
	await mockAPI(page);
	await signIn(page);
	await page.clock.fastForward(310000);
	await expect(
		page.getByRole('heading', { name: 'Déverrouiller Profil de recette' })
	).toBeVisible();
});
test('logout hides data immediately while server revocation is unavailable', async ({ page }) => {
	await mockAPI(page);
	await signIn(page);
	await page.route('**/v1/auth/logout', async (route) => {
		await route.fulfill({ status: 503, body: '{}' });
	});
	await page.getByRole('button', { name: 'Déconnexion', exact: true }).click();
	await expect(page).toHaveURL(/login/);
	await expect(page.getByText('Dépense privée : 777,77 €')).toHaveCount(0);
	expect(await page.evaluate(() => sessionStorage.getItem('opale.token'))).toBeNull();
});
test('milestones expose the observed emergency fund and respect reduced motion', async ({
	page
}) => {
	await page.emulateMedia({ reducedMotion: 'reduce' });
	await mockAPI(page);
	await page.route('**/v1/health-score', (route) =>
		route.fulfill({ json: { score: 80, components: [], emergency_fund_ready: true } })
	);
	await page.route('**/v1/timeline', (route) => route.fulfill({ json: { events: [] } }));
	await signIn(page);
	await page.getByRole('button', { name: 'Jalons', exact: true }).click();
	await expect(
		page.getByRole('heading', {
			name: 'Ton fonds d’urgence couvre six mois de dépenses observées.'
		})
	).toBeVisible();
	expect(
		await page.locator('.milestone-star').evaluate((el) => getComputedStyle(el).animationName)
	).toBe('none');
	await page.getByRole('button', { name: 'Masquer les données' }).click();
	await expect(
		page.getByText('Ton fonds d’urgence couvre six mois de dépenses observées.')
	).toHaveCount(0);
});
