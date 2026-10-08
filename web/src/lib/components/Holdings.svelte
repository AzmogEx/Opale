<script lang="ts">
	import { session } from '$lib/session.svelte';
	import { kindLabels } from '$lib/api';
	import { field, options } from '$lib/forms';
	import { dayString, money, messageOf, currencyExponent } from '$lib/domain';
	import Form from './Form.svelte';
	import DataView from './DataView.svelte';
	let { debt = false }: { debt?: boolean } = $props();
	let items = $state<any[]>([]);
	let error = $state('');
	let busy = $state(false);
	let archived = $state(false);
	let editing = $state<any>(null);
	let selected = $state<any>(null);
	let valuations = $state<any[]>([]);
	let valuation = $state<any>(null);
	let deleting = $state<any>(null);
	let success = $state('');
	let missingValues = $state(0);
	const path = $derived(`/v1/${debt ? 'liabilities' : 'assets'}`);
	const kinds = $derived(
		Object.fromEntries(
			Object.entries(kindLabels).filter(([k]) =>
				debt
					? ['mortgage', 'auto_loan', 'consumer_loan', 'other'].includes(k)
					: !['mortgage', 'auto_loan', 'consumer_loan'].includes(k)
			)
		)
	);
	const fields = $derived([
		field('name', 'Nom', 'text', { required: true }),
		...(editing?.id
			? []
			: [
					field('kind', 'Type', 'select', { required: true, options: options(kinds) }),
					field('currency', 'Devise', 'select', {
						required: true,
						default: 'EUR',
						options: Intl.supportedValuesOf('currency').map((currency) => ({
							value: currency,
							label: `${new Intl.DisplayNames('fr', { type: 'currency' }).of(currency)} (${currency})`
						}))
					})
				]),
		...(editing?.id
			? []
			: [
					field('initial_value_cents', 'Valeur initiale (facultatif)', 'money', {
						omitEmpty: true,
						exponent: (v) => currencyExponent(v.currency || 'EUR'),
						currency: (v) => v.currency || 'EUR'
					}),
					field('initial_as_of', 'Date de la valeur initiale', 'date', {
						default: dayString(new Date()),
						omitEmpty: true
					})
				]),
		field('note', 'Note', 'textarea'),
		...(editing?.id ? [field('archived', 'Archivé / clôturé', 'checkbox')] : [])
	]);
	async function load() {
		busy = true;
		try {
			const r = await session.api.request<any>('GET', `${path}/`);
			items = r[debt ? 'liabilities' : 'assets'] ?? [];
			const total = await session.api.netWorth();
			missingValues = total.missing_valuations ?? 0;
			error = '';
		} catch (e) {
			error = messageOf(e);
		} finally {
			busy = false;
		}
	}
	$effect(() => {
		void path;
		selected = null;
		editing = null;
		void load();
	});
	async function save(value: Record<string, any>) {
		if (!editing.id) {
			editing.client_request_id ??= crypto.randomUUID();
			value.client_request_id = editing.client_request_id;
			if (value.initial_value_cents === undefined) delete value.initial_as_of;
		}
		const result = await session.api.request<any>(
			editing.id ? 'PATCH' : 'POST',
			editing.id ? `${path}/${editing.id}/` : `${path}/`,
			value
		);
		editing = null;
		success = 'Enregistré. Tu peux maintenant ajouter une valorisation.';
		await load();
		await select(items.find((i) => i.id === result.id) ?? result);
	}
	async function select(item: any) {
		selected = item;
		valuation = null;
		try {
			const r = await session.api.request<any>('GET', `${path}/${item.id}/valuations`);
			if (selected?.id === item.id) valuations = r.valuations ?? [];
		} catch (e) {
			error = messageOf(e);
		}
	}
	async function saveVal(value: Record<string, any>) {
		await session.api.request(
			valuation.id ? 'PATCH' : 'POST',
			valuation.id ? `/v1/valuations/${valuation.id}` : `${path}/${selected.id}/valuations`,
			value
		);
		valuation = null;
		await load();
		await select(items.find((i) => i.id === selected.id) ?? selected);
	}
	async function remove() {
		try {
			await session.api.request(
				'DELETE',
				deleting.kind === 'valuation'
					? `/v1/valuations/${deleting.item.id}`
					: `${path}/${deleting.item.id}/`
			);
			deleting = null;
			await load();
			if (selected && items.some((i) => i.id === selected.id))
				await select(items.find((i) => i.id === selected.id));
			else selected = null;
		} catch (e) {
			error = messageOf(e);
		}
	}
</script>

<section class="glass panel">
	{#if missingValues > 0}<p class="notice" role="status">
			Total incomplet : {missingValues} actifs ou dettes sans valorisation.
		</p>{/if}
	<div class="section-heading">
		<h2>{debt ? 'Dettes' : 'Actifs'}</h2>
		<button class="primary" onclick={() => (editing = {})}
			>Ajouter {debt ? 'une dette' : 'un actif'}</button
		>
	</div>
	<label><input type="checkbox" bind:checked={archived} /> Inclure les éléments archivés</label
	>{#if error}<p class="notice error" role="alert">{error}</p>{/if}{#if success}<p
			class="notice"
			role="status"
		>
			{success}
		</p>{/if}{#if editing}<Form
			{fields}
			initial={editing}
			submit={save}
			cancel={() => (editing = null)}
		/>{/if}
	{#if busy}<p role="status">Chargement…</p>{:else if !items.length}<p class="empty">
			{debt ? 'Aucune dette.' : 'Ajoute ton premier actif, puis sa valeur actuelle.'}
		</p>{/if}
	<div class="table-scroll">
		<table>
			<thead><tr><th>Nom</th><th>Type</th><th>Solde / valeur</th><th>Actions</th></tr></thead><tbody
				>{#each items.filter((i) => archived || !i.archived) as item (item.id)}<tr
						><td
							><button onclick={() => select(item)}
								>{item.name}{item.archived ? ' · archivé' : ''}</button
							></td
						><td>{kindLabels[item.kind] ?? item.kind}</td><td class="amount"
							>{item.current_value_cents != null
								? money(item.current_value_cents, item.currency)
								: item.theoretical_cents != null
									? money(item.theoretical_cents, item.currency)
									: item.latest_value_cents != null
										? money(item.latest_value_cents, item.currency)
										: 'À valoriser'}</td
						><td><button onclick={() => (editing = item)}>Modifier</button></td></tr
					>{/each}</tbody
			>
		</table>
	</div>
</section>
{#if selected}<section class="glass panel">
		<div class="section-heading">
			<h2>{selected.name} · historique</h2>
			<button onclick={() => (selected = null)}>Fermer</button>
		</div>
		<p class="muted">
			Une valorisation représente le solde à la fin de sa date de référence. Les mouvements suivants
			ajustent le solde courant. Archiver conserve l’historique.
		</p>
		<div class="actions">
			<button class="primary" onclick={() => (valuation = { as_of: dayString(new Date()) })}
				>Ajouter une valorisation</button
			><button class="danger" onclick={() => (deleting = { kind: 'holding', item: selected })}
				>Supprimer {debt ? 'cette dette' : 'cet actif'}</button
			>
		</div>
		{#if valuation}<Form
				fields={[
					field('value_cents', `Valeur (${selected.currency})`, 'money', {
						required: true,
						exponent: selected.currency_exponent ?? currencyExponent(selected.currency)
					}),
					field('as_of', 'Date de référence', 'date', { required: true }),
					field('note', 'Note', 'textarea')
				]}
				initial={valuation}
				submit={saveVal}
				cancel={() => (valuation = null)}
			/>{/if}
		{#each valuations as val (val.id)}<div class="resource-row">
				<DataView data={{ ...val, currency: selected.currency }} />
				<div class="actions">
					<button onclick={() => (valuation = val)}>Corriger</button><button
						class="danger"
						onclick={() => (deleting = { kind: 'valuation', item: val })}>Supprimer</button
					>
				</div>
			</div>{/each}{#if !valuations.length}<p class="empty">
				Aucune valorisation enregistrée.
			</p>{/if}
	</section>{/if}
{#if deleting}<section class="glass panel notice" role="alert">
		<h2>Confirmer la suppression</h2>
		<p>
			Cette suppression est définitive et peut modifier les totaux. Pour conserver l’historique d’un
			actif, privilégie son archivage.
		</p>
		<div class="actions">
			<button class="danger" onclick={remove}>Supprimer définitivement</button><button
				onclick={() => (deleting = null)}>Annuler</button
			>
		</div>
	</section>{/if}
