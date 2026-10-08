<script lang="ts">
	import { session } from '$lib/session.svelte';
	import { dayString, messageOf, query, money, currencyExponent } from '$lib/domain';
	import { field, options } from '$lib/forms';
	import Form from './Form.svelte';
	import DataView from './DataView.svelte';
	import RemotePanel from './RemotePanel.svelte';
	let assets = $state<any[]>([]);
	let data = $state<any>(null);
	let recurring = $state<any[]>([]);
	let error = $state('');
	let editing = $state<any>(null);
	let deleting = $state('');
	let occurrence = $state<any>(null);
	let transactions = $state<any[]>([]);
	let from = $state(dayString(new Date()));
	let until = $state(
		dayString(new Date(new Date().getFullYear(), new Date().getMonth() + 3, new Date().getDate()))
	);
	let days = $state(90);
	let refresh = $state(0);
	function confirmDetected(r: any) {
		const detected = r.periodicity ?? r.frequency;
		editing = {
			label: r.label,
			merchant_key: r.merchant_key,
			date: String(r.next_date ?? r.next_on ?? from).slice(0, 10),
			frequency: ['weekly', 'monthly', 'quarterly', 'yearly'].includes(detected) ? detected : '',
			active: true,
			from_detection: true
		};
	}
	const fields = $derived([
		field('asset_id', 'Compte', 'select', {
			required: true,
			options: assets.map((a) => ({ value: a.id, label: `${a.name} (${a.currency})` }))
		}),
		field('label', 'Libellé', 'text', { required: true }),
		field('amount_cents', 'Montant signé', 'money', {
			required: true,
			exponent: (v) => currencyExponent(assets.find((a) => a.id === v.asset_id)?.currency ?? 'EUR'),
			currency: (v) => assets.find((a) => a.id === v.asset_id)?.currency ?? 'EUR',
			help: 'Négatif : sortie, positif : entrée.'
		}),
		field('date', 'Première échéance', 'date', { required: true }),
		field('frequency', 'Répétition', 'select', {
			required: true,
			default: 'once',
			options: options({
				once: 'Une seule fois',
				weekly: 'Chaque semaine',
				monthly: 'Chaque mois',
				quarterly: 'Chaque trimestre',
				yearly: 'Chaque année'
			})
		}),
		field('end_date', 'Fin de la série (facultatif)', 'date'),
		field('active', 'Prévision active', 'checkbox', { default: true }),
		field('merchant_key', 'Clé du marchand détecté (facultatif)', 'text', {
			help: 'Une détection confirmée est remplacée par cette série, sans double prévision.'
		})
	]);
	async function load() {
		try {
			const [c, r] = await Promise.all([
				session.api.request<any>('GET', `/v1/calendar${query({ from, until })}`),
				session.api.request<any>('GET', '/v1/recurring')
			]);
			data = c;
			recurring = r.recurring ?? [];
			error = '';
		} catch (e) {
			error = messageOf(e);
		}
	}
	$effect(() => {
		void session.api
			.listAssets()
			.then((a) => (assets = a))
			.catch((e) => (error = messageOf(e)));
		void load();
	});
	async function save(v: Record<string, any>) {
		await session.api.request(
			editing.id ? 'PATCH' : 'POST',
			editing.id ? `/v1/calendar/${editing.id}` : '/v1/calendar',
			v
		);
		editing = null;
		refresh++;
		await load();
	}
	async function remove() {
		try {
			await session.api.request('DELETE', `/v1/calendar/${deleting}`);
			deleting = '';
			refresh++;
			await load();
		} catch (e) {
			error = messageOf(e);
		}
	}
	async function exclude(key: string) {
		try {
			await session.api.request('PUT', '/v1/recurring/exclusion', {
				merchant_key: key,
				excluded: true
			});
			refresh++;
			await load();
		} catch (e) {
			error = messageOf(e);
		}
	}
	async function editOccurrence(o: any) {
		occurrence = o;
		try {
			const tx = await session.api.request<any>(
				'GET',
				`/v1/transactions/${query({ asset_id: o.asset_id, from: o.date, to: o.date, limit: 500 })}`
			);
			transactions = tx.transactions ?? [];
		} catch (e) {
			error = messageOf(e);
		}
	}
	async function saveOccurrence(v: Record<string, any>) {
		await session.api.request('PUT', `/v1/calendar/${occurrence.rule_id}/occurrences`, {
			...v,
			date: occurrence.date
		});
		occurrence = null;
		refresh++;
		await load();
	}
</script>

<div class="stack">
	<section class="glass panel">
		<div class="section-heading">
			<h2>Calendrier financier</h2>
			<button
				class="primary"
				onclick={() => (editing = { date: from, active: true, frequency: 'once' })}
				>Ajouter une échéance</button
			>
		</div>
		<p class="muted">
			Les prévisions sont distinctes des opérations réalisées. Modifier une série affecte ses
			occurrences futures ; modifier une occurrence laisse la série intacte.
		</p>
		<div class="form-grid">
			<label>Du<input type="date" bind:value={from} /></label><label
				>Au<input type="date" bind:value={until} /></label
			><button onclick={load}>Actualiser la période</button>
		</div>
		{#if error}<p class="notice error" role="alert">{error}</p>{/if}{#if editing?.from_detection}<p
				class="notice"
			>
				Le montant détecté est converti en EUR. Choisis le compte et saisis le montant dans sa
				devise. Si la périodicité détectée n’est pas proposée, choisis explicitement une périodicité
				adaptée.
			</p>{/if}{#if editing}<Form
				{fields}
				initial={editing}
				submit={save}
				cancel={() => (editing = null)}
			/>{/if}{#if occurrence}<Form
				fields={[
					field('status', 'État', 'select', {
						required: true,
						options: options({ planned: 'Prévu', excluded: 'Exclu', realized: 'Réalisé' })
					}),
					field('amount_cents', `Montant de cette occurrence (${occurrence.currency})`, 'money', {
						required: true,
						exponent: currencyExponent(occurrence.currency)
					}),
					field('transaction_id', 'Opération réalisée le même jour', 'select', {
						options: transactions.map((t) => ({
							value: t.id,
							label: `${t.label} · ${money(t.amount_cents, occurrence.currency)}`
						})),
						omitEmpty: true
					})
				]}
				initial={occurrence}
				submit={saveOccurrence}
				cancel={() => (occurrence = null)}
			/>{/if}
		<details open>
			<summary>Échéances de la période</summary>{#each data?.occurrences ?? [] as o}<div
					class="resource-row"
				>
					<DataView data={o} /><button onclick={() => editOccurrence(o)}
						>Modifier cette occurrence</button
					>
				</div>{/each}{#if data && !data.occurrences?.length}<p class="empty">
					Aucune échéance prévue sur cette période.
				</p>{/if}
		</details>
		<details>
			<summary>Gérer les séries</summary>{#each data?.rules ?? [] as rule}<div class="resource-row">
					<DataView data={rule} />
					<div class="actions">
						<button onclick={() => (editing = rule)}>Modifier / arrêter la série</button><button
							class="danger"
							onclick={() => (deleting = rule.id)}>Supprimer</button
						>{#if deleting === rule.id}<button class="danger" onclick={remove}>Confirmer</button
							><button onclick={() => (deleting = '')}>Annuler</button>{/if}
					</div>
				</div>{/each}
		</details>
		<details>
			<summary>Récurrences détectées à confirmer</summary>{#each recurring as r}<div
					class="resource-row"
				>
					<DataView data={r} />
					<div class="actions">
						<button onclick={() => confirmDetected(r)}>Confirmer et planifier</button><button
							onclick={() => exclude(r.merchant_key)}>Exclure cette détection</button
						>
					</div>
				</div>{/each}
		</details>
	</section>
	<section class="glass panel">
		<label
			>Horizon du cashflow (jours)<input type="number" min="1" max="730" bind:value={days} /></label
		>
	</section>
	<RemotePanel
		title="Trésorerie projetée"
		path={`/v1/cashflow?days=${days}`}
		{refresh}
		description="Une projection fondée sur les récurrences, échéances et habitudes. Elle reste une estimation."
	/>
</div>
