<script lang="ts">
	// Jauge circulaire du score de santé (EF-015) — équivalent du Gauge iOS.
	let { score, size = 84 }: { score: number; size?: number } = $props();

	const R = 34;
	const C = 2 * Math.PI * R;
	const dash = $derived((score / 100) * C);
	const color = $derived(
		score >= 70 ? 'var(--color-gain)' : score >= 40 ? '#e0a13c' : 'var(--color-loss)'
	);
</script>

<svg
	viewBox="0 0 84 84"
	width={size}
	height={size}
	role="img"
	aria-label="Score de santé {score} sur 100"
>
	<circle cx="42" cy="42" r={R} fill="none" stroke="currentColor" stroke-width="7" opacity="0.12" />
	<circle
		cx="42"
		cy="42"
		r={R}
		fill="none"
		stroke={color}
		stroke-width="7"
		stroke-linecap="round"
		stroke-dasharray="{dash} {C}"
		transform="rotate(-90 42 42)"
		style="transition: stroke-dasharray 0.8s ease"
	/>
	<text
		x="42"
		y="47"
		text-anchor="middle"
		class="fill-current text-[1.35rem] font-bold"
	>
		{score}
	</text>
</svg>
