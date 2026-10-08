<script lang="ts">
	import { untrack } from 'svelte';
	import { parseAmount, amountInput, messageOf } from '$lib/domain';
	import type { Field } from '$lib/forms';
	let {
		fields,
		initial = {},
		submit,
		cancel,
		label = 'Enregistrer'
	}: {
		fields: Field[];
		initial?: Record<string, any>;
		submit: (value: Record<string, any>) => Promise<void>;
		cancel?: () => void;
		label?: string;
	} = $props();
	const exponent = (f: Field, v: Record<string, any>) =>
		typeof f.exponent === 'function' ? f.exponent(v) : (f.exponent ?? 2);
	let values = $state<Record<string, any>>({});
	let error = $state('');
	let busy = $state(false);
	$effect(() => {
		const source = initial;
		// Updating asynchronous select options must not discard a form already being edited.
		values = untrack(() =>
			Object.fromEntries(
				fields.map((f) => {
					const value = source[f.key] ?? f.default ?? (f.type === 'checkbox' ? false : '');
					return [
						f.key,
						f.type === 'money' && value !== ''
							? amountInput(Number(value), exponent(f, source))
							: f.type === 'percent' && value !== ''
								? String(Number(value) / 100)
								: f.type === 'date' && value
									? String(value).slice(0, 10)
									: value
					];
				})
			)
		);
	});
	async function save(event: SubmitEvent) {
		event.preventDefault();
		busy = true;
		error = '';
		try {
			const result: Record<string, any> = {};
			for (const f of fields) {
				const v = values[f.key];
				if (!f.required && (v === '' || v == null) && f.omitEmpty) continue;
				result[f.key] =
					f.type === 'money' || f.type === 'percent'
						? parseAmount(String(v || '0'), f.type === 'percent' ? 2 : exponent(f, values))
						: f.type === 'number'
							? Number(v || 0)
							: f.type === 'checkbox'
								? Boolean(v)
								: (v ?? '');
			}
			await submit(result);
		} catch (e) {
			error = messageOf(e);
		} finally {
			busy = false;
		}
	}
</script>

<form onsubmit={save} class="opale-form">
	<div class="form-grid">
		{#each fields as field (field.key)}
			<label class:wide={field.type === 'textarea'}>
				<span
					>{field.label}{field.currency ? ` (${field.currency(values)})` : ''}{field.required
						? ' *'
						: ''}</span
				>
				{#if field.type === 'select'}<select
						bind:value={values[field.key]}
						required={field.required}
						disabled={busy || field.readonly}
						><option value="">{field.required ? 'Choisir…' : 'Aucun'}</option
						>{#each field.options ?? [] as option}<option value={option.value}
								>{option.label}</option
							>{/each}</select
					>
				{:else if field.type === 'textarea'}<textarea
						rows="3"
						bind:value={values[field.key]}
						required={field.required}
						disabled={busy}></textarea>
				{:else if field.type === 'checkbox'}<input
						type="checkbox"
						bind:checked={values[field.key]}
						disabled={busy}
					/>
				{:else}<input
						type={field.type === 'money' || field.type === 'percent'
							? 'text'
							: (field.type ?? 'text')}
						inputmode={field.type === 'money' || field.type === 'percent' ? 'decimal' : undefined}
						bind:value={values[field.key]}
						min={field.min}
						max={field.max}
						step={field.step ?? 1}
						required={field.required}
						disabled={busy || field.readonly}
						autocomplete={field.type === 'password' ? 'new-password' : undefined}
					/>{/if}
				{#if field.help}<small>{field.help}</small>{/if}
			</label>{/each}
	</div>
	{#if error}<p class="notice error" role="alert">{error}</p>{/if}
	<div class="actions">
		<button class="primary" disabled={busy}>{busy ? 'Enregistrement…' : label}</button
		>{#if cancel}<button type="button" onclick={cancel} disabled={busy}>Annuler</button>{/if}
	</div>
</form>
