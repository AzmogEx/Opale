<script lang="ts">
	import DataView from './DataView.svelte';
	import { money } from '$lib/domain';
	import { kindLabels } from '$lib/api';
	let {
		data,
		depth = 0,
		currency = 'EUR'
	}: { data: any; depth?: number; currency?: string } = $props();
	const localCurrency = $derived(data?.currency ?? data?.asset?.currency ?? currency);
	let expanded = $state(false);
	const labels: Record<string, string> = {
		merchant_key: 'Marchand',
		performance: 'Performance',
		horizon_months: 'Horizon (mois)',
		total_fees_cents: 'Frais totaux',
		investment_return_bps: 'Rendement supposé',
		coverage_complete: 'Historique complet',
		initial_capital_cents: 'Capital initial',
		value_eur_cents: 'Valeur convertie en EUR',
		last_date: 'Dernière valorisation',
		assessment_year: 'Année d’imposition',
		income_year: 'Année des revenus',
		verified_on: 'Paramètres vérifiés le',
		source_url: 'Source officielle',
		deductible_cents: 'Versement déductible',
		per_ceiling_cents: 'Plafond disponible',
		share_bps: 'Part',
		expires_at: 'Expiration',
		start_cash_cents: 'Trésorerie actuelle',
		end_cash_cents: 'Trésorerie projetée',
		until: 'Date projetée',
		upcoming: 'Échéances à venir',
		contributions_cents: 'Versements',
		withdrawals_cents: 'Retraits',
		distributions_cents: 'Distributions',
		fees_cents: 'Frais',
		gain_cents: 'Gain de marché',
		return_bps: 'Performance',
		known: 'Historique suffisant',
		reason: 'Précision',
		final_net_cents: 'Patrimoine final nominal',
		real_final_net_cents: 'Patrimoine final en euros constants',
		initial_outflow_cents: 'Sortie initiale',
		final_liquid_cents: 'Liquidités finales',
		asset_value_cents: 'Valeur du bien',
		remaining_debt_cents: 'Capital restant dû',
		minimum_liquid_cents: 'Trésorerie minimale',
		feasible: 'Hypothèse finançable',
		assumptions: 'Hypothèses',
		sensitivity: 'Sensibilité',
		delta_b_minus_a_cents: 'Avantage final de B sur A',
		last_error: 'Dernière erreur',
		next_sync_at: 'Prochaine synchronisation',
		sync_status: 'État de synchronisation',
		balance_reconciled: 'Solde rapproché',
		balance_date: 'Date du solde',
		balance_type: 'Nature du solde',
		name: 'Nom',
		label: 'Libellé',
		title: 'Titre',
		detail: 'Détail',
		note: 'Note',
		status: 'État',
		kind: 'Type',
		currency: 'Devise',
		score: 'Score',
		comment: 'Analyse',
		components: 'Composantes',
		amount_cents: 'Montant',
		value_cents: 'Valeur',
		cash_cents: 'Trésorerie',
		net_cents: 'Patrimoine net',
		total_cents: 'Total',
		income_cents: 'Revenus',
		expenses_cents: 'Dépenses',
		remaining_cents: 'Restant',
		spent_cents: 'Dépensé',
		target_cents: 'Cible',
		progress_cents: 'Progression',
		monthly_budget_cents: 'Budget mensuel',
		monthly_savings_cents: 'Épargne mensuelle',
		target_date: 'Échéance souhaitée',
		estimated_date: 'Date estimée',
		estimate_reason: 'Estimation',
		on_track: 'Sur la trajectoire',
		percent: 'Progression (%)',
		asset_name: 'Actif',
		category_name: 'Catégorie',
		category: 'Catégorie',
		role: 'Rôle',
		phone: 'Téléphone',
		email: 'E-mail',
		created_at: 'Création',
		recorded_at: 'Instant de capture observé (pas nécessairement la clôture mensuelle)',
		as_of: 'Date de valeur',
		date: 'Date',
		year: 'Année',
		month: 'Mois',
		months: 'Durée (mois)',
		days: 'Horizon (jours)',
		at_5y_cents: 'À 5 ans',
		at_10y_cents: 'À 10 ans',
		at_end_cents: 'À terme',
		annual_return_bps: 'Rendement annuel',
		annual_rate_bps: 'Taux annuel',
		inflation_bps: 'Inflation',
		monthly_payment_cents: 'Mensualité',
		principal_cents: 'Capital',
		total_interest_cents: 'Intérêts',
		total_paid_cents: 'Total remboursé',
		capacity_cents: 'Capacité d’emprunt',
		schedule: 'Échéancier',
		points: 'Trajectoire',
		lines: 'Répartition',
		class: 'Classe',
		actual_cents: 'Valeur actuelle',
		actual_bps: 'Part actuelle',
		target_bps: 'Part cible',
		drift_cents: 'Écart à combler',
		has_targets: 'Cible définie',
		enabled: 'Activée',
		threshold_cents: 'Seuil',
		saved_cents: 'Épargne',
		savings_rate_bps: 'Taux d’épargne',
		active_months: 'Mois actifs',
		transaction_count: 'Opérations',
		top_categories: 'Principaux postes',
		top_merchants: 'Principaux marchands',
		biggest_expense: 'Dépense principale',
		occurred_on: 'Date',
		net_worth_start_cents: 'Patrimoine initial',
		net_worth_end_cents: 'Patrimoine final',
		investments: 'Investissements',
		asset: 'Actif',
		details: 'Caractéristiques',
		properties: 'Biens immobiliers',
		objects: 'Objets',
		companies: 'Sociétés',
		contacts: 'Contacts',
		events: 'Chronologie',
		documents: 'Documents',
		gross_yield_bps: 'Rendement brut',
		monthly_cashflow_cents: 'Flux mensuel net',
		equity_cents: 'Part nette',
		purchase_price_cents: 'Prix d’achat',
		purchase_date: 'Acquisition',
		capital_gain_cents: 'Plus-value latente',
		change_cents: 'Variation de valeur',
		change_bps: 'Variation (%)',
		first_value_cents: 'Valeur initiale',
		first_date: 'Première valorisation',
		allocation_bps: 'Répartition',
		loan_remaining_cents: 'Crédit restant',
		monthly_rent_cents: 'Loyer mensuel',
		monthly_charges_cents: 'Charges mensuelles',
		monthly_loan_payment_cents: 'Mensualité du crédit',
		property_tax_yearly_cents: 'Taxe foncière annuelle',
		company_value_cents: 'Valorisation société',
		my_total_cents: 'Valeur personnelle',
		dividends_net_cents: 'Dividendes estimés nets',
		ownership_bps: 'Détention',
		cca_cents: 'Compte courant d’associé',
		annual_dividends_cents: 'Dividendes annuels',
		monthly_salary_cents: 'Salaire mensuel',
		brand: 'Marque',
		insured: 'Assuré',
		narrative: 'Analyse',
		narrative_tier: 'Origine',
		summary: 'Résumé',
		risks: 'Risques',
		health_score: 'Santé financière',
		independence: 'Indépendance',
		reached: 'Atteinte',
		homelab_available: 'Homelab disponible',
		cloud_configured: 'Cloud configuré',
		available: 'Disponible',
		vault_enabled: 'Coffre configuré',
		configured: 'Configuré',
		last_synced_at: 'Dernière synchronisation',
		institution_name: 'Banque',
		error: 'Erreur',
		imported: 'Importées',
		duplicates: 'Doublons évités',
		categorized: 'Catégorisées',
		mime: 'Format',
		size_bytes: 'Taille (octets)',
		members: 'Membres',
		transactions: 'Opérations',
		profile_name: 'Profil',
		balance_cents: 'Solde',
		frequency: 'Fréquence',
		active: 'Active',
		next_date: 'Prochaine échéance',
		end_date: 'Fin',
		source: 'Source',
		provider: 'Fournisseur',
		updated_at: 'Mise à jour',
		estimate: 'Estimation',
		per_effect: 'Effet du PER',
		deadlines: 'Échéances indicatives',
		tax_cents: 'Impôt estimé',
		tax_before_cents: 'Impôt avant',
		tax_after_cents: 'Impôt après',
		saving_cents: 'Économie estimée',
		marginal_rate_bps: 'Taux marginal',
		effective_rate_bps: 'Taux effectif',
		disclaimer: 'Hypothèses',
		risk_level: 'Niveau de risque',
		recommendation: 'Lecture du résultat',
		scenarios: 'Scénarios',
		impact: 'Impact',
		affordable_cash: 'Finançable au comptant',
		savings_after_cents: 'Épargne restante',
		a: 'Scénario A',
		b: 'Scénario B',
		delta_end_cents: 'Écart final',
		rates: 'Taux',
		missing_currencies: 'Devises sans taux',
		rate_micro: 'Micro-euros par unité',
		yearly_cost_cents: 'Coût annuel',
		monthly_cost_cents: 'Coût mensuel',
		subscriptions: 'Abonnements',
		recurring: 'Récurrences',
		cashflow: 'Prévisions',
		amount: 'Montant',
		future: 'À venir'
	};
	const hidden =
		/(^id$|_id$|sha256|privacy_default|token|profile_id|requisition_id|currency_exponent|^created_by$)/;
	const visible = (key: string) =>
		!hidden.test(key) &&
		!(data?.performance && ['change_cents', 'change_bps'].includes(key)) &&
		!(data?.known === false && ['gain_cents', 'return_bps'].includes(key));
	const values: Record<string, string> = {
		planned: 'Prévu',
		excluded: 'Exclu',
		realized: 'Réalisé',
		once: 'Une fois',
		weekly: 'Chaque semaine',
		monthly: 'Chaque mois',
		yearly: 'Chaque année',
		stocks: 'Actions et placements',
		buy_rent: 'Acheter ou louer',
		cash_credit: 'Comptant ou crédit',
		repay_invest: 'Rembourser ou investir',
		cash: 'Liquidités',
		synced: 'Synchronisé',
		needs_mapping: 'Association requise',
		cooldown: 'Délai avant nouvelle synchronisation',
		pending_consent: 'Consentement en attente',
		renew_consent: 'Consentement à renouveler',
		error: 'Erreur',
		contribution: 'Versement',
		withdrawal: 'Retrait',
		distribution: 'Distribution',
		fee: 'Frais',
		trusted: 'Proche de confiance',
		notary: 'Notaire',
		banker: 'Banquier',
		insurer: 'Assureur',
		accountant: 'Comptable',
		expense_income: 'Revenu ou dépense',
		internal_transfer: 'Virement interne',
		loan_principal: 'Capital remboursé',
		cash_below: 'Trésorerie sous le seuil',
		net_worth_below: 'Patrimoine sous le seuil',
		expenses_month_above: 'Dépenses au-dessus du seuil'
	};
	const display = (key: string, value: any) =>
		key.endsWith('_cents') && typeof value === 'number'
			? money(value, key.endsWith('_eur_cents') ? 'EUR' : localCurrency)
			: key.endsWith('_bps') && typeof value === 'number'
				? `${(value / 100).toLocaleString('fr-FR')} %`
				: typeof value === 'boolean'
					? value
						? 'Oui'
						: 'Non'
					: (kindLabels[String(value)] ?? values[String(value)] ?? String(value));
</script>

{#if data === null || data === undefined}<p class="muted">Aucune donnée disponible.</p>
{:else if Array.isArray(data)}
	{#if !data.length}<p class="muted">Aucun élément pour cette période.</p>{/if}
	<div class="data-list">
		{#each data.slice(0, expanded ? data.length : 12) as row, i}<div class="data-row">
				<DataView data={row} currency={localCurrency} depth={depth + 1} />
			</div>{/each}
	</div>
	{#if data.length > 12}<button onclick={() => (expanded = !expanded)}
			>{expanded ? 'Réduire' : `Afficher les ${data.length} éléments`}</button
		>{/if}
{:else if typeof data === 'object'}
	<dl class="data-grid">
		{#each Object.entries(data).filter(([k, v]) => visible(k) && v !== null && v !== undefined && v !== '') as [key, value]}
			{#if typeof value !== 'object'}<div>
					<dt>{labels[key] ?? key.replace(/_/g, ' ')}</dt>
					<dd>
						{#if key === 'source_url' && typeof value === 'string' && value.startsWith('https://')}<a
								href={value}
								target="_blank"
								rel="noopener noreferrer">Consulter la source officielle ↗</a
							>{:else}{display(key, value)}{/if}
					</dd>
				</div>{/if}{/each}
	</dl>
	{#each Object.entries(data).filter(([k, v]) => visible(k) && v !== null && typeof v === 'object') as [key, value]}<details
			open={depth < 1 && !['points', 'schedule', 'transactions'].includes(key)}
		>
			<summary>{labels[key] ?? key.replace(/_/g, ' ')}</summary><DataView
				data={value}
				currency={localCurrency}
				depth={depth + 1}
			/>
		</details>{/each}
{:else}<p>{data}</p>{/if}
