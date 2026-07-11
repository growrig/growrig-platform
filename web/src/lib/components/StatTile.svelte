<script lang="ts">
	import type { Tone } from '$lib/format';
	import { toneClass } from '$lib/format';
	import Sparkline from '$lib/components/Sparkline.svelte';
	import Maximize2 from '@lucide/svelte/icons/maximize-2';

	interface Props {
		label: string;
		value: string;
		unit?: string;
		tone?: Tone;
		sub?: string;
		/** When provided, an embedded sparkline is drawn from these values. */
		spark?: number[];
		sparkColor?: string;
		sparkTarget?: number;
		/** When provided, the tile becomes a button that opens a detail view. */
		onclick?: () => void;
	}
	let {
		label,
		value,
		unit = '',
		tone = 'muted',
		sub = '',
		spark,
		sparkColor = 'var(--color-leaf)',
		sparkTarget,
		onclick
	}: Props = $props();

	const base =
		'group relative w-full rounded-lg border border-rig-800 bg-rig-950/40 p-4 text-left transition-colors';
	const clickable = 'hover:border-rig-600 cursor-pointer';
</script>

<svelte:element
	this={onclick ? 'button' : 'div'}
	class="{base} {onclick ? clickable : ''}"
	{...onclick ? { type: 'button', onclick } : {}}
>
	<div class="mb-1 flex items-baseline justify-between">
		<span class="text-sm text-rig-400">{label}</span>
		<span class="text-2xl font-semibold tabular-nums {toneClass[tone]}">
			{value}{#if unit}<span class="text-sm font-normal text-rig-400"> {unit}</span>{/if}
		</span>
	</div>
	{#if spark && spark.length > 1}
		<div class="mt-2">
			<Sparkline values={spark} color={sparkColor} target={sparkTarget} height={40} showNow={false} />
		</div>
	{/if}
	{#if sub}
		<p class="text-xs text-rig-500">{sub}</p>
	{/if}
	{#if onclick}
		<Maximize2
			size={13}
			class="absolute right-3 top-3 text-rig-600 opacity-0 transition-opacity group-hover:opacity-100"
		/>
	{/if}
</svelte:element>
