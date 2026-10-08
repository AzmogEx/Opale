import { API, type Profile } from './api';

/** Tokens stay within this tab. Reloads always require the PIN before rendering data. */
class Session {
	token = $state<string | null>(null);
	profile = $state<Profile | null>(null);
	baseURL = $state('');
	expiresAt = $state('');
	locked = $state(true);
	discreet = $state(false);
	reason = $state('');
	theme = $state('system');
	accent = $state('opal');
	generation = $state(0);
	celebrated = $state<string[]>([]);
	chat = $state<
		{ role: 'user' | 'assistant'; text: string; tier: string; state?: string; facts?: any[] }[]
	>([]);
	api: API;
	constructor() {
		if (typeof sessionStorage !== 'undefined') {
			try {
				this.token = sessionStorage.getItem('opale.token');
				this.profile = JSON.parse(sessionStorage.getItem('opale.profile') || 'null');
				this.expiresAt = sessionStorage.getItem('opale.expires') || '';
				this.baseURL = localStorage.getItem('opale.baseURL') || '';
				// Retire the older indefinitely persisted bearer token.
				localStorage.removeItem('opale.token');
				localStorage.removeItem('opale.profile');
			} catch {
				this.token = null;
				this.profile = null;
			}
		}
		this.api = this.client();
	}
	private client() {
		return new API(
			this.baseURL,
			() => this.token,
			() => this.invalidate()
		);
	}
	get loggedIn() {
		return Boolean(this.token && this.profile && !this.locked);
	}
	preferenceKey(key: string) {
		return `opale.${this.profile?.id ?? 'guest'}.${key}`;
	}
	readPreference(key: string, fallback: string) {
		return typeof localStorage === 'undefined'
			? fallback
			: (localStorage.getItem(this.preferenceKey(key)) ?? fallback);
	}
	savePreference(key: string, value: string) {
		localStorage.setItem(this.preferenceKey(key), value);
	}
	setBaseURL(url: string) {
		const trimmed = url.trim().replace(/\/+$/, '');
		if (trimmed) {
			const parsed = new URL(trimmed);
			if (!['https:', 'http:'].includes(parsed.protocol))
				throw new Error('Adresse HTTP ou HTTPS requise.');
		}
		this.clear();
		this.baseURL = trimmed;
		localStorage.setItem('opale.baseURL', trimmed);
		this.api = this.client();
	}
	lock() {
		this.locked = true;
		this.generation++;
	}
	invalidate() {
		this.clear();
		this.reason = 'Ta session a expiré. Reconnecte-toi.';
	}
	clear() {
		this.chat = [];
		this.celebrated = [];
		this.token = null;
		this.profile = null;
		this.expiresAt = '';
		this.locked = true;
		this.generation++;
		if (typeof sessionStorage !== 'undefined')
			['opale.token', 'opale.profile', 'opale.expires'].forEach((k) =>
				sessionStorage.removeItem(k)
			);
	}
	async login(profileID: string, pin: string) {
		const oldToken = this.token;
		const oldProfile = this.profile?.id;
		const res = await this.api.login(profileID, pin);
		if (oldProfile !== res.profile.id) {
			this.chat = [];
			this.celebrated = [];
		}
		this.token = res.token;
		this.profile = res.profile;
		this.expiresAt = res.expires_at;
		this.locked = false;
		this.reason = '';
		this.generation++;
		sessionStorage.setItem('opale.token', res.token);
		sessionStorage.setItem('opale.profile', JSON.stringify(res.profile));
		sessionStorage.setItem('opale.expires', res.expires_at);
		this.theme = this.readPreference('theme', 'system');
		this.accent = this.readPreference('accent', 'opal');
		this.discreet = this.readPreference('discreet', 'false') === 'true';
		// Revoke the previous tab session without coupling successful sign-in to its availability.
		if (oldToken && oldToken !== res.token)
			void new API(this.baseURL, () => oldToken).logout().catch(() => {});
	}
	async logout() {
		const token = this.token;
		const api = new API(this.baseURL, () => token);
		this.clear(); // Hide private content immediately even if the network is unavailable.
		await api.logout();
	}
}
export const session = new Session();
