<script lang="ts">
	import { session } from '$lib/session.svelte';
	import { field, options } from '$lib/forms';
	import { dayString, messageOf } from '$lib/domain';
	import Form from './Form.svelte';
	import DataView from './DataView.svelte';
	import ResourcePanel from './ResourcePanel.svelte';
	import RemotePanel from './RemotePanel.svelte';
	let contacts = $state<any[]>([]);
	let documents = $state<any[]>([]);
	let assets = $state<any[]>([]);
	let profiles = $state<any[]>([]);
	let beneficiaries = $state<any[]>([]);
	let grants = $state<any[]>([]);
	let editing = $state<any>(null);
	let error = $state('');
	let beneficiaryDelete = $state('');
	let recipient = $state('');
	let selectedAssets = $state<string[]>([]);
	let selectedDocs = $state<string[]>([]);
	let expires = $state(dayString(new Date(Date.now() + 30 * 86400000)));
	let grantData = $state<any>(null);
	let busy = $state(false);
	async function load() {
		try {
			const [c, d, a, p, b, g] = await Promise.all([
				session.api.request<any>('GET', '/v1/contacts/'),
				session.api.request<any>('GET', '/v1/documents/'),
				session.api.listAssets(),
				session.api.listProfiles(),
				session.api.request<any>('GET', '/v1/beneficiaries'),
				session.api.request<any>('GET', '/v1/emergency-grants')
			]);
			contacts = c.contacts ?? [];
			documents = d.documents ?? [];
			assets = a;
			profiles = p;
			beneficiaries = b.beneficiaries ?? [];
			grants = g.grants ?? [];
			error = '';
		} catch (e) {
			error = messageOf(e);
		}
	}
	$effect(() => {
		void load();
	});
	async function save(v: Record<string, any>) {
		await session.api.request('PUT', '/v1/beneficiaries', v);
		editing = null;
		await load();
	}
	async function deleteBeneficiary() {
		try {
			await session.api.request('DELETE', `/v1/beneficiaries/${beneficiaryDelete}`);
			beneficiaryDelete = '';
			await load();
		} catch (e) {
			error = messageOf(e);
		}
	}
	async function create(e: SubmitEvent) {
		e.preventDefault();
		busy = true;
		try {
			await session.api.request('POST', '/v1/emergency-grants', {
				recipient_profile_id: recipient,
				asset_ids: selectedAssets,
				document_ids: selectedDocs,
				expires_at: new Date(expires + 'T23:59:59').toISOString()
			});
			selectedAssets = [];
			selectedDocs = [];
			recipient = '';
			await load();
		} catch (e) {
			error = messageOf(e);
		} finally {
			busy = false;
		}
	}
	async function toggle(g: any) {
		try {
			await session.api.request('PATCH', `/v1/emergency-grants/${g.id}`, { active: !g.active });
			grantData = null;
			await load();
		} catch (e) {
			error = messageOf(e);
		}
	}
	async function read(g: any) {
		grantData = null;
		try {
			grantData = await session.api.request('GET', `/v1/emergency/${g.id}`);
		} catch (e) {
			error = messageOf(e);
		}
	}
	async function download(id: string, name: string) {
		try {
			await session.api.download(`/v1/emergency/${grantData.grant.id}/documents/${id}`, name);
		} catch (e) {
			grantData = null;
			error = messageOf(e);
		}
	}
</script>

<ResourcePanel
	title="Contacts de confiance"
	path="/v1/contacts/"
	listKey="contacts"
	description="Un contact enregistré ne dispose d’aucun accès implicite."
	fields={[
		field('name', 'Nom', 'text', { required: true }),
		field('role', 'Rôle', 'select', {
			required: true,
			options: options({
				notary: 'Notaire',
				banker: 'Banquier',
				insurer: 'Assureur',
				accountant: 'Comptable',
				trusted: 'Proche de confiance',
				other: 'Autre'
			})
		}),
		field('phone', 'Téléphone'),
		field('email', 'E-mail', 'email'),
		field('note', 'Note', 'textarea')
	]}
/>
<section class="glass panel">
	<div class="section-heading">
		<h2>Bénéficiaires et contrats</h2>
		<button
			onclick={() => {
				void load();
				editing = {};
			}}>Lier un bénéficiaire</button
		>
	</div>
	<p class="muted">
		Ces liens servent à organiser tes documents. Ils ne remplacent pas les clauses et actes
		juridiquement applicables.
	</p>
	{#if error}<p class="notice error" role="alert">{error}</p>{/if}{#if editing}<Form
			fields={[
				field('contact_id', 'Bénéficiaire', 'select', {
					required: true,
					options: contacts.map((c) => ({ value: c.id, label: c.name }))
				}),
				field('document_id', 'Contrat du coffre', 'select', {
					required: true,
					options: documents.map((d) => ({ value: d.id, label: d.name }))
				}),
				field('share_bps', 'Part (%)', 'percent', { required: true, default: 10000 }),
				field('note', 'Note', 'textarea')
			]}
			initial={editing}
			submit={save}
			cancel={() => (editing = null)}
		/>{/if}{#each beneficiaries as b}<div class="resource-row">
			<DataView
				data={{
					beneficiaire: contacts.find((c) => c.id === b.contact_id)?.name,
					contrat: documents.find((d) => d.id === b.document_id)?.name,
					share_bps: b.share_bps,
					note: b.note
				}}
			/>
			<div class="actions">
				<button onclick={() => (editing = b)}>Modifier</button><button
					class="danger"
					onclick={() => (beneficiaryDelete = b.id)}>Supprimer</button
				>{#if beneficiaryDelete === b.id}<button class="danger" onclick={deleteBeneficiary}
						>Confirmer</button
					><button onclick={() => (beneficiaryDelete = '')}>Annuler</button>{/if}
			</div>
		</div>{/each}
</section>
<section class="glass panel">
	<h2>Accès d’urgence limité</h2>
	<p class="muted">
		Choisis précisément les actifs et documents accessibles à un autre profil. Le droit est créé
		inactif : son activation est une action distincte. Il expire à la date choisie et peut être
		révoqué à tout moment.
	</p>
	<form onsubmit={create} class="opale-form">
		<div class="form-grid">
			<label
				>Profil destinataire<select bind:value={recipient} required
					><option value="">Choisir</option
					>{#each profiles.filter((p) => p.id !== session.profile?.id) as p}<option value={p.id}
							>{p.name}</option
						>{/each}</select
				></label
			><label>Expiration (un an maximum)<input type="date" bind:value={expires} required /></label>
		</div>
		<fieldset>
			<legend>Actifs accessibles</legend>{#each assets as a}<label class="check-line"
					><input type="checkbox" bind:group={selectedAssets} value={a.id} />{a.name}</label
				>{/each}
		</fieldset>
		<fieldset>
			<legend>Documents accessibles</legend>{#each documents as d}<label class="check-line"
					><input type="checkbox" bind:group={selectedDocs} value={d.id} />{d.name}</label
				>{/each}
		</fieldset>
		<button
			class="primary"
			disabled={busy || !recipient || (!selectedAssets.length && !selectedDocs.length)}
			>Créer un droit inactif</button
		>
	</form>
	{#each grants as g}<article class="resource-row">
			<DataView
				data={{
					proprietaire: profiles.find((p) => p.id === g.owner_profile_id)?.name,
					destinataire: profiles.find((p) => p.id === g.recipient_profile_id)?.name,
					active: g.active,
					expires_at: g.expires_at,
					actifs: g.asset_ids?.length ?? 0,
					documents: g.document_ids?.length ?? 0
				}}
			/>{#if g.owner_profile_id === session.profile?.id}<button
					class:danger={g.active}
					onclick={() => toggle(g)}
					>{g.active ? 'Révoquer immédiatement' : 'Activer cet accès'}</button
				>{:else}<button onclick={() => read(g)}>Ouvrir les éléments autorisés</button>{/if}
		</article>{/each}{#if grantData}<div class="subtle">
			<DataView
				data={{ assets: grantData.assets, documents: grantData.documents }}
			/>{#each grantData.documents ?? [] as d}<button onclick={() => download(d.id, d.name)}
					>Télécharger {d.name}</button
				>{/each}<button onclick={() => (grantData = null)}>Fermer cet accès</button>
		</div>{/if}
</section>
<RemotePanel title="Plan de transmission" path="/v1/transmission" />
