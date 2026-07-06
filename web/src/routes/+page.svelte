<script lang="ts">
	// Accueil — le cœur émotionnel (EF-010→015) : patrimoine net en héros,
	// trajectoire, cash, score de santé, alertes et radar de risques.
	import { session } from '$lib/session.svelte';
	import { euros, signedEuros, monthLabel } from '$lib/format';
	import type { NetWorth, NetWorthPoint, Asset, HealthScore, OpaleAlert, Risk } from '$lib/api';
	import Amount from '$lib/components/Amount.svelte';
	import LineChart from '$lib/components/LineChart.svelte';
	import HealthRing from '$lib/components/HealthRing.svelte';

	let netWorth = $state<NetWorth | null>(null);
	let history = $state<NetWorthPoint[]>([]);
	let cash = $state(0);
	let health = $state<HealthScore | null>(null);
	let alerts = $state<OpaleAlert[]>([]);
	let risks = $state<Risk[]>([]);
	let error = $state('');

	$effect(() => {
		void (async () => {
			try {
				const [nw, hist, assets, hs, al, rk] = await Promise.all([
					session.api.netWorth(),
					session.api.netWorthHistory(12),
					session.api.listAssets(),
					session.api.healthScore(),
					session.api.alerts(),
					session.api.risks()
				]);
				netWorth = nw;
				history = hist.points ?? [];
				cash = assets
					.filter((a: Asset) => ['checking', 'savings'].includes(a.kind) && !a.archived)
					.reduce((sum: number, a: Asset) => sum + (a.latest_value_cents ?? 0), 0);
				health = hs;
				alerts = al;
				risks = rk;
			} catch (e) {
				error = e instanceof Error ? e.message : String(e);
			}
		})();
	});

	const delta = $derived(
		history.length >= 2 ? history[history.length - 1].net_cents - history[history.length - 2].net_cents : null
	);
	const chartLabels = $derived(
		history.length >= 2
			? [monthLabel(new Date(history[0].as_of)), monthLabel(new Date(history[history.length - 1].as_of))]
			: []
	);
</script>

<h1 class="sr-only">Accueil</h1>

{#if error}
	<div class="glass border-loss/40 p-4 text-sm">Impossible de charger : {error}</div>
{:else if netWorth}
	<div class="grid gap-5">
		<!-- Alertes intelligentes (EF-053) -->
		{#each alerts as alert (alert.kind + alert.title)}
			<div
				class="glass flex items-start gap-3 p-4 {alert.severity === 'critical'
					? 'border-loss/50'
					: 'border-amber-400/50'}"
			>
				<span aria-hidden="true">{alert.severity === 'critical' ? '🚨' : '⚠️'}</span>
				<div>
					<p class="text-sm font-semibold">{alert.title}</p>
					<p class="text-xs text-neutral-500">{alert.detail}</p>
				</div>
			</div>
		{/each}

		<!-- Héros : le patrimoine net (EF-010) -->
		<section class="glass p-6 md:p-8">
			<p class="text-xs font-semibold tracking-wider text-neutral-500 uppercase">
				Patrimoine net
			</p>
			<p class="iridescent mt-1 text-5xl font-extrabold tracking-tight md:text-6xl">
				<Amount cents={netWorth.net_cents} />
			</p>
			{#if delta !== null}
				<p class="mt-2 text-sm font-medium {delta >= 0 ? 'text-gain' : 'text-loss'}">
					<span class="amount">{signedEuros(delta)}</span> ce mois-ci
				</p>
			{/if}

			{#if history.length >= 2}
				<div class="mt-6">
					<LineChart points={history.map((p) => p.net_cents)} labels={chartLabels} />
				</div>
			{/if}
		</section>

		<!-- Stats + santé -->
		<div class="grid gap-5 md:grid-cols-3">
			<section class="glass p-5">
				<p class="text-xs font-semibold text-neutral-500 uppercase">Actifs</p>
				<p class="mt-1 text-2xl font-bold"><Amount cents={netWorth.assets_total_cents} /></p>
				<p class="text-xs text-neutral-500">
					dont cash <span class="amount">{euros(cash)}</span>
				</p>
			</section>
			<section class="glass p-5">
				<p class="text-xs font-semibold text-neutral-500 uppercase">Dettes</p>
				<p class="text-loss mt-1 text-2xl font-bold">
					<Amount cents={-netWorth.liabilities_total_cents} />
				</p>
			</section>
			{#if health}
				<section class="glass flex items-center gap-4 p-5">
					<HealthRing score={health.score} />
					<div>
						<p class="text-xs font-semibold text-neutral-500 uppercase">Santé financière</p>
						<p class="text-sm text-neutral-500">
							{health.components.reduce(
								(worst, c) => (c.score / c.max < worst.score / worst.max ? c : worst),
								health.components[0]
							)?.comment}
						</p>
					</div>
				</section>
			{/if}
		</div>

		<!-- Radar de risques (EF-061) -->
		{#if risks.length > 0}
			<section class="glass p-5">
				<p class="text-xs font-semibold text-neutral-500 uppercase">Radar de risques</p>
				<ul class="mt-3 grid gap-3 md:grid-cols-2">
					{#each risks as risk (risk.id)}
						<li class="flex items-start gap-2 text-sm">
							<span aria-hidden="true">
								{risk.severity === 'critical' ? '🛑' : risk.severity === 'warning' ? '⚠️' : 'ℹ️'}
							</span>
							<div>
								<p class="font-semibold">{risk.title}</p>
								<p class="text-xs text-neutral-500">{risk.detail}</p>
							</div>
						</li>
					{/each}
				</ul>
			</section>
		{/if}
	</div>
{:else}
	<div class="glass animate-pulse p-8 text-sm text-neutral-500">Chargement du cockpit…</div>
{/if}
