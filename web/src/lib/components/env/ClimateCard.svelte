<script lang="ts">
	import Sparkline from '$lib/components/Sparkline.svelte';
	import type { Tone } from '$lib/format';
	import { toneClass } from '$lib/format';

	interface Props {
		label: string;
		/** Current value already formatted (e.g. "27.6"). */
		value: string;
		unit?: string;
		/** Raw current number, for range/target status math. */
		current?: number;
		/** Single setpoint (0/undefined = none). */
		target?: number;
		/** Optional display band (0 = unset). */
		rangeMin?: number;
		rangeMax?: number;
		spark?: number[];
		sparkColor?: string;
		/** Explicit tone override (else derived from range/target membership). */
		tone?: Tone;
		/** ISO timestamp of the latest reading, shown on hover. */
		updatedAt?: string;
		onclick?: () => void;
	}
	let {
		label,
		value,
		unit = '',
		current,
		target,
		rangeMin,
		rangeMax,
		spark,
		sparkColor = 'var(--color-leaf)',
		tone,
		updatedAt,
		onclick
	}: Props = $props();

	const hasRange = $derived(!!rangeMin && !!rangeMax && rangeMax! > rangeMin!);
	const hasTarget = $derived(!!target && target! > 0);

	// Where the target/range comes through as a caption under the value.
	const targetText = $derived.by(() => {
		if (hasRange) return `target ${rangeMin}–${rangeMax}${unit}`;
		if (hasTarget) return `target ${target}${unit}`;
		return '';
	});

	// Difference/status relative to the band or setpoint.
	const status = $derived.by(() => {
		if (current == null) return '';
		if (hasRange) {
			if (current > rangeMax!) return `+${(current - rangeMax!).toFixed(1)} above`;
			if (current < rangeMin!) return `−${(rangeMin! - current).toFixed(1)} below`;
			return 'in range';
		}
		if (hasTarget) {
			const d = current - target!;
			if (Math.abs(d) < 0.05) return 'on target';
			return `${d > 0 ? '+' : '−'}${Math.abs(d).toFixed(1)} ${d > 0 ? 'above' : 'below'} target`;
		}
		return '';
	});

	// Derive a tone from range membership when not given one.
	const derivedTone = $derived.by<Tone>(() => {
		if (tone) return tone;
		if (current == null) return 'muted';
		if (hasRange) return current >= rangeMin! && current <= rangeMax! ? 'good' : 'warn';
		return 'muted';
	});
</script>

<svelte:element
	this={onclick ? 'button' : 'div'}
	{onclick}
	type={onclick ? 'button' : undefined}
	role={onclick ? 'button' : undefined}
	title={updatedAt ? `Updated ${new Date(updatedAt).toLocaleTimeString()}` : undefined}
	class="group flex w-full flex-col rounded-xl border border-rig-800 bg-rig-900/40 p-4 text-left transition-colors {onclick
		? 'cursor-pointer hover:border-rig-600 focus-visible:border-leaf focus-visible:outline-none'
		: ''}"
>
	<div class="flex items-start justify-between gap-2">
		<span class="text-xs uppercase tracking-wide text-rig-500">{label}</span>
		{#if spark && spark.length}
			<div class="w-20 opacity-80"><Sparkline values={spark} color={sparkColor} height={26} showNow={false} /></div>
		{/if}
	</div>
	<div class="mt-1 text-3xl font-semibold tabular-nums {toneClass[derivedTone]}">
		{value}<span class="ml-0.5 text-base font-normal text-rig-500">{unit}</span>
	</div>
	<div class="mt-1 flex items-center justify-between gap-2 text-xs">
		<span class="text-rig-500">{targetText}</span>
		{#if status}<span class={toneClass[derivedTone]}>{status}</span>{/if}
	</div>
</svelte:element>
