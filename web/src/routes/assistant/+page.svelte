<script lang="ts">
	// Assistant (EF-050/051) : chat avec la cascade N2 homelab → N3 cloud
	// anonymisé (consentement explicite, EIA-021/022) → repli moteur.
	import { tick } from 'svelte';
	import { session } from '$lib/session.svelte';
	import type { AssistantStatus } from '$lib/api';

	interface Message {
		role: 'user' | 'assistant';
		text: string;
		tier: string;
	}

	let messages = $state<Message[]>([]);
	let draft = $state('');
	let thinking = $state(false);
	let status = $state<AssistantStatus | null>(null);
	let pendingCloud = $state<string | null>(null);
	let list = $state<HTMLElement | null>(null);

	const suggestions = [
		'Comment va mon épargne ?',
		'Quels sont mes risques ?',
		'Résume ma situation en 3 phrases'
	];

	$effect(() => {
		void session.api.assistantStatus().then((s) => (status = s));
	});

	const tierLabel: Record<string, string> = {
		data: 'Moteur — réponse exacte',
		n1: 'iPhone — local',
		n2: 'Homelab — privé',
		n3: 'Cloud — anonymisé',
		'': 'Moteur — hors ligne'
	};

	async function ask(question: string, allowCloud = false) {
		if (!allowCloud) messages.push({ role: 'user', text: question, tier: '' });
		draft = '';
		thinking = true;
		pendingCloud = null;
		try {
			const res = await session.api.ask(question, allowCloud);
			messages.push({ role: 'assistant', text: res.answer, tier: res.tier });
			if (res.tier === '' && status?.cloud_configured && !allowCloud) {
				pendingCloud = question;
			}
		} catch (e) {
			messages.push({
				role: 'assistant',
				text: `Impossible de répondre : ${e instanceof Error ? e.message : e}`,
				tier: ''
			});
		} finally {
			thinking = false;
			await tick();
			list?.scrollTo({ top: list.scrollHeight, behavior: 'smooth' });
		}
	}

	function submit(event: SubmitEvent) {
		event.preventDefault();
		const q = draft.trim();
		if (q && !thinking) void ask(q);
	}
</script>

<div class="mb-5 flex items-center justify-between">
	<h1 class="text-2xl font-bold tracking-tight">Assistant</h1>
	{#if status}
		<div class="flex gap-2 text-[11px] font-medium">
			<span class="glass px-2.5 py-1 {status.homelab_available ? 'text-gain' : 'text-neutral-400'}">
				{status.homelab_available ? '● Homelab en ligne' : '○ Homelab hors ligne'}
			</span>
			<span class="glass px-2.5 py-1 {status.cloud_configured ? 'text-accent' : 'text-neutral-400'}">
				{status.cloud_configured ? '● Cloud (anonymisé)' : '○ Cloud non configuré'}
			</span>
		</div>
	{/if}
</div>

<div class="glass flex h-[70vh] flex-col p-4 md:p-6">
	<div bind:this={list} class="flex-1 space-y-3 overflow-y-auto pr-1">
		{#if messages.length === 0}
			<div class="flex h-full flex-col items-center justify-center gap-3">
				<span class="iridescent text-4xl">✦</span>
				<p class="text-sm text-neutral-500">Pose une question sur ton patrimoine</p>
				{#each suggestions as s (s)}
					<button
						onclick={() => ask(s)}
						class="glass hover:border-accent/50 rounded-full px-4 py-2 text-sm transition"
					>
						{s}
					</button>
				{/each}
			</div>
		{/if}

		{#each messages as message, i (i)}
			<div class="flex {message.role === 'user' ? 'justify-end' : 'justify-start'}">
				<div
					class="max-w-[85%] rounded-3xl px-4 py-3 text-sm {message.role === 'user'
						? 'bg-accent/20'
						: 'bg-white/60 dark:bg-white/10'}"
				>
					<p class="whitespace-pre-wrap">{message.text}</p>
					{#if message.role === 'assistant'}
						<p class="mt-1 text-[10px] text-neutral-400">
							🔒 {tierLabel[message.tier] ?? message.tier}
						</p>
					{/if}
				</div>
			</div>
		{/each}

		{#if pendingCloud}
			<button
				onclick={() => ask(pendingCloud!, true)}
				class="bg-accent/15 text-accent mx-auto block rounded-full px-4 py-2 text-xs font-semibold"
			>
				☁️ Analyser avec le modèle cloud (données anonymisées)
			</button>
		{/if}

		{#if thinking}
			<p class="animate-pulse text-xs text-neutral-400">Analyse en cours…</p>
		{/if}
	</div>

	<form onsubmit={submit} class="mt-4 flex gap-2">
		<input
			bind:value={draft}
			placeholder="Ta question…"
			class="focus:border-accent flex-1 rounded-2xl border border-black/10 bg-white/70 px-4 py-3 text-sm outline-none dark:border-white/10 dark:bg-white/10"
		/>
		<button
			type="submit"
			disabled={!draft.trim() || thinking}
			class="bg-accent rounded-2xl px-5 font-semibold text-white transition hover:brightness-105 disabled:opacity-40"
			aria-label="Envoyer"
		>
			↑
		</button>
	</form>
</div>
