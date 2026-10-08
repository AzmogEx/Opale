<script lang="ts">
	import { session } from '$lib/session.svelte';
	import { money, messageOf } from '$lib/domain';
	import DataView from './DataView.svelte';
	let data = $state<any>(null),
		emergency = $state(false),
		error = $state(''),
		celebration = $state('');
	async function load() {
		try {
			const [timeline, health] = await Promise.all([
				session.api.request<any>('GET', '/v1/timeline'),
				session.api.request<any>('GET', '/v1/health-score')
			]);
			data = timeline;
			emergency = health.emergency_fund_ready === true;
			const milestones = (timeline.events ?? []).filter((e: any) => e.kind === 'milestone');
			const latest = milestones.at(-1);
			const key = emergency
				? 'emergency-fund'
				: latest
					? `${latest.date}:${latest.amount_cents}`
					: '';
			if (key && !session.celebrated.includes(key)) {
				session.celebrated = [...session.celebrated, key];
				celebration = emergency
					? 'Ton fonds d’urgence couvre six mois de dépenses observées.'
					: `Un palier franchi : ${money(latest.amount_cents)} !`;
			}
		} catch (e) {
			error = messageOf(e);
		}
	}
	$effect(() => {
		void load();
	});
</script>

<section class="glass panel">
	<div class="section-heading">
		<h2>Jalons et histoire</h2>
		<button onclick={load}>Actualiser les jalons</button>
	</div>
	{#if error}<p class="notice error" role="alert">{error}</p>{/if}
	{#if celebration}<div class="milestone-celebration" role="status">
			<span class="milestone-star" aria-hidden="true">✦</span>
			<h3>{celebration}</h3>
			<p>Prends un instant pour apprécier le chemin parcouru.</p>
			<button onclick={() => (celebration = '')}>Continuer</button>
		</div>{/if}
	<p class="notice">
		Fonds d’urgence : {emergency
			? 'six mois de dépenses couverts.'
			: 'le seuil de six mois n’est pas confirmé par les dépenses observées.'}
	</p>
	{#if data}<DataView {data} />{:else if !error}<p role="status">Chargement des jalons…</p>{/if}
</section>
