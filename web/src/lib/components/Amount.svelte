<script lang="ts">
	// Compteur animé (l'équivalent web d'AmountText) : le montant « défile »
	// vers sa nouvelle valeur, et se floute en mode discret via .amount.
	import { Tween } from 'svelte/motion';
	import { cubicOut } from 'svelte/easing';
	import { euros, signedEuros } from '$lib/format';

	let {
		cents,
		signed = false,
		animate = true
	}: { cents: number; signed?: boolean; animate?: boolean } = $props();

	const tween = new Tween(0, { duration: 700, easing: cubicOut });
	$effect(() => {
		if (animate) tween.target = cents;
		else tween.set(cents, { duration: 0 });
	});

	const text = $derived(signed ? signedEuros(tween.current) : euros(tween.current));
</script>

<span class="amount">{text}</span>
