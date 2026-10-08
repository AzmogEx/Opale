<script lang="ts">
	import { session } from '$lib/session.svelte';
	import { field } from '$lib/forms';
	import { messageOf } from '$lib/domain';
	import Form from './Form.svelte';
	import DataView from './DataView.svelte';
	let data = $state<any>(null);
	let initial = $state<any>({});
	let error = $state('');
	const classes = {
		stocks: 'Actions et placements (%)',
		real_estate: 'Immobilier (%)',
		crypto: 'Crypto (%)',
		cash: 'Liquidités (%)',
		other: 'Autres (%)'
	};
	async function load() {
		try {
			data = await session.api.request<any>('GET', '/v1/allocation/');
			initial = Object.fromEntries((data.lines ?? []).map((l: any) => [l.class, l.target_bps]));
		} catch (e) {
			error = messageOf(e);
		}
	}
	$effect(() => {
		void load();
	});
	async function save(v: Record<string, any>) {
		if (Object.values(v).reduce((s, n) => s + Number(n), 0) !== 10000)
			throw new Error('La répartition cible doit totaliser 100 %.');
		await session.api.request('PUT', '/v1/allocation/', {
			targets: Object.entries(v).map(([key, value]) => ({ class: key, target_bps: value }))
		});
		await load();
	}
</script>

<section class="glass panel">
	<h2>Allocation actuelle et cible</h2>
	{#if error}<p class="notice error" role="alert">{error}</p>{/if}{#if data}<DataView {data} />
		<details>
			<summary>Modifier la répartition cible</summary><Form
				fields={Object.entries(classes).map(([key, label]) => field(key, label, 'percent'))}
				{initial}
				submit={save}
			/>
		</details>{/if}
</section>
