<script lang="ts">
	import type { Tone } from '$lib/format';
	import { toneClass } from '$lib/format';
	import ThresholdSparkline from '$lib/components/ThresholdSparkline.svelte';

	interface Point {
		t: number;
		v: number;
	}

	interface Props {
		label: string;
		value: string;
		unit?: string;
		status: string;
		tone: Tone;
		points: Point[];
		classify: (v: number) => Tone;
		onclick?: () => void;
	}
	let { label, value, unit = '', status, tone, points, classify, onclick }: Props = $props();

	const pillClass: Record<Tone, string> = {
		good: 'border-leaf/40 text-leaf',
		warn: 'border-warn/50 text-warn',
		danger: 'border-danger/50 text-danger',
		muted: 'border-rig-700 text-rig-400'
	};
</script>

<svelte:element
	this={onclick ? 'button' : 'div'}
	{onclick}
	type={onclick ? 'button' : undefined}
	role={onclick ? 'button' : undefined}
	class="w-full rounded-xl border border-rig-800 bg-rig-900/50 px-4 py-3 text-left transition-colors {onclick
		? 'cursor-pointer hover:border-rig-600 focus-visible:border-rig-500 focus-visible:outline-none'
		: ''}"
>
	<div class="flex items-center justify-between gap-3">
		<span class="text-sm text-rig-300">{label}</span>
		<div class="flex items-center gap-2">
			<span class="text-sm font-semibold tabular-nums {toneClass[tone]}">
				{value}{#if unit}<span class="ml-0.5 font-normal opacity-80">{unit}</span>{/if}
			</span>
			{#if status}
				<span
					class="rounded-full border px-2 py-0.5 text-[11px] font-medium {pillClass[tone]}"
				>
					{status}
				</span>
			{/if}
		</div>
	</div>

	{#if points.length > 1}
		<div class="mt-2">
			<ThresholdSparkline {points} {classify} height={72} />
		</div>
	{:else}
		<p class="mt-3 text-xs text-rig-600">No recent readings</p>
	{/if}
</svelte:element>
