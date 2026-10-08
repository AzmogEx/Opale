import { defineConfig, devices, chromium } from '@playwright/test';
export default defineConfig({
	testDir: process.env.OPALE_INTEGRATION ? './tests/integration' : './tests/e2e',
	fullyParallel: false,
	timeout: 90000,
	workers: 1,
	reporter: 'list',
	use: {
		actionTimeout: 15000,
		baseURL: process.env.OPALE_WEB_URL ?? 'http://127.0.0.1:5173',
		trace: 'retain-on-failure',
		launchOptions: { executablePath: process.env.OPALE_CHROMIUM_PATH ?? chromium.executablePath() }
	},
	projects: [{ name: 'chromium', use: { ...devices['Desktop Chrome'] } }]
});
