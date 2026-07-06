<script lang="ts">
	import '../app.css';
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { session } from '$lib/session.svelte';

	let { children } = $props();

	// Porte d'authentification : tout sauf /login exige une session.
	$effect(() => {
		const onLogin = page.url.pathname === '/login';
		if (!session.loggedIn && !onLogin) goto('/login');
		if (session.loggedIn && onLogin) goto('/');
	});

	const nav = [
		{ href: '/', label: 'Accueil', icon: '◉' },
		{ href: '/flux', label: 'Flux', icon: '⇄' },
		{ href: '/patrimoine', label: 'Patrimoine', icon: '🏛' },
		{ href: '/projection', label: 'Projection', icon: '📈' },
		{ href: '/assistant', label: 'Assistant', icon: '✦' }
	];

	async function logout() {
		await session.logout();
		goto('/login');
	}
</script>

<svelte:head>
	<link
		rel="icon"
		href="data:image/svg+xml,<svg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 100 100'><text y='0.9em' font-size='90'>🪩</text></svg>"
	/>
</svelte:head>

{#if session.loggedIn}
	<div
		class="mx-auto flex min-h-screen max-w-6xl gap-6 p-4 md:p-6"
		class:discreet={session.discreet}
	>
		<!-- Barre latérale (desktop) / barre basse (mobile) -->
		<aside
			class="glass fixed inset-x-3 bottom-3 z-20 flex items-center justify-around p-2 md:static md:h-fit md:w-52 md:flex-col md:items-stretch md:justify-start md:gap-1 md:p-4"
		>
			<div
				class="iridescent mb-0 hidden px-3 pb-3 text-2xl font-extrabold tracking-tight md:block"
			>
				Opale
			</div>
			{#each nav as item (item.href)}
				<a
					href={item.href}
					class="flex items-center gap-3 rounded-2xl px-3 py-2 text-sm font-medium transition
						{page.url.pathname === item.href
						? 'bg-accent/15 text-accent'
						: 'text-neutral-600 hover:bg-black/5 dark:text-neutral-300 dark:hover:bg-white/10'}"
				>
					<span aria-hidden="true">{item.icon}</span>
					<span class="hidden md:inline">{item.label}</span>
				</a>
			{/each}
			<div class="md:mt-4 md:border-t md:border-black/5 md:pt-3 dark:md:border-white/10">
				<button
					onclick={() => (session.discreet = !session.discreet)}
					class="flex w-full items-center gap-3 rounded-2xl px-3 py-2 text-sm text-neutral-600 hover:bg-black/5 dark:text-neutral-300 dark:hover:bg-white/10"
					title="Mode discret : flouter les montants"
				>
					<span aria-hidden="true">{session.discreet ? '🙈' : '👁'}</span>
					<span class="hidden md:inline">Mode discret</span>
				</button>
				<button
					onclick={logout}
					class="hidden w-full items-center gap-3 rounded-2xl px-3 py-2 text-sm text-loss hover:bg-loss/10 md:flex"
				>
					<span aria-hidden="true">⎋</span> Se déconnecter
				</button>
			</div>
		</aside>

		<main class="min-w-0 flex-1 pb-24 md:pb-0">
			{@render children()}
		</main>
	</div>
{:else}
	{@render children()}
{/if}
