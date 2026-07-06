<script lang="ts">
	// Flux — les mouvements du mois (EF-020), résumé et navigation mensuelle.
	import { session } from '$lib/session.svelte';
	import { eurosFull, monthLabel, dayString } from '$lib/format';
	import type { Transaction, MonthSummary } from '$lib/api';
	import Amount from '$lib/components/Amount.svelte';

	let offset = $state(0); // décalage en mois vs aujourd'hui
	let transactions = $state<Transaction[]>([]);
	let summary = $state<MonthSummary | null>(null);
	let error = $state('');

	const month = $derived.by(() => {
		const d = new Date();
		return new Date(d.getFullYear(), d.getMonth() + offset, 1);
	});

	$effect(() => {
		const start = month;
		const end = new Date(start.getFullYear(), start.getMonth() + 1, 0);
		void (async () => {
			try {
				[transactions, summary] = await Promise.all([
					session.api.listTransactions(dayString(start), dayString(end)),
					session.api.monthSummary(start.getFullYear(), start.getMonth() + 1)
				]);
				error = '';
			} catch (e) {
				error = e instanceof Error ? e.message : String(e);
			}
		})();
	});

	/** Groupe par jour (les mouvements arrivent triés du plus récent). */
	const byDay = $derived.by(() => {
		const groups: { day: string; items: Transaction[] }[] = [];
		for (const t of transactions) {
			const day = new Date(t.occurred_on).toLocaleDateString('fr-FR', {
				weekday: 'long',
				day: 'numeric',
				month: 'long'
			});
			const last = groups.at(-1);
			if (last?.day === day) last.items.push(t);
			else groups.push({ day, items: [t] });
		}
		return groups;
	});
</script>

<div class="mb-5 flex items-center justify-between">
	<h1 class="text-2xl font-bold tracking-tight">Flux</h1>
	<div class="glass flex items-center gap-1 px-2 py-1">
		<button class="px-2 py-1 hover:text-accent" onclick={() => (offset -= 1)} aria-label="Mois précédent">‹</button>
		<span class="min-w-36 text-center text-sm font-medium capitalize">{monthLabel(month)}</span>
		<button
			class="px-2 py-1 hover:text-accent disabled:opacity-30"
			onclick={() => (offset += 1)}
			disabled={offset >= 0}
			aria-label="Mois suivant">›</button
		>
	</div>
</div>

{#if error}
	<div class="glass border-loss/40 p-4 text-sm">{error}</div>
{:else}
	{#if summary}
		<div class="mb-5 grid grid-cols-3 gap-3">
			<div class="glass p-4">
				<p class="text-xs font-semibold text-neutral-500 uppercase">Revenus</p>
				<p class="text-gain mt-1 font-bold"><Amount cents={summary.income_cents} /></p>
			</div>
			<div class="glass p-4">
				<p class="text-xs font-semibold text-neutral-500 uppercase">Dépenses</p>
				<p class="text-loss mt-1 font-bold"><Amount cents={-summary.expenses_cents} /></p>
			</div>
			<div class="glass p-4">
				<p class="text-xs font-semibold text-neutral-500 uppercase">Solde</p>
				<p class="mt-1 font-bold {summary.net_cents >= 0 ? 'text-gain' : 'text-loss'}">
					<Amount cents={summary.net_cents} signed />
				</p>
			</div>
		</div>
	{/if}

	{#if byDay.length === 0}
		<div class="glass p-8 text-center text-sm text-neutral-500">Aucun mouvement ce mois-ci.</div>
	{/if}

	{#each byDay as group (group.day)}
		<section class="glass mb-4 p-5">
			<p class="text-xs font-semibold text-neutral-500 uppercase">{group.day}</p>
			<ul class="mt-1 divide-y divide-black/5 dark:divide-white/10">
				{#each group.items as t (t.id)}
					<li class="flex items-center justify-between gap-4 py-2.5">
						<div class="min-w-0">
							<p class="truncate font-medium">{t.label}</p>
							{#if t.category_name}
								<span
									class="bg-accent/10 text-accent inline-block rounded-full px-2 py-0.5 text-[11px] font-medium"
								>
									{t.category_name}
								</span>
							{/if}
						</div>
						<p class="amount shrink-0 font-semibold {t.amount_cents >= 0 ? 'text-gain' : ''}">
							{eurosFull(t.amount_cents)}
						</p>
					</li>
				{/each}
			</ul>
		</section>
	{/each}
{/if}
