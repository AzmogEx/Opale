<script lang="ts">
	import { session } from '$lib/session.svelte';
	import { field } from '$lib/forms';
	import ResourcePanel from './ResourcePanel.svelte';
	let categories = $state<any[]>([]);
</script>

<div class="stack">
	<ResourcePanel
		title="Catégories"
		path="/v1/categories"
		listKey="categories"
		onLoaded={(items) => (categories = items)}
		canModify={(c) => Boolean(c.profile_id)}
		description="Les catégories du référentiel restent disponibles. Crée tes catégories personnelles pour adapter le classement."
		fields={[field('name', 'Nom', 'text', { required: true }), field('icon', 'Icône (emoji)')]}
	/><ResourcePanel
		title="Règles de catégorisation"
		path="/v1/rules"
		listKey="rules"
		fields={[
			field('merchant_key', 'Marchand / clé de rapprochement', 'text', { required: true }),
			field('category_id', 'Catégorie', 'select', {
				required: true,
				options: categories.map((c) => ({ value: c.id, label: c.name }))
			})
		]}
	/>
</div>
