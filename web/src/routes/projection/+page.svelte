<script lang="ts">
	import ProjectionSimulator from '$lib/components/ProjectionSimulator.svelte';
	import Goals from '$lib/components/Goals.svelte';
	import Decisions from '$lib/components/Decisions.svelte';
	import ScenarioComparison from '$lib/components/ScenarioComparison.svelte';
	import Calculator from '$lib/components/Calculator.svelte';
	import RemotePanel from '$lib/components/RemotePanel.svelte';
	import { field } from '$lib/forms';
	let section = $state('fire');
	const sections = {
		fire: 'Indépendance',
		goals: 'Objectifs',
		decisions: 'Décisions',
		scenarios: 'Scénarios',
		loan: 'Crédit',
		tax: 'Fiscalité et PER',
		timeline: 'Chronologie'
	};
</script>

<svelte:head><title>Projection · Opale</title></svelte:head>
<h1>Projection</h1>
<div class="tabstrip" aria-label="Sections projection">
	{#each Object.entries(sections) as [key, label]}<button
			aria-pressed={section === key}
			onclick={() => (section = key)}>{label}</button
		>{/each}
</div>
<div class="stack">
	{#key section}{#if section === 'fire'}<ProjectionSimulator />{:else if section === 'goals'}<Goals
			/>{:else if section === 'decisions'}<Decisions
			/>{:else if section === 'scenarios'}<ScenarioComparison
			/>{:else if section === 'loan'}<Calculator
				title="Simuler un crédit"
				path="/v1/loan/simulate"
				description="Crédit à taux fixe, hors assurance et frais. Consulte l’échéancier pour distinguer intérêts et remboursement du capital."
				fields={[
					field('principal_cents', 'Capital emprunté (EUR)', 'money', {
						required: true,
						default: 20000000
					}),
					field('annual_rate_bps', 'Taux annuel (%)', 'percent', { required: true, default: 350 }),
					field('months', 'Durée (mois)', 'number', {
						required: true,
						default: 240,
						min: 1,
						max: 600
					})
				]}
			/><Calculator
				title="Capacité d’emprunt"
				path="/v1/loan/capacity"
				description="Capital finançable pour la mensualité choisie ; ce calcul ne constitue pas un accord bancaire."
				fields={[
					field('monthly_payment_cents', 'Mensualité disponible (EUR)', 'money', {
						required: true,
						default: 100000
					}),
					field('annual_rate_bps', 'Taux annuel (%)', 'percent', { required: true, default: 350 }),
					field('months', 'Durée (mois)', 'number', {
						required: true,
						default: 240,
						min: 1,
						max: 600
					})
				]}
			/>{:else if section === 'tax'}<Calculator
				title="Impôt sur le revenu et effet du PER"
				path="/v1/tax/estimate"
				method="GET"
				description="Estimation indicative selon le millésime et les limites renvoyés par le moteur. Les réductions, crédits d’impôt et situations particulières peuvent modifier le résultat."
				fields={[
					field('income_cents', 'Revenu annuel imposable (EUR)', 'money', { required: true }),
					field('parts_tenths', 'Parts fiscales (dixièmes)', 'number', {
						required: true,
						default: 10,
						min: 10,
						max: 100,
						help: '10 = 1 part ; 15 = 1,5 part ; 20 = 2 parts.'
					}),
					field('year', 'Année d’imposition', 'number', {
						required: true,
						default: 2026,
						min: 2026,
						max: 2026
					}),
					field('per_ceiling_cents', 'Plafond PER disponible sur ton avis (EUR)', 'money', {
						help: 'Requis pour évaluer un versement PER.'
					}),
					field('per_cents', 'Versement PER déductible (EUR)', 'money', {
						help: 'Dans la limite de ton plafond personnel de déduction.'
					})
				]}
			/><RemotePanel
				title="Échéances fiscales indicatives"
				path="/v1/tax/deadlines"
			/>{:else if section === 'timeline'}<RemotePanel
				title="Ta chronologie patrimoniale"
				path="/v1/timeline"
			/>{/if}{/key}
</div>
