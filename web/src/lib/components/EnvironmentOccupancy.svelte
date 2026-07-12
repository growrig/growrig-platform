<script lang="ts">
	import { onMount } from 'svelte';
	import { getEnvironmentPlants, getCultivars } from '$lib/api';
	import type { EnvPlantsGroup, Cultivar, GrowCultivarRef } from '$lib/types';
	import CultivarThumbnails from '$lib/components/CultivarThumbnails.svelte';
	import Sprout from '@lucide/svelte/icons/sprout';

	interface Props {
		environmentId: string;
	}
	let { environmentId }: Props = $props();

	let groups = $state<EnvPlantsGroup[]>([]);
	let cultivars = $state<Cultivar[]>([]);

	function reload() {
		getEnvironmentPlants(environmentId)
			.then((g) => (groups = g))
			.catch(() => {});
		getCultivars()
			.then((c) => (cultivars = c))
			.catch(() => {});
	}
	onMount(reload);

	function countPlants(units: EnvPlantsGroup['units']): number {
		return units.filter((u) => u.status === 'active').reduce((n, u) => n + u.quantity, 0);
	}

	// Aggregate active units into per-cultivar counts, in first-seen order.
	function cultivarRefs(units: EnvPlantsGroup['units']): GrowCultivarRef[] {
		const refs: GrowCultivarRef[] = [];
		const idx = new Map<string, number>();
		for (const u of units) {
			if (u.status !== 'active') continue;
			const key = u.cultivar;
			if (idx.has(key)) refs[idx.get(key)!].count += u.quantity;
			else {
				idx.set(key, refs.length);
				refs.push({ cultivar: key, count: u.quantity });
			}
		}
		return refs;
	}
</script>

{#if groups.length}
	<section>
		<h2 class="mb-3 text-sm font-semibold uppercase tracking-wide text-rig-400">Occupants</h2>
		<div class="space-y-3">
			{#each groups as g (g.grow.id)}
				<a
					href="/grows/{g.grow.id}"
					class="group block rounded-lg border border-rig-800 bg-rig-950/40 p-4 transition-colors hover:border-rig-600 focus-visible:border-rig-500 focus-visible:outline-none"
				>
					<div class="flex items-center justify-between">
						<span class="font-medium">{g.grow.name}</span>
						<div class="flex items-center gap-2 text-sm text-rig-400">
							<span class="rounded-full bg-rig-800 px-2 py-0.5 text-xs capitalize text-leaf">{g.grow.stage}</span>
							<span class="inline-flex items-center gap-1"><Sprout size={13} /> {countPlants(g.units)}</span>
						</div>
					</div>
					<div class="mt-3">
						<CultivarThumbnails refs={cultivarRefs(g.units)} {cultivars} />
					</div>
				</a>
			{/each}
		</div>
	</section>
{/if}
