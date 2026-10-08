<script lang="ts">
	import { session } from '$lib/session.svelte';
	import { messageOf } from '$lib/domain';
	import { field, options } from '$lib/forms';
	import Form from './Form.svelte';
	import RemotePanel from './RemotePanel.svelte';
	import Sharing from './Sharing.svelte';
	let error = $state('');
	let note = $state('');
	let confirmation = $state('');
	let action = $state('');
	async function exportData() {
		try {
			await session.api.download(
				'/v1/export',
				`opale-${new Date().toISOString().slice(0, 10)}.zip`
			);
			note = 'Export téléchargé. Il contient tes données lisibles, même en mode discret.';
		} catch (e) {
			error = messageOf(e);
		}
	}
	async function profileUpdate(value: Record<string, any>) {
		const p = await session.api.request<any>('PATCH', '/v1/me', value);
		session.profile = p;
		sessionStorage.setItem('opale.profile', JSON.stringify(p));
		note = 'Profil mis à jour.';
	}
	async function remove() {
		if (confirmation !== session.profile?.name) return;
		try {
			await session.api.request(
				'DELETE',
				`${action === 'profile' ? '/v1/me' : '/v1/me/data'}?confirm=${encodeURIComponent(confirmation)}`
			);
			if (action === 'profile') session.clear();
			else {
				note = 'Données réinitialisées.';
				session.generation++;
			}
			action = '';
			confirmation = '';
		} catch (e) {
			error = messageOf(e);
		}
	}
</script>

<div class="stack">
	<section class="glass panel">
		<h2>Préférences et confidentialité</h2>
		<div class="form-grid">
			<label
				>Apparence<select
					bind:value={session.theme}
					onchange={() => session.savePreference('theme', session.theme)}
					><option value="system">Système</option><option value="light">Clair</option><option
						value="dark">Sombre</option
					></select
				></label
			><label
				>Couleur d’accent<select
					bind:value={session.accent}
					onchange={() => session.savePreference('accent', session.accent)}
					><option value="opal">Opale</option><option value="violet">Améthyste</option><option
						value="blue">Saphir</option
					></select
				></label
			>
		</div>
		<p class="muted">
			Les préférences sont isolées par profil. Aucun solde ni document n’est conservé dans le
			stockage permanent du navigateur. L’accès hors ligne après verrouillage n’est pas disponible
			sur le web.
		</p>
		<Form
			fields={[
				field('name', 'Nom', 'text', { required: true }),
				field('privacy_default', 'Confidentialité', 'select', {
					required: true,
					options: options({
						N1: 'Local uniquement — aucun cloud',
						N2: 'Contexte minimisé possible avec accord par demande',
						N3: 'Cloud interdit'
					})
				}),
				field('pin', 'Nouveau code (laisser vide pour conserver)', 'password', { omitEmpty: true })
			]}
			initial={session.profile ?? {}}
			submit={profileUpdate}
		/>
	</section>
	<section class="glass panel">
		<h2>Mes données</h2>
		<p class="muted">
			L’export explicitement demandé contient les informations du profil et peut inclure des données
			sensibles.
		</p>
		<div class="actions">
			<button onclick={exportData}>Télécharger mon export</button><button
				class="danger"
				onclick={() => (action = 'data')}>Réinitialiser les données</button
			><button class="danger" onclick={() => (action = 'profile')}>Supprimer mon profil</button>
		</div>
		{#if action}<div class="notice">
				<p>
					{action === 'profile'
						? 'Le profil et toutes ses données seront supprimés.'
						: 'Les données financières de ce profil seront supprimées.'} Saisis « {session.profile
						?.name} » pour confirmer.
				</p>
				<label>Nom du profil<input bind:value={confirmation} /></label>
				<div class="actions">
					<button class="danger" disabled={confirmation !== session.profile?.name} onclick={remove}
						>Confirmer la suppression</button
					><button onclick={() => (action = '')}>Annuler</button>
				</div>
			</div>{/if}{#if error}<p role="alert" class="notice error">{error}</p>{/if}{#if note}<p
				role="status"
				class="notice"
			>
				{note}
			</p>{/if}
	</section>
	<Sharing /><RemotePanel title="Journal d’accès" path="/v1/access-log" />
</div>
