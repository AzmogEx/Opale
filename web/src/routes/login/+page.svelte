<script lang="ts">
	// Porte de profils + PIN (EF-001/002) — miroir de ProfileGateView iOS.
	import { goto } from '$app/navigation';
	import { session } from '$lib/session.svelte';
	import type { Profile } from '$lib/api';

	let profiles = $state<Profile[]>([]);
	let selected = $state<Profile | null>(null);
	let pin = $state('');
	let error = $state('');
	let serverVisible = $state(false);
	let serverInput = $state(session.baseURL);
	let loading = $state(true);

	async function loadProfiles() {
		loading = true;
		error = '';
		try {
			profiles = await session.api.listProfiles();
		} catch (e) {
			error = `Serveur injoignable : ${e instanceof Error ? e.message : e}`;
		} finally {
			loading = false;
		}
	}

	$effect(() => {
		void loadProfiles();
	});

	async function login(event: SubmitEvent) {
		event.preventDefault();
		if (!selected) return;
		error = '';
		try {
			await session.login(selected.id, pin);
			goto('/');
		} catch (e) {
			error = e instanceof Error ? e.message : String(e);
			pin = '';
		}
	}

	function applyServer() {
		session.setBaseURL(serverInput);
		serverVisible = false;
		void loadProfiles();
	}
</script>

<div class="flex min-h-screen items-center justify-center p-6">
	<div class="glass w-full max-w-md p-8">
		<h1 class="iridescent text-4xl font-extrabold tracking-tight">Opale</h1>
		<p class="mt-1 text-sm text-neutral-500">Ton cockpit patrimonial — privé, chez toi.</p>

		{#if loading}
			<p class="mt-8 animate-pulse text-sm text-neutral-500">Connexion au serveur…</p>
		{:else if !selected}
			<h2 class="mt-8 text-sm font-semibold text-neutral-500 uppercase">Qui es-tu ?</h2>
			<div class="mt-3 grid gap-2">
				{#each profiles as profile (profile.id)}
					<button
						onclick={() => (selected = profile)}
						class="glass hover:border-accent/50 flex items-center gap-3 px-4 py-3 text-left font-medium transition"
					>
						<span
							class="bg-accent/15 text-accent flex h-9 w-9 items-center justify-center rounded-full font-bold"
						>
							{profile.name.slice(0, 1).toUpperCase()}
						</span>
						{profile.name}
					</button>
				{/each}
				{#if profiles.length === 0 && !error}
					<p class="text-sm text-neutral-500">
						Aucun profil — crée le premier depuis l'app iOS.
					</p>
				{/if}
			</div>
		{:else}
			<form onsubmit={login} class="mt-8">
				<h2 class="text-sm font-semibold text-neutral-500 uppercase">
					Code de {selected.name}
				</h2>
				<!-- svelte-ignore a11y_autofocus -->
				<input
					type="password"
					inputmode="numeric"
					autocomplete="current-password"
					placeholder="Code PIN"
					bind:value={pin}
					autofocus
					class="focus:border-accent mt-3 w-full rounded-2xl border border-black/10 bg-white/70 px-4 py-3 text-lg tracking-[0.4em] outline-none dark:border-white/10 dark:bg-white/10"
				/>
				<div class="mt-4 flex gap-2">
					<button
						type="button"
						onclick={() => ((selected = null), (pin = ''))}
						class="rounded-2xl px-4 py-3 text-sm text-neutral-500 hover:bg-black/5 dark:hover:bg-white/10"
					>
						Retour
					</button>
					<button
						type="submit"
						disabled={pin.length < 4}
						class="bg-accent flex-1 rounded-2xl px-4 py-3 font-semibold text-white shadow-lg transition hover:brightness-105 disabled:opacity-40"
					>
						Déverrouiller
					</button>
				</div>
			</form>
		{/if}

		{#if error}
			<p class="text-loss mt-4 text-sm">{error}</p>
		{/if}

		<div class="mt-8 border-t border-black/5 pt-4 dark:border-white/10">
			{#if serverVisible}
				<div class="flex gap-2">
					<input
						bind:value={serverInput}
						placeholder="https://opale.mondomaine.fr (vide = même origine)"
						class="flex-1 rounded-xl border border-black/10 bg-white/70 px-3 py-2 text-sm outline-none dark:border-white/10 dark:bg-white/10"
					/>
					<button
						onclick={applyServer}
						class="bg-accent/15 text-accent rounded-xl px-3 py-2 text-sm font-medium">OK</button
					>
				</div>
			{:else}
				<button
					onclick={() => (serverVisible = true)}
					class="text-xs text-neutral-400 hover:text-neutral-600"
				>
					Serveur : {session.baseURL || 'même origine'} — modifier
				</button>
			{/if}
		</div>
	</div>
</div>
