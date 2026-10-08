<script lang="ts">
	import { session } from '$lib/session.svelte';
	import { messageOf } from '$lib/domain';
	import type { Field } from '$lib/forms';
	import Form from './Form.svelte';
	import DataView from './DataView.svelte';
	let {
		title,
		path,
		listKey,
		fields,
		createMethod = 'POST',
		updateMethod = 'PATCH',
		allowEdit = true,
		description = '',
		updateFields,
		canModify = () => true,
		onLoaded
	}: {
		title: string;
		path: string;
		listKey: string;
		fields: Field[];
		createMethod?: string;
		updateMethod?: string;
		allowEdit?: boolean;
		description?: string;
		updateFields?: Field[];
		canModify?: (item: any) => boolean;
		onLoaded?: (items: any[]) => void;
	} = $props();
	let items = $state<any[]>([]);
	let loading = $state(false);
	let error = $state('');
	let editing = $state<any>(null);
	let deleting = $state<string | null>(null);
	let success = $state('');
	async function load() {
		loading = true;
		try {
			const data = await session.api.request<any>('GET', path);
			items = data[listKey] ?? [];
			onLoaded?.(items);
			error = '';
		} catch (e) {
			error = messageOf(e);
		} finally {
			loading = false;
		}
	}
	$effect(() => {
		void path;
		void load();
	});
	async function save(value: Record<string, any>) {
		await session.api.request(
			editing.id ? updateMethod : createMethod,
			editing.id && createMethod !== 'PUT' ? `${path.replace(/\/$/, '')}/${editing.id}` : path,
			value
		);
		editing = null;
		success = 'Enregistré.';
		await load();
	}
	async function remove(id: string) {
		try {
			await session.api.request('DELETE', `${path.replace(/\/$/, '')}/${id}`);
			deleting = null;
			success = 'Suppression effectuée.';
			await load();
		} catch (e) {
			error = messageOf(e);
		}
	}
</script>

<section class="glass panel">
	<div class="section-heading">
		<h2>{title}</h2>
		<button class="primary" onclick={() => (editing = {})}>Ajouter</button>
	</div>
	{#if description}<p class="muted">{description}</p>{/if}{#if error}<p
			class="notice error"
			role="alert"
		>
			{error}
		</p>{/if}{#if success}<p role="status" class="notice">{success}</p>{/if}
	{#if editing}<Form
			fields={editing.id ? (updateFields ?? fields) : fields}
			initial={editing}
			submit={save}
			cancel={() => (editing = null)}
		/>{/if}
	{#if loading}<p class="muted">Chargement…</p>{:else if !items.length}<p class="empty">
			Aucun élément. Ajoute le premier pour commencer.
		</p>{/if}
	{#each items as item (item.id)}<article class="resource-row">
			<DataView data={item} />
			<div class="actions">
				{#if allowEdit && canModify(item)}<button onclick={() => (editing = item)}>Modifier</button
					>{/if}{#if canModify(item)}<button class="danger" onclick={() => (deleting = item.id)}
						>Supprimer</button
					>{:else}<small class="muted">Catégorie du référentiel</small>{/if}
			</div>
			{#if deleting === item.id}<div class="notice" role="alert">
					<p>Supprimer cet élément ? Cette action est définitive.</p>
					<div class="actions">
						<button class="danger" onclick={() => remove(item.id)}>Confirmer la suppression</button
						><button onclick={() => (deleting = null)}>Annuler</button>
					</div>
				</div>{/if}
		</article>{/each}
</section>
