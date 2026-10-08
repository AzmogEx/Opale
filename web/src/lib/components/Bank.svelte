<script lang="ts">
	import { session } from '$lib/session.svelte';
	import { messageOf } from '$lib/domain';
	import DataView from './DataView.svelte';
	let status = $state<any>(null);
	let assets = $state<any[]>([]);
	let institutions = $state<any[]>([]);
	let institution = $state('');
	let assetID = $state('');
	let country = $state('FR');
	let error = $state('');
	let result = $state<any>(null);
	let busy = $state(false);
	let consent = $state('');
	let deleting = $state('');
	let accounts = $state<any[]>([]);
	let pending = $state<any[]>([]);
	let mappings = $state<Record<string, string>>({});
	async function load() {
		try {
			const [s, a] = await Promise.all([
				session.api.request<any>('GET', '/v1/bank/status'),
				session.api.listAssets()
			]);
			status = s;
			assets = a;
			if (s.configured) {
				const response = await session.api.request<any>('GET', '/v1/bank/accounts');
				accounts = response.accounts ?? [];
				pending = response.pending ?? [];
				mappings = Object.fromEntries(accounts.map((a) => [a.id, a.asset_id ?? '']));
			}
		} catch (e) {
			error = messageOf(e);
		}
	}
	$effect(() => {
		void load();
	});
	async function banks() {
		busy = true;
		try {
			const r = await session.api.request<any>(
				'GET',
				`/v1/bank/institutions?country=${encodeURIComponent(country)}`
			);
			institutions = r.institutions ?? [];
			error = '';
		} catch (e) {
			error = messageOf(e);
		} finally {
			busy = false;
		}
	}
	async function connect(e: SubmitEvent) {
		e.preventDefault();
		busy = true;
		try {
			const i = institutions.find((i) => i.id === institution);
			const r = await session.api.request<any>('POST', '/v1/bank/connect', {
				institution_id: institution,
				institution_name: i?.name ?? '',
				asset_id: assetID,
				redirect: `${window.location.origin}/patrimoine?section=bank`
			});
			const url = new URL(r.consent_link);
			if (url.protocol !== 'https:') throw new Error('Adresse de consentement invalide.');
			consent = url.href;
			await load();
			error = '';
		} catch (e) {
			error = messageOf(e);
		} finally {
			busy = false;
		}
	}
	async function sync() {
		busy = true;
		try {
			result = await session.api.request('POST', '/v1/bank/sync');
			await load();
		} catch (e) {
			error = messageOf(e);
		} finally {
			busy = false;
		}
	}
	async function mapAccount(id: string) {
		try {
			await session.api.request('PUT', `/v1/bank/accounts/${id}`, { asset_id: mappings[id] ?? '' });
			await load();
		} catch (e) {
			error = messageOf(e);
		}
	}
	async function renew(id: string) {
		try {
			const r = await session.api.request<any>('POST', `/v1/bank/links/${id}/renew`, {
				redirect: `${window.location.origin}/patrimoine?section=bank`
			});
			const url = new URL(r.consent_link);
			if (url.protocol !== 'https:') throw new Error('Adresse de consentement invalide.');
			consent = url.href;
		} catch (e) {
			error = messageOf(e);
		}
	}
	async function disconnect() {
		try {
			await session.api.request('DELETE', `/v1/bank/links/${deleting}`);
			deleting = '';
			await load();
		} catch (e) {
			error = messageOf(e);
		}
	}
</script>

<section class="glass panel">
	<h2>Connexions bancaires</h2>
	<p class="muted">
		Le consentement se donne sur le site de ta banque. Aucun identifiant bancaire n’est demandé ici.
	</p>
	{#if error}<p class="notice error" role="alert">
			{error}
		</p>{/if}{#if status && !status.configured}<p class="notice">
			La connexion GoCardless n’est pas configurée sur le serveur. La saisie et les imports restent
			disponibles.
		</p>{:else if status}<div class="actions">
			<label>Pays<input bind:value={country} maxlength="2" /></label><button
				onclick={banks}
				disabled={busy}>Rechercher les banques</button
			><button onclick={sync} disabled={busy}>Synchroniser maintenant</button>
		</div>
		<form onsubmit={connect} class="opale-form">
			<div class="form-grid">
				<label
					>Banque<select bind:value={institution} required
						><option value="">Choisir</option>{#each institutions as i}<option value={i.id}
								>{i.name}</option
							>{/each}</select
					></label
				><label
					>Compte local<select bind:value={assetID} required
						><option value="">Choisir</option
						>{#each assets.filter((a) => a.kind === 'checking' || a.kind === 'savings') as a}<option
								value={a.id}>{a.name}</option
							>{/each}</select
					></label
				>
			</div>
			<button class="primary" disabled={busy || !institution || !assetID}
				>Préparer la connexion</button
			>
		</form>
		{#if consent}<a class="notice" href={consent} target="_blank" rel="noopener noreferrer"
				>Continuer vers le consentement bancaire ↗</a
			><button onclick={sync}>J’ai terminé le consentement</button
			>{/if}{/if}{#each status?.links ?? [] as link}<article class="resource-row">
			<DataView data={link} />
			<div class="actions">
				<button onclick={() => renew(link.id)}>Renouveler le consentement</button><button
					class="danger"
					onclick={() => (deleting = link.id)}>Déconnecter</button
				>
			</div>
			{#if deleting === link.id}<div class="notice">
					<p>Révoquer cette connexion ? Les opérations déjà importées sont conservées.</p>
					<div class="actions">
						<button class="danger" onclick={disconnect}>Confirmer</button><button
							onclick={() => (deleting = '')}>Annuler</button
						>
					</div>
				</div>{/if}
		</article>{/each}
	<h3>Association des comptes bancaires</h3>
	<p class="muted">
		Après consentement, synchronise pour découvrir les comptes, associe chacun à son actif local,
		puis synchronise à nouveau. Aucun compte non associé ne doit être fusionné automatiquement.
	</p>
	{#each accounts as account}<article class="resource-row">
			<DataView data={account} /><label
				>Actif local correspondant<select bind:value={mappings[account.id]}
					><option value="">Non associé</option
					>{#each assets.filter((a) => !a.archived && ['checking', 'savings'].includes(a.kind) && a.currency === account.currency) as a}<option
							value={a.id}>{a.name} ({a.currency})</option
						>{/each}</select
				></label
			><button onclick={() => mapAccount(account.id)}>Enregistrer l’association</button>
		</article>{/each}
	{#if pending.length}<h3>Opérations provisoires</h3>
		<p class="notice">
			Ces opérations bancaires en attente ne sont pas comptabilisées dans les dépenses ni les
			soldes. Leur date peut encore être inconnue.
		</p>
		<DataView data={pending} />{/if}
	{#if result}<DataView data={result} />{/if}
</section>
