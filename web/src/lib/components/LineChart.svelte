<script lang="ts">
	// Courbe SVG maison — pas de dépendance graphique : une aire dégradée et
	// une ligne lissée, comme la trajectoire de l'app iOS.
	let {
		points,
		height = 180,
		stroke = 'var(--color-accent)',
		target = null,
		labels = []
	}: {
		points: number[];
		height?: number;
		stroke?: string;
		/** Ligne cible horizontale optionnelle (ex. seuil d'indépendance). */
		target?: number | null;
		labels?: string[];
	} = $props();

	const W = 600;
	const PAD = 8;

	const geometry = $derived.by(() => {
		if (points.length < 2) return null;
		const all = target !== null ? [...points, target] : points;
		const min = Math.min(...all);
		const max = Math.max(...all);
		const span = max - min || 1;
		const x = (i: number) => PAD + (i / (points.length - 1)) * (W - 2 * PAD);
		const y = (v: number) => PAD + (1 - (v - min) / span) * (height - 2 * PAD);

		const line = points.map((v, i) => `${i === 0 ? 'M' : 'L'}${x(i)},${y(v)}`).join(' ');
		const area = `${line} L${x(points.length - 1)},${height} L${x(0)},${height} Z`;
		return { line, area, targetY: target !== null ? y(target) : null };
	});

	const gradientID = `grad-${Math.random().toString(36).slice(2, 8)}`;
</script>

{#if geometry}
	<svg viewBox="0 0 {W} {height}" class="w-full" role="img" aria-label="Courbe du patrimoine">
		<defs>
			<linearGradient id={gradientID} x1="0" y1="0" x2="0" y2="1">
				<stop offset="0%" stop-color={stroke} stop-opacity="0.28" />
				<stop offset="100%" stop-color={stroke} stop-opacity="0" />
			</linearGradient>
		</defs>
		<path d={geometry.area} fill="url(#{gradientID})" />
		<path
			d={geometry.line}
			fill="none"
			stroke-width="3"
			stroke-linecap="round"
			stroke-linejoin="round"
			{stroke}
		/>
		{#if geometry.targetY !== null}
			<line
				x1={PAD}
				x2={W - PAD}
				y1={geometry.targetY}
				y2={geometry.targetY}
				stroke="var(--color-gain)"
				stroke-width="1.5"
				stroke-dasharray="6 4"
				opacity="0.8"
			/>
		{/if}
	</svg>
	{#if labels.length > 0}
		<div class="flex justify-between text-xs text-neutral-500">
			{#each labels as label (label)}<span>{label}</span>{/each}
		</div>
	{/if}
{/if}
