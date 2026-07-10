<script lang="ts">
	import type { Tone } from '$lib/format';
	import { toneClass } from '$lib/format';

	interface Props {
		label: string;
		value: string;
		unit?: string;
		tone?: Tone;
		sub?: string;
	}
	let { label, value, unit = '', tone = 'muted', sub = '' }: Props = $props();
</script>

<div class="rounded-lg border border-rig-800 bg-rig-950/40 p-4">
	<div class="mb-1 flex items-baseline justify-between">
		<span class="text-sm text-rig-400">{label}</span>
		<span class="text-2xl font-semibold tabular-nums {toneClass[tone]}">
			{value}{#if unit}<span class="text-sm font-normal text-rig-400"> {unit}</span>{/if}
		</span>
	</div>
	{#if sub}
		<p class="text-xs text-rig-500">{sub}</p>
	{/if}
</div>
