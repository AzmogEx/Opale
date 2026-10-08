<script lang="ts">
	import { session } from '$lib/session.svelte';
	import ResourcePanel from './ResourcePanel.svelte';
	import { field } from '$lib/forms';
	let assets = $state<any[]>([]);
	let error = $state('');
	$effect(() => {
		void session.api
			.listAssets()
			.then((a) => (assets = a))
			.catch((e) => (error = e.message));
	});
</script>

{#if error}<p class="notice error" role="alert">{error}</p>{/if}<ResourcePanel
	title="Objectifs"
	path="/v1/goals/"
	listKey="goals"
	description="L’épargne affectée à chaque objectif est réservée à ce projet, dans la limite de la capacité observée sur les trois derniers mois. Sans revenus saisis, laisse ce rythme à zéro. Une date indéterminée signifie que le rythme ne permet pas encore une estimation."
	fields={[
		field('name', 'Nom de l’objectif', 'text', { required: true }),
		field('icon', 'Icône (emoji)', 'text', { default: '🎯' }),
		field('target_cents', 'Montant cible (EUR)', 'money', { required: true }),
		field('target_date', 'Échéance souhaitée', 'date'),
		field('monthly_savings_cents', 'Épargne mensuelle affectée (EUR)', 'money'),
		field('asset_id', 'Actif dédié (facultatif)', 'select', {
			options: assets.filter((a) => !a.archived).map((a) => ({ value: a.id, label: a.name }))
		})
	]}
/>
