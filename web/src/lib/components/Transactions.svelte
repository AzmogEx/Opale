<script lang="ts">
	import { onDestroy } from 'svelte';
	import { session } from '$lib/session.svelte';
	import { field, options } from '$lib/forms';
	import {
		dayString,
		monthBounds,
		money,
		messageOf,
		parseAmount,
		currencyExponent,
		amountInput,
		query
	} from '$lib/domain';
	import Form from './Form.svelte';
	import DataView from './DataView.svelte';
	let date = $state(dayString(new Date()).slice(0, 7));
	let search = $state('');
	let category = $state('');
	let account = $state('');
	let offset = $state(0);
	const pageSize = 50;
	let rows = $state<any[]>([]);
	let categories = $state<any[]>([]);
	let assets = $state<any[]>([]);
	let liabilities = $state<any[]>([]);
	let spaces = $state<any[]>([]);
	let summary = $state<any>(null);
	let error = $state('');
	let busy = $state(false);
	let editing = $state<any>(null);
	let deleting = $state<any>(null);
	let splitting = $state<any>(null);
	let parts = $state<{ label: string; category_id: string; amount: string }[]>([]);
	let sharing = $state<any>(null);
	let labelTarget = $state<any>(null);
	let labelSuggestion = $state<any>(null);
	let labelBusy = $state(false);
	let labelError = $state('');
	let labelSerial = 0;
	onDestroy(() => {
		labelSerial++;
	});
	function openLabelSuggestion(tx: any) {
		labelSerial++;
		labelTarget = tx;
		labelSuggestion = null;
		labelError = '';
		labelBusy = false;
	}
	function closeLabelSuggestion() {
		labelSerial++;
		labelTarget = null;
		labelSuggestion = null;
		labelBusy = false;
	}
	async function suggestLabel() {
		const serial = ++labelSerial;
		const target = labelTarget;
		labelBusy = true;
		labelError = '';
		labelSuggestion = null;
		try {
			const suggestion = await session.api.request<any>(
				'POST',
				`/v1/transactions/${target.id}/label-suggestion`,
				{}
			);
			if (serial === labelSerial) labelSuggestion = suggestion;
		} catch (e) {
			if (serial === labelSerial) labelError = messageOf(e);
		} finally {
			if (serial === labelSerial) labelBusy = false;
		}
	}
	async function acceptLabel() {
		if (!labelSuggestion?.available || !labelSuggestion.suggested_label || labelBusy) return;
		const serial = ++labelSerial;
		const target = labelTarget;
		const label = labelSuggestion.suggested_label;
		labelBusy = true;
		labelError = '';
		try {
			await session.api.request('PATCH', `/v1/transactions/${target.id}/`, { label });
			if (serial === labelSerial) {
				closeLabelSuggestion();
				await load();
			}
		} catch (e) {
			if (serial === labelSerial) labelError = messageOf(e);
		} finally {
			if (serial === labelSerial) labelBusy = false;
		}
	}
	let spaceID = $state('');
	let importOpen = $state(false);
	let transferOpen = $state(false);
	let transferRequest = $state('');
	let file = $state<FileList | undefined>();
	let importAccount = $state('');
	let content = $state('');
	let importResult = $state<any>(null);
	let requestSerial = 0;
	const accountOptions = $derived(
		assets
			.filter((a) => !a.archived)
			.map((a) => ({ value: a.id, label: `${a.name} (${a.currency})` }))
	);
	const transferFields = $derived([
		field('from_asset_id', 'Compte débité', 'select', { required: true, options: accountOptions }),
		field('to_asset_id', 'Compte crédité', 'select', { required: true, options: accountOptions }),
		field('from_amount_cents', 'Montant débité hors frais', 'money', {
			required: true,
			exponent: (v) =>
				currencyExponent(assets.find((a) => a.id === v.from_asset_id)?.currency ?? 'EUR'),
			currency: (v) => assets.find((a) => a.id === v.from_asset_id)?.currency ?? 'EUR'
		}),
		field('to_amount_cents', 'Montant crédité', 'money', {
			required: true,
			exponent: (v) =>
				currencyExponent(assets.find((a) => a.id === v.to_asset_id)?.currency ?? 'EUR'),
			currency: (v) => assets.find((a) => a.id === v.to_asset_id)?.currency ?? 'EUR'
		}),
		field('fee_cents', 'Frais dans la devise débitée', 'money', {
			exponent: (v) =>
				currencyExponent(assets.find((a) => a.id === v.from_asset_id)?.currency ?? 'EUR'),
			currency: (v) => assets.find((a) => a.id === v.from_asset_id)?.currency ?? 'EUR'
		}),
		field('occurred_on', 'Date du virement', 'date', {
			required: true,
			default: dayString(new Date())
		})
	]);
	async function saveTransfer(v: Record<string, any>) {
		if (v.from_asset_id === v.to_asset_id) throw new Error('Choisis deux comptes différents.');
		if (v.from_amount_cents <= 0 || v.to_amount_cents <= 0 || v.fee_cents < 0)
			throw new Error('Les montants doivent être positifs et les frais nuls ou positifs.');
		await session.api.request('POST', '/v1/transfers', { ...v, request_id: transferRequest });
		transferOpen = false;
		await load();
	}
	const categoryOptions = $derived(categories.map((c) => ({ value: c.id, label: c.name })));
	const editCurrency = $derived(assets.find((a) => a.id === editing?.asset_id)?.currency ?? 'EUR');
	const fields = $derived([
		field('asset_id', 'Compte', 'select', { required: true, options: accountOptions }),
		field('label', 'Libellé', 'text', { required: true }),
		field('amount_cents', 'Montant signé', 'money', {
			required: true,
			exponent: (v) => currencyExponent(assets.find((a) => a.id === v.asset_id)?.currency ?? 'EUR'),
			currency: (v) => assets.find((a) => a.id === v.asset_id)?.currency ?? 'EUR',
			help: 'Dépense : négatif. Revenu : positif.'
		}),
		field('occurred_on', 'Date', 'date', { required: true }),
		field('category_id', 'Catégorie', 'select', { options: categoryOptions }),
		field('flow_kind', 'Nature du flux', 'select', {
			default: 'expense_income',
			required: true,
			options: options({
				expense_income: 'Revenu / dépense',
				internal_transfer: 'Virement interne',
				investment_contribution: 'Apport en investissement',
				investment_withdrawal: 'Retrait d’investissement',
				loan_principal: 'Remboursement de capital',
				interest: 'Intérêts',
				fee: 'Frais'
			})
		}),
		field('linked_liability_id', 'Dette remboursée (capital uniquement)', 'select', {
			options: liabilities
				.filter((l) => !l.archived)
				.map((l) => ({ value: l.id, label: `${l.name} (${l.currency})` })),
			omitEmpty: true,
			help: 'À choisir pour un remboursement de capital négatif, dans la même devise. Le restant dû est réduit automatiquement.'
		}),
		field('note', 'Note', 'textarea'),
		...(editing?.id
			? [
					field(
						'apply_to_similar',
						'Appliquer cette catégorie aux opérations du même marchand',
						'checkbox'
					)
				]
			: [])
	]);
	async function load() {
		const serial = ++requestSerial;
		busy = true;
		error = '';
		try {
			const [year, month] = date.split('-').map(Number);
			const bounds = monthBounds(year, month);
			const [tx, sum] = await Promise.all([
				session.api.request<any>(
					'GET',
					`/v1/transactions/${query({ ...bounds, q: search, category_id: category, asset_id: account, limit: pageSize + 1, offset })}`
				),
				session.api.monthSummary(year, month)
			]);
			if (serial === requestSerial) {
				rows = tx.transactions ?? [];
				summary = sum;
			}
		} catch (e) {
			if (serial === requestSerial) error = messageOf(e);
		} finally {
			if (serial === requestSerial) busy = false;
		}
	}
	$effect(() => {
		void session.api
			.listLiabilities()
			.then((l) => (liabilities = l))
			.catch((e) => (error = messageOf(e)));
		void session.api
			.listAssets()
			.then((a) => (assets = a))
			.catch((e) => (error = messageOf(e)));
		void session.api
			.request<any>('GET', '/v1/categories')
			.then((r) => (categories = r.categories ?? []))
			.catch((e) => (error = messageOf(e)));
		void session.api
			.request<any>('GET', '/v1/spaces/')
			.then((r) => (spaces = r.spaces ?? []))
			.catch((e) => (error = messageOf(e)));
	});
	$effect(() => {
		void date;
		void offset;
		void load();
	});
	function filters(e: SubmitEvent) {
		e.preventDefault();
		offset = 0;
		void load();
	}
	async function save(v: Record<string, any>) {
		await session.api.request(
			editing.id ? 'PATCH' : 'POST',
			editing.id ? `/v1/transactions/${editing.id}/` : '/v1/transactions/',
			v
		);
		editing = null;
		await load();
	}
	async function remove() {
		try {
			await session.api.request(
				'DELETE',
				deleting.transfer_id
					? `/v1/transfers/${deleting.transfer_id}`
					: `/v1/transactions/${deleting.id}/`
			);
			deleting = null;
			await load();
		} catch (e) {
			error = messageOf(e);
		}
	}
	function split(tx: any) {
		splitting = tx;
		parts = [
			{
				label: tx.label,
				category_id: tx.category_id ?? '',
				amount: amountInput(
					tx.amount_cents,
					currencyExponent(
						tx.currency ?? assets.find((a) => a.id === tx.asset_id)?.currency ?? 'EUR'
					)
				)
			},
			{ label: '', category_id: '', amount: '0' }
		];
	}
	async function saveSplit(e: SubmitEvent) {
		e.preventDefault();
		try {
			const exponent = currencyExponent(
				splitting.currency ?? assets.find((a) => a.id === splitting.asset_id)?.currency ?? 'EUR'
			);
			const parsed = parts.map((p) => ({
				label: p.label,
				category_id: p.category_id,
				amount_cents: parseAmount(p.amount, exponent)
			}));
			if (parsed.reduce((s, p) => s + p.amount_cents, 0) !== splitting.amount_cents)
				throw new Error('La somme des parts doit correspondre exactement au montant initial.');
			await session.api.request('POST', `/v1/transactions/${splitting.id}/split`, {
				parts: parsed
			});
			splitting = null;
			await load();
		} catch (e) {
			error = messageOf(e);
		}
	}
	async function share() {
		try {
			await session.api.request('PUT', `/v1/transactions/${sharing.id}/space`, {
				space_id: spaceID
			});
			sharing = null;
			await load();
		} catch (e) {
			error = messageOf(e);
		}
	}
	async function readFile() {
		error = '';
		content = '';
		const f = file?.[0];
		if (!f) return;
		try {
			if (f.size > 5 * 1024 * 1024) throw new Error('Le fichier dépasse la limite de 5 Mio.');
			const bytes = await f.arrayBuffer();
			try {
				content = new TextDecoder('utf-8', { fatal: true }).decode(bytes);
			} catch {
				content = new TextDecoder('windows-1252').decode(bytes);
			}
		} catch (e) {
			error = messageOf(e);
		}
	}
	async function importCSV(e: SubmitEvent) {
		e.preventDefault();
		busy = true;
		try {
			importResult = await session.api.request('POST', '/v1/transactions/import', {
				asset_id: importAccount,
				csv: content
			});
			content = '';
			file = undefined;
			await load();
		} catch (e) {
			error = messageOf(e);
		} finally {
			busy = false;
		}
	}
</script>

<section class="glass panel">
	<div class="section-heading">
		<h2>Opérations</h2>
		<div class="actions">
			<button
				class="primary"
				onclick={() =>
					(editing = {
						occurred_on: dayString(new Date()),
						asset_id: account || assets[0]?.id || ''
					})}>Ajouter</button
			><button
				onclick={() => {
					transferRequest = crypto.randomUUID();
					transferOpen = !transferOpen;
				}}>Virement entre comptes</button
			><button onclick={() => (importOpen = !importOpen)}>Importer CSV / OFX</button>
		</div>
	</div>
	<form onsubmit={filters} class="form-grid">
		<label
			>Mois<input type="month" bind:value={date} required onchange={() => (offset = 0)} /></label
		><label>Rechercher<input bind:value={search} placeholder="Libellé ou marchand" /></label><label
			>Compte<select bind:value={account}
				><option value="">Tous les comptes</option>{#each accountOptions as a}<option
						value={a.value}>{a.label}</option
					>{/each}</select
			></label
		><label
			>Catégorie<select bind:value={category}
				><option value="">Toutes les catégories</option>{#each categoryOptions as c}<option
						value={c.value}>{c.label}</option
					>{/each}</select
			></label
		><button disabled={busy}>Appliquer les filtres</button>
	</form>
	{#if error}<p class="notice error" role="alert">{error}</p>{/if}{#if summary}<details>
			<summary>Résumé complet du mois (tous comptes et catégories)</summary><DataView
				data={summary}
			/>
		</details>{/if}
	{#if editing}<Form
			{fields}
			initial={editing}
			submit={save}
			cancel={() => (editing = null)}
		/>{/if}
	{#if transferOpen}<div class="subtle">
			<h3>Virement entre tes comptes</h3>
			<p class="muted">
				Les deux côtés et les frais sont enregistrés ensemble. Les montants hors frais doivent être
				égaux si les devises sont identiques. Le virement ne devient pas un revenu ou une dépense.
			</p>
			<Form fields={transferFields} submit={saveTransfer} cancel={() => (transferOpen = false)} />
		</div>{/if}
	{#if importOpen}<div class="subtle">
			<h3>Importer des opérations</h3>
			<p class="muted">
				Choisis le compte auquel appartient le fichier. Les doublons sont détectés par le serveur.
				Le récapitulatif indique les lignes enregistrées et ignorées. Limites : 5 Mio et 10 000
				opérations par import.
			</p>
			<form onsubmit={importCSV} class="opale-form">
				<label
					>Compte de destination<select bind:value={importAccount} required
						><option value="">Choisir</option>{#each accountOptions as a}<option value={a.value}
								>{a.label}</option
							>{/each}</select
					></label
				><label
					>Fichier CSV ou OFX (5 Mio maximum)<input
						type="file"
						accept=".csv,.ofx,.qfx,text/csv"
						bind:files={file}
						onchange={readFile}
						required
					/></label
				>{#if content}<details open>
						<summary>Aperçu du fichier avant import</summary>
						<pre class="import-preview">{content.split(/\r?\n/).slice(0, 8).join('\n')}</pre>
					</details>{/if}<button class="primary" disabled={!content || !importAccount || busy}
					>Importer ce fichier</button
				>
			</form>
			{#if importResult}<DataView data={importResult} />{/if}
		</div>{/if}
	{#if splitting}<form onsubmit={saveSplit} class="subtle opale-form">
			<h3>
				Ventiler {splitting.label} · {money(splitting.amount_cents, splitting.currency ?? 'EUR')}
			</h3>
			{#each parts as part, i}<div class="form-grid">
					<label>Libellé de la part {i + 1}<input bind:value={part.label} required /></label><label
						>Montant signé<input bind:value={part.amount} inputmode="decimal" required /></label
					><label
						>Catégorie<select bind:value={part.category_id}
							><option value="">Aucune</option>{#each categoryOptions as c}<option value={c.value}
									>{c.label}</option
								>{/each}</select
						></label
					>{#if parts.length > 2}<button
							type="button"
							onclick={() => (parts = parts.filter((_, index) => index !== i))}
							>Retirer cette part</button
						>{/if}
				</div>{/each}
			<div class="actions">
				<button
					type="button"
					onclick={() => (parts = [...parts, { label: '', category_id: '', amount: '0' }])}
					>Ajouter une part</button
				><button class="primary">Enregistrer la ventilation</button><button
					type="button"
					onclick={() => (splitting = null)}>Annuler</button
				>
			</div>
		</form>{/if}
	{#if sharing}<div class="subtle">
			<h3>Partager « {sharing.label} »</h3>
			<label
				>Espace<select bind:value={spaceID}
					><option value="">Privé : retirer du partage</option>{#each spaces as s}<option
							value={s.id}>{s.name}</option
						>{/each}</select
				></label
			>
			<div class="actions">
				<button onclick={share}>Enregistrer le partage</button><button
					onclick={() => (sharing = null)}>Annuler</button
				>
			</div>
		</div>{/if}
	{#if labelTarget}<section class="subtle" aria-label="Suggestion de libellé">
			<h3>Nettoyer le libellé</h3>
			<p class="muted">
				Seul ce libellé sera envoyé à ton homelab privé, sans montant, note ni catégorie. Aucun
				appel au cloud. Rien n’est enregistré avant ton acceptation.
			</p>
			<dl>
				<dt>Libellé actuel</dt>
				<dd>{labelSuggestion?.original_label ?? labelTarget.label}</dd>
			</dl>
			{#if labelError}<p class="notice error" role="alert">{labelError}</p>{/if}
			{#if labelBusy}<p role="status">Traitement en cours…</p>{/if}
			{#if labelSuggestion}
				{#if labelSuggestion.available && labelSuggestion.suggested_label}
					<div aria-live="polite">
						<h4>Suggestion du homelab privé</h4>
						<p>{labelSuggestion.suggested_label}</p>
					</div>
					<div class="actions">
						<button class="primary" disabled={labelBusy} onclick={acceptLabel}
							>Accepter ce libellé</button
						><button disabled={labelBusy} onclick={closeLabelSuggestion}
							>Conserver le libellé actuel</button
						>
					</div>
				{:else}
					<p class="notice" role="status">
						{labelSuggestion.reason ||
							'La suggestion est indisponible. Ton libellé reste inchangé.'}
					</p>
					<button onclick={closeLabelSuggestion}>Fermer</button>
				{/if}
			{:else}
				<div class="actions">
					<button disabled={labelBusy} onclick={suggestLabel}>Suggérer via le homelab</button
					><button onclick={closeLabelSuggestion}>Annuler la suggestion</button>
				</div>
			{/if}
		</section>{/if}
	{#if deleting}<div class="notice" role="alert">
			<p>
				Supprimer « {deleting.label} » ? {deleting.transfer_id
					? 'Les deux côtés du virement et ses frais seront supprimés ensemble.'
					: ''}
			</p>
			<div class="actions">
				<button class="danger" onclick={remove}>Confirmer la suppression</button><button
					onclick={() => (deleting = null)}>Annuler</button
				>
			</div>
		</div>{/if}
	{#if busy}<p role="status" class="muted">Chargement…</p>{/if}
	<div class="table-scroll">
		<table>
			<thead><tr><th>Date / compte</th><th>Libellé</th><th>Montant</th><th>Actions</th></tr></thead
			><tbody
				>{#each rows.slice(0, pageSize) as tx (tx.id)}<tr
						><td
							>{String(tx.occurred_on).slice(0, 10)}<br /><small
								>{assets.find((a) => a.id === tx.asset_id)?.name ?? ''}</small
							></td
						><td
							>{tx.label}<br /><small>{tx.category_name ?? 'Sans catégorie'}</small
							>{#if tx.space_id}<small> · Partagée</small>{/if}</td
						><td class="amount"
							>{money(
								tx.amount_cents,
								tx.currency ?? assets.find((a) => a.id === tx.asset_id)?.currency ?? 'EUR'
							)}</td
						><td
							><details>
								<summary>Actions</summary>
								<div class="actions">
									{#if !tx.transfer_id && !tx.linked_liability_id}<button
											onclick={() => (editing = tx)}>Modifier</button
										><button onclick={() => split(tx)}>Ventiler</button>{:else}<span class="muted"
											>Écriture liée · pour corriger, supprimer et recréer</span
										>{/if}<button onclick={() => openLabelSuggestion(tx)}
										>Nettoyer le libellé</button
									><button
										onclick={() => {
											sharing = tx;
											spaceID = tx.space_id ?? '';
										}}>Partager</button
									><button class="danger" onclick={() => (deleting = tx)}>Supprimer</button>
								</div>
							</details></td
						></tr
					>{/each}</tbody
			>
		</table>
	</div>
	{#if !rows.length && !busy}<p class="empty">Aucune opération ne correspond à ces filtres.</p>{/if}
	<div class="section-heading actions">
		<button
			disabled={offset === 0 || busy}
			onclick={() => (offset = Math.max(0, offset - pageSize))}>Page précédente</button
		><span class="muted">Page {offset / pageSize + 1}</span><button
			disabled={rows.length <= pageSize || busy}
			onclick={() => (offset += pageSize)}>Page suivante</button
		>
	</div>
</section>
