<script lang="ts">
	import { session } from '$lib/session.svelte';
	import { money, messageOf } from '$lib/domain';
	import { monthLabel } from '$lib/format';
	import { field, options } from '$lib/forms';
	import Amount from '$lib/components/Amount.svelte';
	import LineChart from '$lib/components/LineChart.svelte';
	import HealthRing from '$lib/components/HealthRing.svelte';
	import DataView from '$lib/components/DataView.svelte';
	import ResourcePanel from '$lib/components/ResourcePanel.svelte';
	import Milestones from '$lib/components/Milestones.svelte';
	let section = $state('overview');
	let nw = $state<any>(null);
	let history = $state<any[]>([]);
	let cash = $state<number | null>(null);
	let health = $state<any>(null);
	let alerts = $state<any[]>([]);
	let risks = $state<any[]>([]);
	let errors = $state<string[]>([]);
	let loading = $state(true);
	let stamp = $state('');
	async function load() {
		loading = true;
		errors = [];
		const results = await Promise.allSettled([
			session.api.netWorth(),
			session.api.netWorthHistory(12),
			session.api.request<any>('GET', '/v1/cashflow?days=1'),
			session.api.healthScore(),
			session.api.alerts(),
			session.api.risks()
		]);
		const names = ['Patrimoine', 'Historique', 'Trésorerie', 'Santé', 'Alertes', 'Risques'];
		results.forEach((r, i) => {
			if (r.status === 'rejected') {
				errors.push(`${names[i]} : ${messageOf(r.reason)}`);
				return;
			}
			const v: any = r.value;
			if (i === 0) nw = v;
			if (i === 1) history = v.points ?? [];
			if (i === 2) cash = v.start_cash_cents;
			if (i === 3) health = v;
			if (i === 4) alerts = v;
			if (i === 5) risks = v;
		});
		stamp = new Date().toLocaleTimeString('fr-FR');
		loading = false;
	}
	$effect(() => {
		void load();
	});
	const delta = $derived(
		history.length > 1 ? history.at(-1).net_cents - history.at(-2).net_cents : null
	);
	const pct = $derived(
		delta !== null && history.at(-2)?.net_cents
			? (delta / Math.abs(history.at(-2).net_cents)) * 100
			: null
	);
</script>

<svelte:head><title>Accueil · Opale</title></svelte:head>
<div class="section-heading">
	<h1>Vue d’ensemble</h1>
	<button onclick={load} disabled={loading}>Actualiser</button>
</div>
<div class="tabstrip">
	<button aria-pressed={section === 'overview'} onclick={() => (section = 'overview')}
		>Mon patrimoine</button
	><button aria-pressed={section === 'alerts'} onclick={() => (section = 'alerts')}
		>Mes alertes</button
	><button aria-pressed={section === 'milestones'} onclick={() => (section = 'milestones')}
		>Jalons</button
	>
</div>
<div class="stack">
	{#if nw?.missing_valuations > 0}<p class="notice" role="status">
			Total incomplet : {nw.missing_valuations} actifs ou dettes sans valorisation.
		</p>{/if}
	{#if section === 'overview'}{#if errors.length}<div class="notice error" role="alert">
				{#each errors as error}<p>{error}</p>{/each}
				<p>Les autres rubriques restent disponibles.</p>
			</div>{/if}{#if loading && !nw}<p class="glass panel" role="status">
				Chargement du patrimoine…
			</p>{/if}
		{#each alerts as alert, i (i)}<article class="glass panel">
				<h2>{alert.title}</h2>
				<p class="muted">{alert.detail}</p>
			</article>{/each}
		{#if nw}<section class="glass panel hero">
				<p class="muted">PATRIMOINE NET</p>
				<p class="iridescent hero-amount"><Amount cents={nw.net_cents} /></p>
				{#if delta !== null}<p>
						{delta >= 0 ? '+' : ''}{money(delta)} ce mois-ci {pct !== null
							? `(${pct >= 0 ? '+' : ''}${pct.toLocaleString('fr-FR', { maximumFractionDigits: 1 })} %)`
							: ''}
					</p>{/if}{#if history.length > 1}<LineChart
						points={history.map((p) => p.net_cents)}
						pointLabels={history.map((p) => monthLabel(new Date(p.as_of)))}
						labels={[
							monthLabel(new Date(history[0].as_of)),
							monthLabel(new Date(history.at(-1).as_of))
						]}
					/>{:else}<p class="muted">
						Ajoute des valorisations datées pour voir ta trajectoire.
					</p>{/if}
			</section>
			<div class="stat-grid">
				<section class="glass panel">
					<p class="muted">Actifs</p>
					<strong>{money(nw.assets_total_cents)}</strong>
				</section>
				<section class="glass panel">
					<p class="muted">Dettes</p>
					<strong>{money(nw.liabilities_total_cents)}</strong>
				</section>
				<section class="glass panel">
					<p class="muted">Trésorerie courante</p>
					<strong>{cash !== null ? money(cash) : 'Indisponible'}</strong>
					<p class="muted">Soldes actualisés des comptes et livrets, convertis en euros.</p>
				</section>
			</div>{/if}
		{#if health}<section class="glass panel">
				<div class="section-heading">
					<h2>Santé financière</h2>
					<HealthRing score={health.score} />
				</div>
				<DataView data={health.components} />
			</section>{/if}{#if risks.length}<section class="glass panel">
				<h2>Radar de risques</h2>
				<DataView data={risks} />
			</section>{/if}<small class="muted"
			>Dernière actualisation : {stamp || 'en cours'}. Les données des rubriques en erreur peuvent
			être antérieures.</small
		>
	{:else if section === 'alerts'}<ResourcePanel
			title="Alertes personnalisées"
			path="/v1/alerts/custom/"
			listKey="alerts"
			fields={[
				field('kind', 'Déclencheur', 'select', {
					required: true,
					options: options({
						cash_below: 'Trésorerie sous le seuil',
						net_worth_below: 'Patrimoine sous le seuil',
						expenses_month_above: 'Dépenses mensuelles au-dessus'
					})
				}),
				field('threshold_cents', 'Seuil (EUR)', 'money', { required: true })
			]}
			updateFields={[
				field('threshold_cents', 'Seuil (EUR)', 'money', { required: true }),
				field('enabled', 'Alerte activée', 'checkbox')
			]}
		/>{:else}<Milestones />{/if}
</div>
