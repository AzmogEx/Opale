<script lang="ts">
	import { session } from '$lib/session.svelte';
	import { field } from '$lib/forms';
	import ResourcePanel from './ResourcePanel.svelte';
	let categories = $state<any[]>([]);
	let error = $state('');
	$effect(() => {
		void session.api
			.request<any>('GET', '/v1/categories')
			.then((r) => (categories = r.categories ?? []))
			.catch((e) => (error = e.message));
	});
</script>

{#if error}<p class="notice error" role="alert">{error}</p>{/if}<ResourcePanel
	title="Enveloppes mensuelles"
	path="/v1/envelopes/"
	listKey="envelopes"
	createMethod="PUT"
	updateMethod="PUT"
	fields={[
		field('category_id', 'Catégorie', 'select', {
			required: true,
			options: categories.map((c) => ({ value: c.id, label: c.name }))
		}),
		field('monthly_budget_cents', 'Budget mensuel (EUR)', 'money', { required: true })
	]}
/>
