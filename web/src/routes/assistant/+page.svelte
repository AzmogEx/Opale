<script lang="ts">
	import { onDestroy, tick } from 'svelte';
	import { session } from '$lib/session.svelte';
	import { messageOf, money } from '$lib/domain';
	import type { AssistantStatus } from '$lib/api';
	import RemotePanel from '$lib/components/RemotePanel.svelte';
	let section = $state('chat');
	let draft = $state('');
	let thinking = $state(false);
	let status = $state<AssistantStatus | null>(null);
	let error = $state('');
	let pendingCloud = $state<string | null>(null);
	let list = $state<HTMLElement | null>(null);
	let controller: AbortController | null = null;
	const previousMonth = new Date(new Date().getFullYear(), new Date().getMonth() - 1, 1);
	let year = $state(previousMonth.getFullYear());
	let month = $state(previousMonth.getMonth() + 1);
	const suggestions = [
		'Comment va mon épargne ?',
		'Quels sont mes risques ?',
		'Résume ma situation en 3 phrases'
	];
	const tiers: Record<string, string> = {
		data: 'Moteur financier',
		n1: 'Appareil local',
		n2: 'Homelab privé',
		n3: 'Cloud — contexte minimisé',
		'': 'Moteur de repli'
	};
	const states: Record<string, string> = {
		grounded: 'Réponse fondée',
		clarification: 'Précision nécessaire',
		clarification_needed: 'Précision nécessaire',
		unsupported: 'Demande non prise en charge',
		unavailable: 'Analyse indisponible',
		incapable: 'Analyse impossible'
	};
	$effect(() => {
		void session.api
			.assistantStatus()
			.then((s) => (status = s))
			.catch((e) => (error = messageOf(e)));
	});
	onDestroy(() => controller?.abort());
	async function ask(question: string, allowCloud = false) {
		if (thinking) return;
		if (allowCloud && session.profile?.privacy_default !== 'N2') {
			error = 'Le cloud est désactivé pour ce profil.';
			return;
		}
		const history = session.chat.slice(-18).map((m) => ({ role: m.role, text: m.text }));
		if (!allowCloud)
			session.chat = [...session.chat, { role: 'user' as const, text: question, tier: '' }].slice(
				-20
			);
		draft = '';
		thinking = true;
		error = '';
		pendingCloud = null;
		controller = new AbortController();
		const generation = session.generation;
		try {
			const res = await session.api.ask(question, allowCloud, history, controller.signal);
			if (generation !== session.generation) return;
			session.chat = [
				...session.chat,
				{
					role: 'assistant' as const,
					text: res.answer,
					tier: res.tier,
					state: res.state,
					facts: res.facts
				}
			].slice(-20);
			if (
				res.cloud_eligible &&
				status?.cloud_configured &&
				!allowCloud &&
				session.profile?.privacy_default === 'N2'
			)
				pendingCloud = question;
		} catch (e) {
			if (generation === session.generation)
				error = controller.signal.aborted ? 'Demande annulée.' : messageOf(e);
		} finally {
			thinking = false;
			await tick();
			list?.scrollTo({ top: list.scrollHeight, behavior: 'auto' });
		}
	}
	function submit(e: SubmitEvent) {
		e.preventDefault();
		if (draft.trim()) void ask(draft.trim());
	}
</script>

<svelte:head><title>Assistant · Opale</title></svelte:head>
<h1>Assistant</h1>
<div class="tabstrip">
	<button aria-pressed={section === 'chat'} onclick={() => (section = 'chat')}>Conversation</button
	><button aria-pressed={section === 'review'} onclick={() => (section = 'review')}
		>Bilan mensuel</button
	><button aria-pressed={section === 'twin'} onclick={() => (section = 'twin')}
		>Portrait financier</button
	>
</div>
{#if section === 'chat'}<section class="glass panel">
		<div class="section-heading">
			<h2>Parlons de ton patrimoine</h2>
			<button
				onclick={() => {
					controller?.abort();
					session.chat = [];
					pendingCloud = null;
				}}>Effacer l’historique</button
			>
		</div>
		<p class="muted">
			{status?.homelab_available ? 'Homelab disponible' : 'Homelab indisponible ou non configuré'} · {session
				.profile?.privacy_default === 'N2' && status?.cloud_configured
				? 'Cloud possible avec ton accord'
				: 'Cloud désactivé'}
		</p>
		<p class="muted">
			Les 20 derniers messages restent en mémoire dans cet onglet. Ils sont supprimés à la
			déconnexion ou au rechargement.
		</p>
		{#if error}<p class="notice error" role="alert">{error}</p>{/if}
		<div class="chat-log" bind:this={list} role="log" aria-live="polite" aria-label="Conversation">
			{#if !session.chat.length}<p class="empty">Pose une question sur tes finances.</p>
				<div class="actions">
					{#each suggestions as s}<button onclick={() => ask(s)} disabled={thinking}>{s}</button
						>{/each}
				</div>{/if}{#each session.chat as message, i (i)}<article
					class:user-message={message.role === 'user'}
					class="chat-message"
				>
					<h3>{message.role === 'user' ? 'Toi' : 'Opale'}</h3>
					<p class="whitespace-pre-wrap">{message.text}</p>
					{#if message.role === 'assistant'}<small
							>{tiers[message.tier] ?? message.tier}{message.state
								? ` · ${states[message.state] ?? message.state}`
								: ''}</small
						>{#if message.facts?.length}<details>
								<summary>Chiffres et provenance</summary>{#each message.facts as fact (fact.id)}<p>
										{fact.text ?? fact.id}{fact.value_cents !== undefined
											? ` : ${money(fact.value_cents, fact.unit === 'EUR' ? 'EUR' : 'EUR')}`
											: ''}<br /><small>{fact.period ?? ''} · {fact.source ?? 'Moteur Opale'}</small
										>
									</p>{/each}
							</details>{/if}{/if}
				</article>{/each}
		</div>
		{#if pendingCloud}<div class="notice">
				<p>
					Une analyse cloud peut recevoir une intention structurée et des agrégats financiers
					arrondis. Ces données restent sensibles. Les textes libres de cette conversation restent
					sur le service Opale de Vaycode.
				</p>
				<div class="actions">
					<button onclick={() => ask(pendingCloud!, true)}>Autoriser pour cette demande</button
					><button onclick={() => (pendingCloud = null)}>Rester en local</button>
				</div>
			</div>{/if}
		<form onsubmit={submit} class="actions">
			<label
				>Ta question<input
					bind:value={draft}
					maxlength="2000"
					required
					autocomplete="off"
					placeholder="Combien ai-je dépensé en courses ce mois-ci ?"
				/></label
			><button class="primary" disabled={thinking || !draft.trim()}
				>{thinking ? 'Analyse…' : 'Envoyer'}</button
			>{#if thinking}<button type="button" onclick={() => controller?.abort()}>Annuler</button>{/if}
		</form>
	</section>
{:else if section === 'review'}<div class="stack">
		<section class="glass panel form-grid">
			<label>Année<input type="number" min="2000" max="2100" bind:value={year} /></label><label
				>Mois<input type="number" min="1" max="12" bind:value={month} /></label
			>
		</section>
		<RemotePanel
			title="Bilan mensuel"
			path={`/v1/monthly-review?year=${year}&month=${month}`}
			description="Bilan local fondé sur les chiffres du moteur. Aucun accord cloud n’est transmis par cet écran."
		/>
	</div>{:else}<RemotePanel
		title="Portrait financier"
		path="/v1/twin"
		description="Faits utilisés par le moteur pour les simulations, risques et analyses."
	/>{/if}
