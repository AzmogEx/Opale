<script lang="ts">
	import { session } from '$lib/session.svelte';
	import { messageOf } from '$lib/domain';
	import DataView from './DataView.svelte';
	let spaces = $state<any[]>([]);
	let profiles = $state<any[]>([]);
	let selected = $state('');
	let detail = $state<any>(null);
	let name = $state('');
	let member = $state('');
	let error = $state('');
	let busy = $state(false);
	let removing = $state('');
	const owner = $derived(spaces.find((s) => s.id === selected)?.created_by);
	const isOwner = $derived(owner === session.profile?.id);
	async function load() {
		try {
			const [s, p] = await Promise.all([
				session.api.request<any>('GET', '/v1/spaces/'),
				session.api.listProfiles()
			]);
			spaces = s.spaces ?? [];
			profiles = p;
			if (selected && !spaces.some((s) => s.id === selected)) {
				selected = '';
				detail = null;
			}
		} catch (e) {
			error = messageOf(e);
		}
	}
	$effect(() => {
		void load();
	});
	$effect(() => {
		const id = selected;
		detail = null;
		if (id)
			void session.api
				.request('GET', `/v1/spaces/${id}`)
				.then((d) => {
					if (selected === id) detail = d;
				})
				.catch((e) => (error = messageOf(e)));
	});
	async function create(e: SubmitEvent) {
		e.preventDefault();
		busy = true;
		try {
			const s = await session.api.request<any>('POST', '/v1/spaces/', { name });
			await load();
			selected = s.id;
			name = '';
		} catch (e) {
			error = messageOf(e);
		} finally {
			busy = false;
		}
	}
	async function members(method: string, id: string) {
		try {
			await session.api.request(
				method,
				`/v1/spaces/${selected}/members${method === 'DELETE' ? `/${id}` : ''}`,
				method === 'POST' ? { profile_id: id } : undefined
			);
			removing = '';
			detail = await session.api.request('GET', `/v1/spaces/${selected}`);
			await load();
		} catch (e) {
			error = messageOf(e);
		}
	}
</script>

<section class="glass panel">
	<h2>Espaces partagés</h2>
	<p class="muted">
		Seules les opérations explicitement attribuées à un espace sont partagées. Un membre n’accède
		pas à ton patrimoine privé. Les 50 dernières opérations communes sont présentées ; la balance
		couvre toutes les opérations partagées.
	</p>
	{#if error}<p class="notice error" role="alert">{error}</p>{/if}
	<form onsubmit={create} class="actions">
		<label>Nouvel espace<input bind:value={name} required /></label><button disabled={busy}
			>Créer</button
		>
	</form>
	<label
		>Espace à consulter<select bind:value={selected}
			><option value="">Choisir un espace</option>{#each spaces as s}<option value={s.id}
					>{s.name}</option
				>{/each}</select
		></label
	>{#if detail}<DataView data={detail} />
		{#if isOwner}<div class="actions">
				<label
					>Ajouter un profil<select bind:value={member}
						><option value="">Choisir</option>{#each profiles as p}<option value={p.id}
								>{p.name}</option
							>{/each}</select
					></label
				><button disabled={!member} onclick={() => members('POST', member)}>Ajouter</button>
			</div>{/if}
		{#each detail.members ?? [] as m}<div class="actions">
				<span>{m.name ?? m.profile_name ?? 'Membre'}</span
				>{#if m.profile_id !== owner && (isOwner || m.profile_id === session.profile?.id)}<button
						class="danger"
						onclick={() => (removing = m.profile_id ?? m.id)}>Retirer l’accès</button
					>{#if removing === (m.profile_id ?? m.id)}<button
							class="danger"
							onclick={() => members('DELETE', removing)}>Confirmer le retrait</button
						><button onclick={() => (removing = '')}>Annuler</button>{/if}{/if}
			</div>{/each}{/if}
</section>
