<script lang="ts">
	import { session } from '$lib/session.svelte';
	import { messageOf, parseAmount } from '$lib/domain';
	import DataView from './DataView.svelte';
	let data = $state<any>(null);
	let currency = $state('USD');
	let rate = $state('');
	let date = $state('');
	let error = $state('');
	let busy = $state(false);
	let removing = $state('');
	async function load() {
		try {
			data = await session.api.request('GET', '/v1/fx/');
			error = '';
		} catch (e) {
			error = messageOf(e);
		}
	}
	$effect(() => {
		void load();
	});
	async function save(e: SubmitEvent) {
		e.preventDefault();
		busy = true;
		try {
			const value = parseAmount(rate, 6);
			if (value <= 0) throw new Error('Le taux doit être positif.');
			await session.api.request('PUT', `/v1/fx/${currency.toUpperCase()}`, {
				rate_micro: value,
				...(date ? { as_of: date } : {})
			});
			await load();
		} catch (e) {
			error = messageOf(e);
		} finally {
			busy = false;
		}
	}
	async function remove() {
		try {
			await session.api.request('DELETE', `/v1/fx/${removing}`);
			removing = '';
			await load();
		} catch (e) {
			error = messageOf(e);
		}
	}
</script>

<section class="glass panel">
	<h2>Devises et taux</h2>
	<p class="muted">
		Saisis la valeur d’une unité de devise en euros. Un taux manquant doit être résolu pour obtenir
		un total fiable.
	</p>
	{#if error}<p class="notice error" role="alert">{error}</p>{/if}
	<form onsubmit={save} class="opale-form">
		<div class="form-grid">
			<label
				>Code devise<input
					bind:value={currency}
					pattern="[A-Za-z]{3}"
					maxlength="3"
					required
				/></label
			><label
				>1 unité vaut (EUR)<input
					bind:value={rate}
					inputmode="decimal"
					required
					placeholder="0,923456"
				/></label
			>
			<label>Date du taux (facultative)<input type="date" bind:value={date} /></label>
		</div>
		<button class="primary" disabled={busy}>Enregistrer le taux</button>
	</form>
	{#if data}<DataView {data} />{#each data.rates ?? [] as r}<div class="actions">
				<span>{r.currency}</span><button
					onclick={() => {
						currency = r.currency;
						rate = String(r.rate_micro / 1e6);
						date = r.as_of?.slice(0, 10) ?? '';
					}}>Modifier</button
				><button class="danger" onclick={() => (removing = r.currency)}>Supprimer</button>
			</div>{/each}{/if}{#if removing}<p>Supprimer le taux {removing} ?</p>
		<div class="actions">
			<button class="danger" onclick={remove}>Confirmer</button><button
				onclick={() => (removing = '')}>Annuler</button
			>
		</div>{/if}
</section>
