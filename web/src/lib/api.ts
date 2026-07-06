// Client API Opale — miroir typé de l'API Go (montants en CENTIMES entiers,
// jamais de float côté domaine : les nombres JS ne servent qu'à l'affichage).

export interface Profile {
	id: string;
	name: string;
}

export interface NetWorth {
	assets_total_cents: number;
	liabilities_total_cents: number;
	net_cents: number;
	currency: string;
}

export interface NetWorthPoint {
	as_of: string;
	net_cents: number;
}

export interface Asset {
	id: string;
	name: string;
	kind: string;
	currency: string;
	latest_value_cents: number | null;
	theoretical_cents?: number;
	archived: boolean;
}

export interface Liability {
	id: string;
	name: string;
	kind: string;
	latest_value_cents: number | null;
	archived: boolean;
}

export interface Transaction {
	id: string;
	amount_cents: number;
	occurred_on: string;
	label: string;
	category_name?: string;
}

export interface MonthSummary {
	income_cents: number;
	expenses_cents: number;
	net_cents: number;
}

export interface HealthComponent {
	name: string;
	score: number;
	max: number;
	comment: string;
}

export interface HealthScore {
	score: number;
	components: HealthComponent[];
}

export interface OpaleAlert {
	kind: string;
	severity: string;
	title: string;
	detail: string;
}

export interface Risk {
	id: string;
	title: string;
	severity: string;
	detail: string;
}

export interface Independence {
	reached: boolean;
	months: number;
	target_cents: number;
}

export interface ProjectionPoint {
	month: number;
	net_cents: number;
}

export interface Projection {
	start_net_cents: number;
	annual_return_bps: number;
	inflation_bps: number;
	points: ProjectionPoint[];
	independence: Independence;
}

export interface AskResponse {
	answer: string;
	tier: string; // "n2" | "n3" | "" (repli moteur)
}

export interface AssistantStatus {
	homelab_available: boolean;
	cloud_configured: boolean;
}

export class APIError extends Error {
	constructor(
		public status: number,
		message: string
	) {
		super(message);
	}
}

/** Libellés français des types d'actifs (alignés sur l'app iOS). */
export const kindLabels: Record<string, string> = {
	checking: 'Compte courant',
	savings: 'Livret / épargne',
	life_insurance: 'Assurance-vie',
	pea: 'PEA',
	cto: 'Compte-titres',
	crypto: 'Crypto',
	real_estate: 'Immobilier',
	precious_metal: 'Or / métaux',
	vehicle: 'Véhicule',
	valuable: 'Objet de valeur',
	company_share: 'Parts de société',
	other: 'Autre',
	mortgage: 'Crédit immobilier',
	auto_loan: 'Crédit auto',
	consumer_loan: 'Crédit conso'
};

/** Client HTTP minimal : base configurable, jeton porteur, erreurs typées. */
export class API {
	constructor(
		private base: string,
		private token: () => string | null
	) {}

	private async request<T>(method: string, path: string, body?: unknown): Promise<T> {
		const headers: Record<string, string> = { 'Content-Type': 'application/json' };
		const token = this.token();
		if (token) headers.Authorization = `Bearer ${token}`;

		const res = await fetch(this.base + path, {
			method,
			headers,
			body: body === undefined ? undefined : JSON.stringify(body)
		});
		if (!res.ok) {
			let message = `HTTP ${res.status}`;
			try {
				const data = await res.json();
				message = data?.error?.message ?? message;
			} catch {
				/* corps non JSON */
			}
			throw new APIError(res.status, message);
		}
		if (res.status === 204) return undefined as T;
		return (await res.json()) as T;
	}

	// ── Auth ──────────────────────────────────────────────────────────────
	listProfiles = () =>
		this.request<{ profiles: Profile[] }>('GET', '/v1/profiles').then((r) => r.profiles ?? []);
	login = (profile_id: string, pin: string) =>
		this.request<{ token: string; profile: Profile }>('POST', '/v1/auth/login', {
			profile_id,
			pin
		});
	logout = () => this.request<void>('POST', '/v1/auth/logout');

	// ── Patrimoine ────────────────────────────────────────────────────────
	netWorth = () => this.request<NetWorth>('GET', '/v1/net-worth');
	netWorthHistory = (months = 12) =>
		this.request<{ points: NetWorthPoint[] }>('GET', `/v1/net-worth/history?months=${months}`);
	listAssets = () =>
		this.request<{ assets: Asset[] }>('GET', '/v1/assets/').then((r) => r.assets ?? []);
	listLiabilities = () =>
		this.request<{ liabilities: Liability[] }>('GET', '/v1/liabilities/').then(
			(r) => r.liabilities ?? []
		);

	// ── Flux ──────────────────────────────────────────────────────────────
	listTransactions = (from: string, to: string) =>
		this.request<{ transactions: Transaction[] }>(
			'GET',
			`/v1/transactions/?from=${from}&to=${to}`
		).then((r) => r.transactions ?? []);
	monthSummary = (year: number, month: number) =>
		this.request<MonthSummary>('GET', `/v1/transactions/summary?year=${year}&month=${month}`);

	// ── Pilotage & cerveau ────────────────────────────────────────────────
	healthScore = () => this.request<HealthScore>('GET', '/v1/health-score');
	alerts = () =>
		this.request<{ alerts: OpaleAlert[] }>('GET', '/v1/alerts').then((r) => r.alerts ?? []);
	risks = () => this.request<{ risks: Risk[] }>('GET', '/v1/risks').then((r) => r.risks ?? []);
	projection = (p: {
		savings: number;
		returnBps: number;
		expenses: number;
		inflationBps: number;
	}) =>
		this.request<Projection>(
			'GET',
			`/v1/projection?monthly_savings_cents=${p.savings}&annual_return_bps=${p.returnBps}` +
				`&monthly_expenses_cents=${p.expenses}&inflation_bps=${p.inflationBps}`
		);
	ask = (question: string, allow_cloud = false) =>
		this.request<AskResponse>('POST', '/v1/assistant/ask', { question, allow_cloud });
	assistantStatus = () => this.request<AssistantStatus>('GET', '/v1/assistant/status');
}
