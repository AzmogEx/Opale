<script lang="ts">
	import { goto } from '$app/navigation';
	import { session } from '$lib/session.svelte';
	import type { Profile } from '$lib/api';
	import { messageOf } from '$lib/domain';
	let profiles = $state<Profile[]>([]);
	let selected = $state<Profile | null>(session.profile);
	let pin = $state('');
	let name = $state('');
	let confirmation = $state('');
	let creating = $state(false);
	let error = $state('');
	let busy = $state(false);
	let loading = $state(true);
	async function load() {
		loading = true;
		try {
			profiles = await session.api.listProfiles();
			error = '';
		} catch (e) {
			error = messageOf(e);
		} finally {
			loading = false;
		}
	}
	$effect(() => {
		void load();
	});
	async function login(event: SubmitEvent) {
		event.preventDefault();
		busy = true;
		error = '';
		try {
			if (creating) {
				if (pin !== confirmation) throw new Error('Les codes ne correspondent pas.');
				selected = await session.api.createProfile(name, pin);
			}
			if (selected) {
				await session.login(selected.id, pin);
				await goto('/');
			}
		} catch (e) {
			error = messageOf(e);
			pin = '';
		} finally {
			busy = false;
		}
	}
	async function demo() {
		busy = true;
		error = '';
		try {
			const res = await session.api.request<{ profile: Profile; pin: string }>(
				'POST',
				'/v1/profiles/demo'
			);
			await session.login(res.profile.id, res.pin);
			await goto('/');
		} catch (e) {
			error = messageOf(e);
		} finally {
			busy = false;
		}
	}
</script>

<svelte:head><title>Connexion · Opale</title></svelte:head>
<div class="gate">
	<div class="glass panel gate-card">
		<h1 class="iridescent text-4xl font-extrabold">Opale</h1>
		<p class="muted">Ton patrimoine, tes projets. En toute confidentialité.</p>
		{#if session.reason}<p class="notice" role="status">{session.reason}</p>{/if}
		{#if error}<p class="notice error" role="alert">{error}</p>{/if}
		{#if selected || creating}<form onsubmit={login} class="opale-form">
				<h2>{creating ? 'Créer un profil' : `Déverrouiller ${selected?.name}`}</h2>
				{#if creating}<label
						>Prénom ou nom du profil<input
							bind:value={name}
							required
							maxlength="80"
							autocomplete="nickname"
						/></label
					>{/if}<label
					>Code personnel<input
						type="password"
						bind:value={pin}
						required
						minlength="4"
						maxlength="72"
						autocomplete={creating ? 'new-password' : 'current-password'}
					/></label
				>{#if creating}<label
						>Confirmer le code<input
							type="password"
							bind:value={confirmation}
							required
							minlength="4"
							autocomplete="new-password"
						/></label
					>
					<p class="muted">
						Le cloud est désactivé par défaut. Choisis un code difficile à deviner.
					</p>{:else}<p class="muted">
						Le code est demandé à chaque ouverture, après 5 minutes d’inactivité et au retour dans
						l’onglet. Le déverrouillage nécessite le serveur.
					</p>{/if}
				<div class="actions">
					<button class="primary" disabled={busy}
						>{busy ? 'Connexion…' : creating ? 'Créer mon profil' : 'Déverrouiller'}</button
					><button
						type="button"
						onclick={() => {
							selected = null;
							creating = false;
							pin = '';
						}}
						disabled={busy}>Changer de profil</button
					>
				</div>
			</form>
		{:else}{#if loading}<p role="status">Connexion au serveur…</p>{/if}
			<div class="profile-list">
				{#each profiles as profile (profile.id)}<button onclick={() => (selected = profile)}
						><span class="avatar">{profile.name.slice(0, 1).toUpperCase()}</span
						>{profile.name}</button
					>{/each}
			</div>
			{#if !loading && !profiles.length}<p class="empty">
					Bienvenue ! Crée ton premier profil pour commencer.
				</p>{/if}
			<div class="actions">
				<button class="primary" onclick={() => (creating = true)}>Créer un profil</button><button
					onclick={demo}
					disabled={busy}>Découvrir avec une démo</button
				><button onclick={load} disabled={loading}>Réessayer</button>
			</div>{/if}
	</div>
</div>
