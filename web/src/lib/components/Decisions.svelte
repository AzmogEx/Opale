<script lang="ts">
	import { session } from '$lib/session.svelte';
	import { money } from '$lib/domain';
	import Form from './Form.svelte';
	import DataView from './DataView.svelte';
	import { field, options } from '$lib/forms';
	let result = $state<any>(null);
	let horizon = $state(120);
	async function submit(value: Record<string, any>) {
		result = null;
		const next = await session.api.request<any>('POST', '/v1/decisions/compare', value);
		horizon = value.horizon_months;
		result = next;
	}
	const scenarioNames: Record<string, string> = {
		prudent: 'Prudent',
		normal: 'Normal',
		ambitieux: 'Ambitieux'
	};
	const fields = [
		field('kind', 'Décision', 'select', {
			required: true,
			default: 'buy_rent',
			options: options({
				buy_rent: 'Acheter ou louer',
				cash_credit: 'Comptant ou crédit',
				repay_invest: 'Rembourser ou investir'
			})
		}),
		field('horizon_months', 'Horizon commun (mois)', 'number', {
			required: true,
			default: 120,
			min: 1,
			max: 600
		}),
		field('initial_cash_cents', 'Capital disponible (EUR)', 'money', { required: true }),
		field('monthly_budget_cents', 'Budget mensuel avant logement / crédit (EUR)', 'money', {
			required: true
		}),
		field('asset_price_cents', 'Prix du bien (EUR)', 'money'),
		field('down_payment_cents', 'Apport (EUR)', 'money'),
		field('purchase_fees_cents', 'Frais d’acquisition (EUR)', 'money'),
		field('monthly_rent_cents', 'Loyer initial (EUR/mois)', 'money'),
		field('monthly_ownership_cost_cents', 'Charges propriétaire (EUR/mois)', 'money'),
		field('loan_principal_cents', 'Capital emprunté ou restant dû (EUR)', 'money'),
		field('loan_rate_bps', 'Taux annuel du crédit (%)', 'percent'),
		field('loan_months', 'Durée restante du crédit (mois)', 'number', {
			default: 240,
			min: 1,
			max: 600
		}),
		field('repayment_cents', 'Remboursement anticipé (EUR)', 'money'),
		field('repayment_fee_cents', 'Indemnité de remboursement (EUR)', 'money'),
		field('investment_return_bps', 'Rendement d’investissement supposé (%)', 'percent', {
			default: 400
		}),
		field('asset_growth_bps', 'Évolution annuelle du bien supposée (%)', 'percent'),
		field('rent_growth_bps', 'Évolution annuelle du loyer (%)', 'percent'),
		field('inflation_bps', 'Inflation annuelle supposée (%)', 'percent', { default: 200 }),
		field('sale_fee_bps', 'Frais de revente (%)', 'percent')
	];
</script>

<section class="glass panel">
	<h2>Comparer deux décisions</h2>
	<p class="muted">
		Les deux alternatives utilisent le même capital, le même budget et le même horizon. Laisse à
		zéro les paramètres non utilisés. Les rendements et l’inflation sont des hypothèses, jamais une
		promesse.
	</p>
	<Form {fields} {submit} label="Calculer" />
	{#if result}<section class="subtle" aria-live="polite" aria-label="Résultat de la comparaison">
			<h3>Résultat</h3>
			<p>Horizon choisi : {horizon} mois. A : {result.a.label}. B : {result.b.label}.</p>
			{#if result.recommendation}<div class="notice">
					<h4>Orientation conditionnelle</h4>
					<p>{result.recommendation.message}</p>
				</div>{/if}
			<div class="comparison-grid">
				<section aria-label="Alternative A">
					<h4>A · {result.a.label}</h4>
					<DataView data={result.a} />
				</section>
				<section aria-label="Alternative B">
					<h4>B · {result.b.label}</h4>
					<DataView data={result.b} />
				</section>
			</div>
			{#if result.timeline?.length}<h4>Impact à 0, 5 et 10 ans</h4>
				<p class="muted">
					Hypothèses centrales identiques ; les opérations initiales et les frais sont déjà intégrés
					à 0 an. Un écart positif favorise B.
				</p>
				<div class="table-scroll">
					<table>
						<caption>Patrimoine et liquidités en euros aux trois dates</caption>
						<thead
							><tr
								><th scope="col">Horizon</th><th scope="col">A · patrimoine net</th><th scope="col"
									>B · patrimoine net</th
								><th scope="col">Écart B − A</th><th scope="col">A · liquidités</th><th scope="col"
									>B · liquidités</th
								><th scope="col">Financement</th></tr
							></thead
						>
						<tbody
							>{#each result.timeline as point}<tr
									><th scope="row">{point.months / 12} {point.months === 0 ? 'an' : 'ans'}</th><td
										>{money(point.a.final_net_cents)}</td
									><td>{money(point.b.final_net_cents)}</td><td
										>{money(point.delta_b_minus_a_cents)}</td
									><td>{money(point.a.final_liquid_cents)}</td><td
										>{money(point.b.final_liquid_cents)}</td
									><td
										>A : {point.a.feasible ? 'financé' : 'déficit'} · B : {point.b.feasible
											? 'financé'
											: 'déficit'}</td
									></tr
								>{/each}</tbody
						>
					</table>
				</div>
			{/if}
			{#if result.scenarios?.length}<h4>Trois scénarios à {horizon} mois</h4>
				<p class="muted">
					Ces variantes mesurent la sensibilité au rendement supposé. Elles ne sont pas des
					probabilités de résultat.
				</p>
				<div class="table-scroll">
					<table>
						<caption>Comparaison prudent, normal et ambitieux</caption>
						<thead
							><tr
								><th scope="col">Scénario</th><th scope="col">Rendement annuel supposé</th><th
									scope="col">A · patrimoine net</th
								><th scope="col">B · patrimoine net</th><th scope="col">Écart B − A</th><th
									scope="col">Financement</th
								></tr
							></thead
						>
						<tbody
							>{#each result.scenarios as scenario}<tr
									><th scope="row">{scenarioNames[scenario.name] ?? scenario.name}</th><td
										>{(scenario.investment_return_bps / 100).toLocaleString('fr-FR')} %</td
									><td>{money(scenario.a.final_net_cents)}</td><td
										>{money(scenario.b.final_net_cents)}</td
									><td>{money(scenario.delta_b_minus_a_cents)}</td><td
										>A : {scenario.a.feasible ? 'financé' : 'déficit'} · B : {scenario.b.feasible
											? 'financé'
											: 'déficit'}</td
									></tr
								>{/each}</tbody
						>
					</table>
				</div>
			{/if}
			{#if result.risks?.length}<h4>Risques et limites</h4>
				<ul>
					{#each result.risks as risk}<li>{risk}</li>{/each}
				</ul>{/if}
			{#if result.assumptions?.length}<h4>Hypothèses de calcul</h4>
				<ul>
					{#each result.assumptions as assumption}<li>{assumption}</li>{/each}
				</ul>{/if}
			{#if result.sensitivity?.length}<details>
					<summary>Sensibilité détaillée</summary><DataView data={result.sensitivity} />
				</details>{/if}
		</section>{/if}
</section>

<style>
	.panel {
		min-width: 0;
	}
	.comparison-grid {
		display: grid;
		grid-template-columns: repeat(2, minmax(0, 1fr));
		gap: 1.25rem;
	}
	caption {
		text-align: left;
		padding: 0.75rem 0;
	}
	@media (max-width: 700px) {
		.comparison-grid {
			grid-template-columns: 1fr;
		}
	}
</style>
