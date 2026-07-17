<script lang="ts">
	import type { Tone } from '$lib/format';

	interface Point {
		t: number;
		v: number;
	}

	interface Props {
		points: Point[];
		/** Classify each value into a status tone that drives segment color. */
		classify: (v: number) => Tone;
		height?: number;
		/** Show start / mid / end time labels under the chart. */
		showTimes?: boolean;
	}
	let { points, classify, height = 56, showTimes = true }: Props = $props();

	const width = 400;

	const toneStroke: Record<Tone, string> = {
		good: 'var(--color-leaf)',
		warn: 'var(--color-warn)',
		danger: 'var(--color-danger)',
		muted: 'var(--color-rig-400)'
	};

	const bounds = $derived.by(() => {
		if (points.length === 0) return { min: 0, max: 1 };
		let min = Math.min(...points.map((p) => p.v));
		let max = Math.max(...points.map((p) => p.v));
		if (max - min < 1) {
			min -= 0.5;
			max += 0.5;
		}
		const pad = (max - min) * 0.12;
		return { min: min - pad, max: max + pad };
	});

	function x(i: number): number {
		if (points.length <= 1) return 0;
		return (i / (points.length - 1)) * width;
	}

	function y(v: number): number {
		const { min, max } = bounds;
		return height - ((v - min) / (max - min)) * height;
	}

	// Area fill under the whole series (muted).
	const area = $derived.by(() => {
		if (points.length === 0) return '';
		const top = points.map((p, i) => `${i === 0 ? 'M' : 'L'}${x(i).toFixed(1)},${y(p.v).toFixed(1)}`).join(' ');
		return `${top} L${x(points.length - 1).toFixed(1)},${height} L0,${height} Z`;
	});

	// Each segment i-1→i is colored by the end point's tone; contiguous same-tone
	// runs merge into one polyline so joins stay clean.
	const segments = $derived.by(() => {
		if (points.length < 2) return [] as { tone: Tone; d: string }[];
		const out: { tone: Tone; d: string }[] = [];
		let tone = classify(points[1].v);
		let d = `M${x(0).toFixed(1)},${y(points[0].v).toFixed(1)} L${x(1).toFixed(1)},${y(points[1].v).toFixed(1)}`;
		for (let i = 2; i < points.length; i++) {
			const next = classify(points[i].v);
			const xi = x(i).toFixed(1);
			const yi = y(points[i].v).toFixed(1);
			if (next !== tone) {
				out.push({ tone, d });
				// Restart from previous point so there's no visual gap at the boundary.
				d = `M${x(i - 1).toFixed(1)},${y(points[i - 1].v).toFixed(1)} L${xi},${yi}`;
				tone = next;
			} else {
				d += ` L${xi},${yi}`;
			}
		}
		out.push({ tone, d });
		return out;
	});

	const timeLabels = $derived.by(() => {
		if (!showTimes || points.length < 2) return [] as string[];
		const fmt = (ms: number) =>
			new Date(ms).toLocaleTimeString([], { hour: 'numeric', minute: '2-digit' });
		const mid = points[Math.floor(points.length / 2)];
		return [fmt(points[0].t), fmt(mid.t), fmt(points[points.length - 1].t)];
	});
</script>

{#if points.length > 0}
	<div>
		<svg viewBox="0 0 {width} {height}" class="w-full" preserveAspectRatio="none" style="height:{height}px">
			{#if area}
				<path d={area} fill="currentColor" class="text-rig-700" opacity="0.35" />
			{/if}
			{#each segments as seg}
				<path
					d={seg.d}
					fill="none"
					stroke={toneStroke[seg.tone]}
					stroke-width="2.25"
					stroke-linejoin="round"
					stroke-linecap="round"
					vector-effect="non-scaling-stroke"
				/>
			{/each}
		</svg>
		{#if timeLabels.length}
			<div class="mt-1 flex justify-between text-[10px] tabular-nums text-rig-600">
				{#each timeLabels as label}
					<span>{label}</span>
				{/each}
			</div>
		{/if}
	</div>
{/if}
