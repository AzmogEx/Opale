<script lang="ts">
	import { session } from '$lib/session.svelte';
	import { field } from '$lib/forms';
	import { messageOf, currencyExponent } from '$lib/domain';
	import Form from './Form.svelte';
	import RemotePanel from './RemotePanel.svelte';
	let { kind }: { kind: 'property' | 'object' | 'company' | 'quote' } = $props();
	let assets = $state<any[]>([]);
	let liabilities = $state<any[]>([]);
	let selected = $state('');
	let initial = $state<any>({});
	let error = $state('');
	let refresh = $state(0);
	let loadingDetails = $state(false);
	const path = $derived(
		kind === 'property'
			? '/v1/real-estate'
			: kind === 'object'
				? '/v1/objects'
				: kind === 'company'
					? '/v1/company'
					: '/v1/investments'
	);
	const title = $derived(
		kind === 'property'
			? 'Immobilier'
			: kind === 'object'
				? 'Objets de valeur'
				: kind === 'company'
					? 'Entrepreneur'
					: 'Cours automatiques'
	);
	const rawFields = $derived(
		kind === 'property'
			? [
					field('purchase_price_cents', 'Prix d’achat', 'money'),
					field('purchase_date', 'Date d’achat', 'date'),
					field('monthly_rent_cents', 'Loyer mensuel', 'money'),
					field('monthly_charges_cents', 'Charges mensuelles', 'money'),
					field('property_tax_yearly_cents', 'Taxe foncière annuelle', 'money'),
					field('liability_id', 'Crédit associé', 'select', {
						options: liabilities.map((l) => ({ value: l.id, label: l.name }))
					}),
					field('monthly_loan_payment_cents', 'Mensualité du crédit', 'money')
				]
			: kind === 'object'
				? [
						field('category', 'Catégorie'),
						field('brand', 'Marque'),
						field('purchase_price_cents', 'Prix d’achat', 'money'),
						field('purchase_date', 'Date d’achat', 'date'),
						field('insured', 'Assuré', 'checkbox')
					]
				: kind === 'company'
					? [
							field('siren', 'SIREN'),
							field('ownership_bps', 'Part détenue (%)', 'percent', { default: 10000 }),
							field('cca_cents', 'Compte courant d’associé', 'money'),
							field('cca_asset_id', 'Créance CCA existante (facultatif)', 'select', {
								options: assets
									.filter((a) => a.kind === 'other' && a.id !== selected)
									.map((a) => ({ value: a.id, label: a.name })),
								omitEmpty: true,
								help: 'Sans rattachement, une créance distincte est créée automatiquement. La valorisation de la société représente uniquement ta part.'
							}),
							field('annual_dividends_cents', 'Dividendes annuels', 'money'),
							field('monthly_salary_cents', 'Salaire mensuel', 'money')
						]
					: [
							field('symbol', 'Identifiant CoinGecko (vide pour désactiver)'),
							field('quantity_micro', 'Quantité détenue', 'money', {
								exponent: 6,
								help: 'Jusqu’à 6 décimales. Fournisseur : CoinGecko.'
							})
						]
	);
	const currency = $derived(assets.find((a) => a.id === selected)?.currency ?? 'EUR');
	const fields = $derived(
		rawFields.map((f) =>
			f.type === 'money' && f.key !== 'quantity_micro'
				? { ...f, exponent: currencyExponent(currency), currency: () => currency }
				: f
		)
	);
	$effect(() => {
		void Promise.all([session.api.listAssets(), session.api.listLiabilities()])
			.then(([a, l]) => {
				assets = a;
				liabilities = l;
			})
			.catch((e) => (error = messageOf(e)));
	});
	$effect(() => {
		void refresh;
		const id = selected;
		const k = kind;
		initial = {};
		loadingDetails = Boolean(id);
		if (!id) return;
		error = '';
		let active = true;
		void session.api
			.request<any>('GET', k === 'quote' ? `/v1/assets/${id}/quote` : path)
			.then((r) => {
				if (!active) return;
				const list = r.properties ?? r.objects ?? r.companies ?? [];
				initial = k === 'quote' ? r : (list.find((x: any) => x.asset.id === id)?.details ?? {});
				loadingDetails = false;
			})
			.catch((e) => {
				if (active) error = messageOf(e);
			});
		return () => {
			active = false;
		};
	});
	async function save(v: Record<string, any>) {
		await session.api.request('PUT', `/v1/assets/${selected}/${kind}`, v);
		refresh++;
	}
	async function refreshQuotes() {
		try {
			await session.api.request('POST', '/v1/quotes/refresh');
			refresh++;
		} catch (e) {
			error = messageOf(e);
		}
	}
</script>

<section class="glass panel">
	<h2>Renseigner {title.toLowerCase()}</h2>
	{#if error}<p class="notice error" role="alert">{error}</p>{/if}<label
		>Actif concerné<select bind:value={selected}
			><option value="">Choisir un actif</option
			>{#each assets.filter((a) => !a.archived && (kind === 'property' ? a.kind === 'real_estate' : kind === 'company' ? a.kind === 'company_share' : kind === 'object' ? ['valuable', 'vehicle', 'precious_metal', 'other'].includes(a.kind) : true)) as a}<option
					value={a.id}>{a.name}</option
				>{/each}</select
		></label
	>{#if selected && loadingDetails}<p role="status">
			{error ? 'Informations indisponibles.' : 'Chargement des informations…'}
		</p>
		{#if error}<button onclick={() => refresh++}>Réessayer</button>{/if}
	{:else if selected}<Form {fields} {initial} submit={save} />{:else}<p class="muted">
			Crée d’abord l’actif du bon type dans l’onglet Actifs.
		</p>{/if}{#if kind === 'quote'}<p class="muted">
			Fournisseur : {initial.source ?? 'CoinGecko'} · Dernière valeur : {initial.as_of ?? 'aucune'}
			{initial.last_error ?? ''}
		</p>
		<button onclick={refreshQuotes}>Actualiser les cours configurés</button>{/if}
</section>
<RemotePanel {title} {path} {refresh} />
