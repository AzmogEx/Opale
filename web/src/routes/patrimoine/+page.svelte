<script lang="ts">
	// Patrimoine — actifs et dettes (EF-030→032), lecture seule côté web :
	// la saisie se fait dans l'app iOS, le web est le cockpit.
	import { session } from '$lib/session.svelte';
	import { kindLabels, type Asset, type Liability } from '$lib/api';
	import Amount from '$lib/components/Amount.svelte';

	let assets = $state<Asset[]>([]);
	let liabilities = $state<Liability[]>([]);
	let error = $state('');

	$effect(() => {
		void (async () => {
			try {
				[assets, liabilities] = await Promise.all([
					session.api.listAssets(),
					session.api.listLiabilities()
				]);
			} catch (e) {
				error = e instanceof Error ? e.message : String(e);
			}
		})();
	});

	const activeAssets = $derived(assets.filter((a) => !a.archived));
	const activeLiabilities = $derived(liabilities.filter((l) => !l.archived));
</script>

<h1 class="mb-5 text-2xl font-bold tracking-tight">Patrimoine</h1>

{#if error}
	<div class="glass border-loss/40 p-4 text-sm">{error}</div>
{:else}
	<div class="grid gap-5">
		<section class="glass p-5">
			<p class="text-xs font-semibold text-neutral-500 uppercase">Actifs</p>
			<ul class="mt-2 divide-y divide-black/5 dark:divide-white/10">
				{#each activeAssets as asset (asset.id)}
					<li class="flex items-center justify-between gap-4 py-3">
						<div>
							<p class="font-medium">{asset.name}</p>
							<p class="text-xs text-neutral-500">
								{kindLabels[asset.kind] ?? asset.kind}
								{#if asset.currency !== 'EUR'}&nbsp;· {asset.currency}{/if}
							</p>
						</div>
						<div class="text-right">
							{#if asset.latest_value_cents !== null}
								<p class="font-semibold"><Amount cents={asset.latest_value_cents} animate={false} /></p>
								{#if asset.theoretical_cents !== undefined && asset.theoretical_cents !== asset.latest_value_cents}
									<p class="text-xs text-amber-600" title="Dernière valorisation + mouvements postérieurs">
										théorique <Amount cents={asset.theoretical_cents} animate={false} />
									</p>
								{/if}
							{:else}
								<p class="text-xs text-neutral-400">À valoriser</p>
							{/if}
						</div>
					</li>
				{/each}
			</ul>
		</section>

		<section class="glass p-5">
			<p class="text-xs font-semibold text-neutral-500 uppercase">Dettes</p>
			{#if activeLiabilities.length === 0}
				<p class="mt-2 text-sm text-neutral-500">Aucune dette.</p>
			{/if}
			<ul class="mt-2 divide-y divide-black/5 dark:divide-white/10">
				{#each activeLiabilities as liability (liability.id)}
					<li class="flex items-center justify-between gap-4 py-3">
						<div>
							<p class="font-medium">{liability.name}</p>
							<p class="text-xs text-neutral-500">{kindLabels[liability.kind] ?? liability.kind}</p>
						</div>
						{#if liability.latest_value_cents !== null}
							<p class="text-loss font-semibold">
								<Amount cents={-liability.latest_value_cents} animate={false} />
							</p>
						{/if}
					</li>
				{/each}
			</ul>
		</section>

		<p class="text-center text-xs text-neutral-400">
			La saisie (actifs, valorisations, centres) se fait dans l'app iOS — le web est le cockpit.
		</p>
	</div>
{/if}
