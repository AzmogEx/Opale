<script lang="ts">
	import { session } from '$lib/session.svelte';
	import { messageOf } from '$lib/domain';
	import { field, options } from '$lib/forms';
	import Form from './Form.svelte';
	import DataView from './DataView.svelte';
	let documents = $state<any[]>([]);
	let assets = $state<any[]>([]);
	let enabled = $state(false);
	let error = $state('');
	let files = $state<FileList | undefined>();
	let name = $state('');
	let kind = $state('other');
	let assetID = $state('');
	let editing = $state<any>(null);
	let deleting = $state('');
	let busy = $state(false);
	const kinds = options({
		deed: 'Acte',
		contract: 'Contrat',
		invoice: 'Facture',
		insurance: 'Assurance',
		identity: 'Identité',
		tax: 'Fiscalité',
		photo: 'Photo',
		other: 'Autre'
	});
	async function load() {
		try {
			const [r, a] = await Promise.all([
				session.api.request<any>('GET', '/v1/documents/'),
				session.api.listAssets()
			]);
			documents = r.documents ?? [];
			enabled = r.vault_configured ?? false;
			assets = a;
			error = '';
		} catch (e) {
			error = messageOf(e);
		}
	}
	$effect(() => {
		void load();
	});
	async function upload(e: SubmitEvent) {
		e.preventDefault();
		const file = files?.[0];
		if (!file) return;
		busy = true;
		try {
			if (file.size > 10 * 1024 * 1024) throw new Error('Le fichier dépasse 10 Mo.');
			const content = await new Promise<string>((resolve, reject) => {
				const reader = new FileReader();
				reader.onload = () => resolve(String(reader.result).split(',')[1]);
				reader.onerror = () => reject(new Error('Lecture impossible.'));
				reader.readAsDataURL(file);
			});
			await session.api.request('POST', '/v1/documents/', {
				name: name || file.name,
				kind,
				mime: file.type || 'application/octet-stream',
				asset_id: assetID,
				content_base64: content
			});
			files = undefined;
			name = '';
			await load();
		} catch (e) {
			error = messageOf(e);
		} finally {
			busy = false;
		}
	}
	async function update(v: Record<string, any>) {
		await session.api.request('PATCH', `/v1/documents/${editing.id}`, v);
		editing = null;
		await load();
	}
	async function remove() {
		try {
			await session.api.request('DELETE', `/v1/documents/${deleting}`);
			deleting = '';
			await load();
		} catch (e) {
			error = messageOf(e);
		}
	}
	async function download(doc: any) {
		try {
			await session.api.download(`/v1/documents/${doc.id}/content`, doc.name);
		} catch (e) {
			error = messageOf(e);
		}
	}
</script>

<section class="glass panel">
	<h2>Coffre-fort</h2>
	<p class="muted">
		Documents chiffrés sur le serveur. Les téléchargements sont des copies lisibles à protéger sur
		ton appareil.
	</p>
	{#if error}<p role="alert" class="notice error">{error}</p>{/if}{#if !enabled}<p class="notice">
			Le coffre doit être configuré sur le serveur avant d’ajouter un document.
		</p>{/if}
	<form onsubmit={upload} class="opale-form">
		<div class="form-grid">
			<label
				>Fichier (10 Mo maximum)<input
					type="file"
					bind:files
					required
					disabled={!enabled || busy}
				/></label
			><label>Nom<input bind:value={name} /></label><label
				>Type<select bind:value={kind}
					>{#each kinds as k}<option value={k.value}>{k.label}</option>{/each}</select
				></label
			><label
				>Actif associé<select bind:value={assetID}
					><option value="">Aucun</option>{#each assets as a}<option value={a.id}>{a.name}</option
						>{/each}</select
				></label
			>
		</div>
		<button class="primary" disabled={!files?.length || busy || !enabled}
			>{busy ? 'Envoi…' : 'Ajouter au coffre'}</button
		>
	</form>
	{#if editing}<Form
			fields={[
				field('name', 'Nom', 'text', { required: true }),
				field('kind', 'Type', 'select', { options: kinds }),
				field('asset_id', 'Actif associé', 'select', {
					options: assets.map((a) => ({ value: a.id, label: a.name }))
				})
			]}
			initial={editing}
			submit={update}
			cancel={() => (editing = null)}
		/>{/if}{#each documents as doc (doc.id)}<article class="resource-row">
			<DataView data={doc} />
			<div class="actions">
				<button onclick={() => download(doc)}>Télécharger</button><button
					onclick={() => (editing = doc)}>Modifier</button
				><button class="danger" onclick={() => (deleting = doc.id)}>Supprimer</button
				>{#if deleting === doc.id}<button class="danger" onclick={remove}>Confirmer</button><button
						onclick={() => (deleting = '')}>Annuler</button
					>{/if}
			</div>
		</article>{/each}{#if !documents.length}<p class="empty">Aucun document dans ce coffre.</p>{/if}
</section>
