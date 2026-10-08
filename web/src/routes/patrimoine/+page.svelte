<script lang="ts">
	import { page } from '$app/state';
	import Holdings from '$lib/components/Holdings.svelte';
	import AssetDetails from '$lib/components/AssetDetails.svelte';
	import Vault from '$lib/components/Vault.svelte';
	import RemotePanel from '$lib/components/RemotePanel.svelte';
	import ResourcePanel from '$lib/components/ResourcePanel.svelte';
	import Currencies from '$lib/components/Currencies.svelte';
	import Bank from '$lib/components/Bank.svelte';
	import Investments from '$lib/components/Investments.svelte';
	import Transmission from '$lib/components/Transmission.svelte';
	import { field, options } from '$lib/forms';
	let section = $state(page.url.searchParams.get('section') ?? 'assets');
	const sections = {
		assets: 'Actifs',
		liabilities: 'Dettes',
		property: 'Immobilier',
		investments: 'Investissements',
		object: 'Objets',
		company: 'Entrepreneur',
		vault: 'Coffre-fort',
		transmission: 'Transmission',
		currencies: 'Devises',
		bank: 'Banques',
		quote: 'Cours'
	};
</script>

<svelte:head><title>Patrimoine · Opale</title></svelte:head>
<h1>Patrimoine</h1>
<div class="tabstrip" aria-label="Sections patrimoine">
	{#each Object.entries(sections) as [key, label]}<button
			aria-pressed={section === key}
			onclick={() => (section = key)}>{label}</button
		>{/each}
</div>
<div class="stack">
	{#key section}
		{#if section === 'assets'}<Holdings />{:else if section === 'liabilities'}<Holdings
				debt
			/>{:else if section === 'property' || section === 'object' || section === 'company' || section === 'quote'}<AssetDetails
				kind={section}
			/>{:else if section === 'investments'}<Investments />{:else if section === 'vault'}<Vault
			/>{:else if section === 'transmission'}<Transmission
			/>{:else if section === 'currencies'}<Currencies />{:else if section === 'bank'}<Bank
			/>{/if}{/key}
</div>
