<script lang="ts">
	import Transactions from '$lib/components/Transactions.svelte';
	import Budgets from '$lib/components/Budgets.svelte';
	import Calendar from '$lib/components/Calendar.svelte';
	import Categories from '$lib/components/Categories.svelte';
	import RemotePanel from '$lib/components/RemotePanel.svelte';
	let section = $state('transactions');
	let year = $state(new Date().getFullYear());
	let month = $state(new Date().getMonth() + 1);
	const sections = {
		transactions: 'Opérations',
		budgets: 'Enveloppes',
		calendar: 'Calendrier',
		subscriptions: 'Abonnements',
		analytics: 'Analyse',
		categories: 'Catégories et règles',
		wrapped: 'Bilan annuel'
	};
</script>

<svelte:head><title>Flux · Opale</title></svelte:head>
<h1>Flux</h1>
<div class="tabstrip" aria-label="Sections flux">
	{#each Object.entries(sections) as [key, label]}<button
			aria-pressed={section === key}
			onclick={() => (section = key)}>{label}</button
		>{/each}
</div>
<div class="stack">
	{#key section}{#if section === 'transactions'}<Transactions
			/>{:else if section === 'budgets'}<Budgets />{:else if section === 'calendar'}<Calendar
			/>{:else if section === 'categories'}<Categories
			/>{:else if section === 'subscriptions'}<RemotePanel
				title="Abonnements détectés"
				path="/v1/subscriptions"
				description="Ces détections sont des estimations. Confirme ou exclus leur récurrence dans le calendrier."
			/>{:else if section === 'analytics'}<section class="glass panel form-grid">
				<label>Année<input type="number" min="2000" max="2100" bind:value={year} /></label><label
					>Mois<input type="number" min="1" max="12" bind:value={month} /></label
				>
			</section>
			<RemotePanel
				title="Analyse des dépenses"
				path={`/v1/analytics?year=${year}&month=${month}`}
			/>{:else if section === 'wrapped'}<section class="glass panel">
				<label
					>Année du bilan<input
						type="number"
						min="2000"
						max={new Date().getFullYear()}
						bind:value={year}
					/></label
				>
			</section>
			<RemotePanel title={`Ton année ${year}`} path={`/v1/wrapped?year=${year}`} />{/if}{/key}
</div>
