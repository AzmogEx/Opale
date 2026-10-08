<script lang="ts">
	import '../app.css';
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { onMount } from 'svelte';
	import { session } from '$lib/session.svelte';
	import Settings from '$lib/components/Settings.svelte';
	let { children } = $props();
	let settings = $state(false);
	let offline = $state(false);
	let lastActivity = Date.now();
	$effect(() => {
		if (session.loggedIn) lastActivity = Date.now();
	});
	const nav = [
		{ href: '/', label: 'Accueil', icon: '◉' },
		{ href: '/flux', label: 'Flux', icon: '⇄' },
		{ href: '/patrimoine', label: 'Patrimoine', icon: '◇' },
		{ href: '/projection', label: 'Projection', icon: '↗' },
		{ href: '/assistant', label: 'Assistant', icon: '✦' }
	];
	$effect(() => {
		if (!session.loggedIn && page.url.pathname !== '/login') void goto('/login');
		if (session.loggedIn && page.url.pathname === '/login') void goto('/');
	});
	$effect(() => {
		document.documentElement.dataset.theme = session.theme;
		document.documentElement.dataset.accent = session.accent;
	});
	onMount(() => {
		const activity = () => {
			lastActivity = Date.now();
		};
		const visibility = () => {
			if (document.hidden && session.loggedIn) session.lock();
		};
		const network = () => (offline = !navigator.onLine);
		const expiration = setInterval(() => {
			if (session.loggedIn && Date.now() - lastActivity >= 300000) session.lock();
			if (session.token && session.expiresAt && Date.parse(session.expiresAt) <= Date.now())
				session.invalidate();
		}, 10000);
		window.addEventListener('pointerdown', activity);
		window.addEventListener('keydown', activity);
		document.addEventListener('visibilitychange', visibility);
		window.addEventListener('offline', network);
		window.addEventListener('online', network);
		network();
		activity();
		return () => {
			clearInterval(expiration);
			window.removeEventListener('pointerdown', activity);
			window.removeEventListener('keydown', activity);
			document.removeEventListener('visibilitychange', visibility);
			window.removeEventListener('offline', network);
			window.removeEventListener('online', network);
		};
	});
	async function logout() {
		try {
			await session.logout();
		} catch {
			session.clear();
		}
		settings = false;
	}
</script>

<svelte:head><meta name="theme-color" content="#59b8b5" /></svelte:head>
{#if session.loggedIn}<div class="app-shell">
		<a class="sr-link" href="#main-content">Aller au contenu</a>
		<aside class="sidebar glass">
			<a class="brand iridescent" href="/" onclick={() => (settings = false)}>Opale</a>
			<nav aria-label="Navigation principale">
				{#each nav as item}<a
						href={item.href}
						aria-label={item.label}
						aria-current={!settings && page.url.pathname === item.href ? 'page' : undefined}
						onclick={() => (settings = false)}
						><span aria-hidden="true">{item.icon}</span><span>{item.label}</span></a
					>{/each}
			</nav>
			<div class="sidebar-tools">
				<button
					onclick={() => {
						session.discreet = !session.discreet;
						session.savePreference('discreet', String(session.discreet));
					}}
					aria-pressed={session.discreet}
				>
					{session.discreet ? 'Afficher' : 'Masquer'} les données</button
				><button onclick={() => (settings = !settings)} aria-pressed={settings}>Paramètres</button
				><button onclick={() => session.lock()}>Verrouiller</button><button onclick={logout}
					>Déconnexion</button
				>
			</div>
		</aside>
		<div class="main-column">
			<header class="topbar">
				<span>{session.profile?.name}</span><span class="muted">Espace personnel</span>
			</header>
			{#if offline}<p class="notice" role="status">
					Hors ligne : les données déjà ouvertes restent consultables jusqu’au verrouillage. Les
					modifications nécessitent le serveur.
				</p>{/if}
			<main id="main-content">
				{#if session.discreet}<section class="glass panel">
						<h1>Données masquées</h1>
						<p>
							Les montants, graphiques, messages et formulaires sont retirés de l’affichage et des
							lecteurs d’écran.
						</p>
						<button
							onclick={() => {
								session.discreet = false;
								session.savePreference('discreet', 'false');
							}}>Afficher mes données</button
						>
					</section>{:else}{#key session.generation}{#if settings}<h1>Paramètres</h1>
							<Settings />{:else}{@render children()}{/if}{/key}{/if}
			</main>
		</div>
	</div>{:else if page.url.pathname === '/login'}{@render children()}{:else}<p
		class="gate"
		role="status"
	>
		Verrouillage…
	</p>{/if}
