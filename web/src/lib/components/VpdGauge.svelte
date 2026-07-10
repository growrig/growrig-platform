<script lang="ts">
	import { vpdZone, toneClass } from '$lib/format';

	interface Props {
		vpd: number;
		ok: boolean;
	}
	let { vpd, ok }: Props = $props();

	const zone = $derived(vpdZone(vpd));
	// Scale 0–2 kPa across the bar.
	const pct = $derived(Math.max(0, Math.min(100, (vpd / 2) * 100)));
</script>

<div class="rounded-lg border border-rig-800 bg-rig-950/40 p-4">
	<div class="mb-1 flex items-baseline justify-between">
		<span class="text-sm text-rig-400">VPD</span>
		{#if ok}
			<span class="text-2xl font-semibold tabular-nums {toneClass[zone.tone]}">
				{vpd.toFixed(2)} <span class="text-sm font-normal text-rig-400">kPa</span>
			</span>
		{:else}
			<span class="text-rig-500">—</span>
		{/if}
	</div>
	{#if ok}
		<!-- zone scale: propagation | vegetative | flowering | dry -->
		<div class="relative mt-3 h-2 rounded-full bg-gradient-to-r from-sky-500/40 via-leaf/50 to-danger/50">
			<div
				class="absolute top-1/2 h-3.5 w-1 -translate-y-1/2 rounded-full bg-rig-50 shadow"
				style="left:{pct}%"
			></div>
		</div>
		<div class="mt-1.5 flex justify-between text-[10px] text-rig-500">
			<span>0</span>
			<span class={toneClass[zone.tone]}>{zone.label}</span>
			<span>2 kPa</span>
		</div>
	{:else}
		<p class="mt-2 text-xs text-rig-500">needs temperature + humidity</p>
	{/if}
</div>
