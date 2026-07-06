// Session Opale — état global en runes, persisté en localStorage.
// Même philosophie que SessionStore.swift : un jeton porteur, un profil,
// une base d'API configurable (dev : proxy vite → '' suffit).

import { API, type Profile } from './api';

const TOKEN_KEY = 'opale.token';
const PROFILE_KEY = 'opale.profile';
const BASE_KEY = 'opale.baseURL';

class Session {
	token = $state<string | null>(null);
	profile = $state<Profile | null>(null);
	baseURL = $state('');
	/** Mode discret (EF-004) : floute tous les montants. */
	discreet = $state(false);

	api: API;

	constructor() {
		if (typeof localStorage !== 'undefined') {
			this.token = localStorage.getItem(TOKEN_KEY);
			this.baseURL = localStorage.getItem(BASE_KEY) ?? '';
			const raw = localStorage.getItem(PROFILE_KEY);
			if (raw) this.profile = JSON.parse(raw) as Profile;
		}
		this.api = new API(this.baseURL, () => this.token);
	}

	get loggedIn(): boolean {
		return this.token !== null;
	}

	setBaseURL(url: string) {
		this.baseURL = url.replace(/\/+$/, '');
		localStorage.setItem(BASE_KEY, this.baseURL);
		this.api = new API(this.baseURL, () => this.token);
	}

	async login(profileID: string, pin: string) {
		const res = await this.api.login(profileID, pin);
		this.token = res.token;
		this.profile = res.profile;
		localStorage.setItem(TOKEN_KEY, res.token);
		localStorage.setItem(PROFILE_KEY, JSON.stringify(res.profile));
	}

	async logout() {
		try {
			await this.api.logout();
		} catch {
			/* la session locale se ferme quoi qu'il arrive */
		}
		this.token = null;
		this.profile = null;
		localStorage.removeItem(TOKEN_KEY);
		localStorage.removeItem(PROFILE_KEY);
	}
}

export const session = new Session();
