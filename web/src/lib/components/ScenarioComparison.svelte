<script lang="ts">
	import { session } from '$lib/session.svelte';
	import { field } from '$lib/forms';
	import Form from './Form.svelte';
	import DataView from './DataView.svelte';
	let result = $state<any>(null);
	const fields = [
		field('months', 'Horizon (mois)', 'number', { required: true, default: 120, min: 1, max: 600 }),
		...['a', 'b'].flatMap((k) => [
			field(`${k}_label`, `${k.toUpperCase()} · Nom`, 'text', {
				required: true,
				default: k === 'a' ? 'Projet actuel' : 'Alternative'
			}),
			field(`${k}_monthly_savings_cents`, `${k.toUpperCase()} · Épargne mensuelle`, 'money', {
				default: 50000
			}),
			field(`${k}_monthly_expenses_cents`, `${k.toUpperCase()} · Dépenses mensuelles`, 'money', {
				default: 200000
			}),
			field(`${k}_annual_return_bps`, `${k.toUpperCase()} · Rendement annuel (%)`, 'percent', {
				default: 500
			}),
			field(`${k}_one_time_cost_cents`, `${k.toUpperCase()} · Dépense initiale`, 'money')
		])
	];
	async function submit(v: Record<string, any>) {
		const scenario = (key: string) =>
			Object.fromEntries(
				Object.entries(v)
					.filter(([k]) => k.startsWith(key + '_'))
					.map(([k, val]) => [k.slice(2), val])
			);
		result = await session.api.request('POST', '/v1/scenarios/compare', {
			months: v.months,
			a: scenario('a'),
			b: scenario('b')
		});
	}
</script>

<section class="glass panel">
	<h2>Deux futurs côte à côte</h2>
	<p class="muted">
		Même patrimoine initial, deux jeux d’hypothèses. Montants nominaux en euros ; rendements non
		garantis.
	</p>
	<Form {fields} {submit} label="Comparer" />{#if result}<DataView data={result} />{/if}
</section>
