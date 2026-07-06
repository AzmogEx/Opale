<script lang="ts">
	// Projection (EF-040/041/043) : curseurs d'hypothèses, courbe 30 ans,
	// date d'indépendance — tout est calculé par le moteur backend.
	import { session } from '$lib/session.svelte';
	import { euros, percent } from '$lib/format';
	import type { Projection } from '$lib/api';
	import LineChart from '$lib/components/LineChart.svelte';

	// Hypothèses persistées (euros entiers / bps), défauts alignés sur iOS.
	let savings = $state(Number(localStorage.getItem('proj.savings') ?? 500));
	let returnBps = $state(Number(localStorage.getItem('proj.return') ?? 500));
	let expenses = $state(Number(localStorage.getItem('proj.expenses') ?? 2000));
	let inflationBps = $state(0); // euros constants (EF-043)

	let projection = $state<Projection | null>(null);
	let error = $state('');

	$effect(() => {
		localStorage.setItem('proj.savings', String(savings));
		localStorage.setItem('proj.return', String(returnBps));
		localStorage.setItem('proj.expenses', String(expenses));
		const params = {
			savings: savings * 100,
			returnBps,
			expenses: expenses * 100,
			inflationBps
		};
		void session.api
			.projection(params)
			.then((p) => ((projection = p), (error = '')))
			.catch((e) => (error = e instanceof Error ? e.message : String(e)));
	});

	const freedomYear = $derived.by(() => {
		if (!projection?.independence.reached) return null;
		return new Date().getFullYear() + Math.floor(projection.independence.months / 12);
	});
</script>

<h1 class="mb-5 text-2xl font-bold tracking-tight">Projection</h1>

{#if error}
	<div class="glass border-loss/40 mb-4 p-4 text-sm">{error}</div>
{/if}

{#if projection}
	<div class="grid gap-5">
		<!-- Indépendance financière (EF-040) -->
		<section class="glass p-6 md:p-8">
			<p class="text-xs font-semibold text-neutral-500 uppercase">
				Indépendance financière {inflationBps > 0 ? '· euros constants' : ''}
			</p>
			{#if projection.independence.reached}
				<p class="iridescent mt-1 text-4xl font-extrabold md:text-5xl">
					{projection.independence.months === 0 ? 'Déjà libre 🎉' : `Libre en ${freedomYear}`}
				</p>
				{#if projection.independence.months > 0}
					<p class="mt-1 text-sm text-neutral-500">
						dans {Math.floor(projection.independence.months / 12)} ans et {projection
							.independence.months % 12} mois
					</p>
				{/if}
			{:else}
				<p class="mt-1 text-3xl font-bold text-neutral-400">Hors d'atteinte</p>
				<p class="mt-1 text-sm text-neutral-500">
					Augmente l'épargne ou réduis les dépenses pour rejoindre la cible.
				</p>
			{/if}
			<p class="mt-2 text-xs text-neutral-500">
				Cible : <span class="amount">{euros(projection.independence.target_cents)}</span> (retrait 4 %)
			</p>

			<div class="mt-6">
				<LineChart
					points={projection.points.map((p) => p.net_cents)}
					target={projection.independence.target_cents}
					labels={['aujourd’hui', '30 ans']}
					height={220}
				/>
			</div>
		</section>

		<!-- Hypothèses -->
		<section class="glass grid gap-6 p-6 md:grid-cols-2">
			<label class="block">
				<span class="flex justify-between text-sm font-medium">
					Épargne mensuelle <strong class="text-accent amount">{savings} €</strong>
				</span>
				<input type="range" min="0" max="5000" step="50" bind:value={savings} class="accent-accent w-full" />
			</label>
			<label class="block">
				<span class="flex justify-between text-sm font-medium">
					Rendement annuel <strong class="text-accent">{percent(returnBps)}</strong>
				</span>
				<input type="range" min="0" max="1200" step="50" bind:value={returnBps} class="accent-accent w-full" />
			</label>
			<label class="block">
				<span class="flex justify-between text-sm font-medium">
					Dépenses mensuelles <strong class="text-accent amount">{expenses} €</strong>
				</span>
				<input type="range" min="500" max="10000" step="100" bind:value={expenses} class="accent-accent w-full" />
			</label>
			<label class="block">
				<span class="flex justify-between text-sm font-medium">
					Inflation (euros constants) <strong class="text-accent">{percent(inflationBps)}</strong>
				</span>
				<input type="range" min="0" max="500" step="25" bind:value={inflationBps} class="accent-accent w-full" />
				<span class="text-xs text-neutral-400">
					{inflationBps > 0 ? 'La courbe est en pouvoir d’achat d’aujourd’hui.' : 'À 0, la courbe est en euros courants.'}
				</span>
			</label>
		</section>
	</div>
{:else if !error}
	<div class="glass animate-pulse p-8 text-sm text-neutral-500">Le moteur projette…</div>
{/if}
