<script lang="ts">
	import type { GrowView, Cultivar } from '$lib/types';
	import { titleCase } from '$lib/format';
	import CultivarThumbnails from '$lib/components/CultivarThumbnails.svelte';
	import Sprout from '@lucide/svelte/icons/sprout';
	import MapPin from '@lucide/svelte/icons/map-pin';

	interface Props {
		grow: GrowView;
		/** Cultivar library, used to resolve per-plant thumbnails by name. */
		cultivars?: Cultivar[];
	}
	let { grow, cultivars = [] }: Props = $props();

	const isActive = $derived(grow.status === 'active');
</script>

<a
	href="/grows/{grow.id}"
	class="block rounded-xl border border-rig-800 bg-rig-900/50 p-4 transition-colors hover:border-rig-600"
>
	<div class="mb-2 flex items-center justify-between gap-2">
		<h3 class="min-w-0 truncate font-semibold">{grow.name}</h3>
		<div class="flex shrink-0 items-center gap-2">
			<span class="text-sm capitalize text-rig-400">{titleCase(grow.species) || 'No species'}</span>
			<span class="rounded-full bg-rig-800 px-2 py-0.5 text-xs capitalize {isActive ? 'text-leaf' : 'text-rig-400'}">
				{isActive ? grow.stage || '—' : titleCase(grow.status)}
			</span>
		</div>
	</div>
	<div class="flex flex-wrap items-center gap-x-3 gap-y-1 text-xs text-rig-500">
		<span class="tabular-nums">day {grow.totalDays}</span>
		<span>·</span>
		<span class="inline-flex items-center gap-1"><Sprout size={12} /> {grow.plantCount} plants</span>
		<span>·</span>
		<span>{grow.stageDays}d in {grow.stage}</span>
		{#if (grow.environments ?? []).length}
			<span>·</span>
			<span class="inline-flex items-center gap-1"><MapPin size={11} /> {(grow.environments ?? []).map((e) => e.name).join(', ')}</span>
		{/if}
	</div>
	{#if (grow.cultivars ?? []).length}
		<div class="mt-3"><CultivarThumbnails refs={grow.cultivars ?? []} {cultivars} /></div>
	{/if}
</a>
