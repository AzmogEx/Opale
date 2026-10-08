<script lang="ts">
	import { session } from '$lib/session.svelte';
	import { field, options } from '$lib/forms';
	import { dayString, messageOf, currencyExponent } from '$lib/domain';
	import Form from './Form.svelte';
	import DataView from './DataView.svelte';
	import RemotePanel from './RemotePanel.svelte';
	import Allocation from './Allocation.svelte';
	let assets = $state<any[]>([]);
	let selected = $state('');
	let data = $state<any>(null);
	let editing = $state<any>(null);
	let error = $state('');
	let deleting = $state('');
	let coverage = $state(false);
	let refresh = $state(0);
	$effect(() => {
		void session.api
			.listAssets()
			.then((a) => (assets = a))
			.catch((e) => (error = messageOf(e)));
	});
	async function load() {
		if (!selected) return;
		try {
			data = await session.api.request<any>('GET', `/v1/assets/${selected}/investment`);
			coverage = data.coverage_complete;
			error = '';
		} catch (e) {
			error = messageOf(e);
		}
	}
	$effect(() => {
		void selected;
		data = null;
		editing = null;
		void load();
	});
	async function save(v: Record<string, any>) {
		await session.api.request(
			editing.id ? 'PATCH' : 'POST',
			`/v1/assets/${selected}/investment/flows${editing.id ? `/${editing.id}` : ''}`,
			v
		);
		editing = null;
		await load();
		refresh++;
	}
	async function remove() {
		try {
			await session.api.request('DELETE', `/v1/assets/${selected}/investment/flows/${deleting}`);
			deleting = '';
			await load();
			refresh++;
		} catch (e) {
			error = messageOf(e);
		}
	}
	async function confirmCoverage() {
		try {
			await session.api.request('PUT', `/v1/assets/${selected}/investment/coverage`, {
				complete: coverage
			});
			await load();
			refresh++;
		} catch (e) {
			error = messageOf(e);
		}
	}
</script>

<RemotePanel
	title="Placements et performance"
	path="/v1/investments"
	{refresh}
	description="Le moteur distingue versements et performance. Sans historique confirmé complet, le rendement reste indéterminé."
/>
<section class="glass panel">
	<h2>Flux d’investissement</h2>
	<label
		>Placement<select bind:value={selected}
			><option value="">Choisir</option
			>{#each assets.filter( (a) => ['pea', 'cto', 'life_insurance', 'crypto'].includes(a.kind) ) as a}<option
					value={a.id}>{a.name}{a.archived ? ' · archivé' : ''}</option
				>{/each}</select
		></label
	>{#if error}<p role="alert" class="notice error">{error}</p>{/if}{#if data}<DataView
			data={{
				currency: data.asset.currency,
				performance: data.performance,
				first_date: data.first_date,
				last_date: data.last_date
			}}
		/>
		<p class="muted">
			Distributions et frais enregistrés ici sont extérieurs à la valeur du portefeuille. Ne saisis
			pas à nouveau un frais déjà déduit dans les valorisations.
		</p>
		<button
			class="primary"
			onclick={() => (editing = { occurred_on: dayString(new Date()), kind: 'contribution' })}
			>Ajouter un flux</button
		>{#if editing}<Form
				fields={[
					field('kind', 'Nature', 'select', {
						required: true,
						options: options({
							contribution: 'Versement',
							withdrawal: 'Retrait',
							distribution: 'Distribution',
							fee: 'Frais externes'
						})
					}),
					field('amount_cents', `Montant positif (${data.asset.currency})`, 'money', {
						required: true,
						exponent: currencyExponent(data.asset.currency)
					}),
					field('occurred_on', 'Date', 'date', { required: true }),
					field('note', 'Note', 'textarea')
				]}
				initial={editing}
				submit={save}
				cancel={() => (editing = null)}
			/>{/if}{#each data.flows ?? [] as flow}<div class="resource-row">
				<DataView data={{ ...flow, currency: data.asset.currency }} />
				<div class="actions">
					<button onclick={() => (editing = flow)}>Modifier</button><button
						class="danger"
						onclick={() => (deleting = flow.id)}>Supprimer</button
					>{#if deleting === flow.id}<button class="danger" onclick={remove}>Confirmer</button
						><button onclick={() => (deleting = '')}>Annuler</button>{/if}
				</div>
			</div>{/each}<label
			><input type="checkbox" bind:checked={coverage} /> Je confirme avoir saisi tous les flux sur la
			période de valorisation.</label
		><button onclick={confirmCoverage}>Enregistrer la couverture historique</button>{/if}
</section>
<Allocation />
