<script lang="ts">
	import { session } from '$lib/session.svelte';
	import { messageOf } from '$lib/domain';
	import DataView from './DataView.svelte';
	let {
		title,
		path,
		description = '',
		refresh = 0
	}: { title: string; path: string; description?: string; refresh?: number } = $props();
	let result = $state<any>(null);
	let error = $state('');
	let busy = $state(false);
	let stamp = $state('');
	let serial = 0;
	async function load() {
		const request = ++serial;
		busy = true;
		error = '';
		try {
			const data = await session.api.request('GET', path);
			if (request !== serial) return;
			result = data;
			stamp = new Date().toLocaleTimeString('fr-FR');
		} catch (e) {
			if (request === serial) error = messageOf(e);
		} finally {
			if (request === serial) busy = false;
		}
	}
	$effect(() => {
		void refresh;
		void path;
		void load();
	});
</script>

<section class="glass panel">
	<div class="section-heading">
		<h2>{title}</h2>
		<button onclick={load} disabled={busy} aria-label={`Actualiser ${title}`}>Actualiser</button>
	</div>
	{#if description}<p class="muted">{description}</p>{/if}{#if error}<p
			class="notice error"
			role="alert"
		>
			{error}
		</p>{/if}{#if busy}<p role="status" class="muted">Chargement…</p>{:else if result}<DataView
			data={result}
		/><small class="muted">Actualisé à {stamp}</small>{/if}
</section>
