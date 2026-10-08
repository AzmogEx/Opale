<script lang="ts">
	import { session } from '$lib/session.svelte';
	import type { Field } from '$lib/forms';
	import { query } from '$lib/domain';
	import Form from './Form.svelte';
	import DataView from './DataView.svelte';
	let {
		title,
		path,
		fields,
		description = '',
		method = 'POST',
		initial = {}
	}: {
		title: string;
		path: string;
		fields: Field[];
		description?: string;
		method?: string;
		initial?: Record<string, any>;
	} = $props();
	let result = $state<any>(null);
	async function submit(value: Record<string, any>) {
		result = await session.api.request(
			method,
			method === 'GET' ? path + query(value) : path,
			method === 'GET' ? undefined : value
		);
	}
</script>

<section class="glass panel">
	<h2>{title}</h2>
	{#if description}<p class="muted">{description}</p>{/if}<Form
		{fields}
		{initial}
		{submit}
		label="Calculer"
	/>{#if result}<div class="subtle" aria-live="polite">
			<h3>Résultat</h3>
			<DataView data={result} />
		</div>{/if}
</section>
