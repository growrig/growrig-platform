<script lang="ts">
	import { onMount } from 'svelte';
	import { getEnvironmentPlants } from '$lib/api';
	import type { EnvPlantsGroup } from '$lib/types';
	import { titleCase } from '$lib/format';
	import Sprout from '@lucide/svelte/icons/sprout';

	interface Props {
		environmentId: string;
	}
	let { environmentId }: Props = $props();

	let groups = $state<EnvPlantsGroup[]>([]);

	function reload() {
		getEnvironmentPlants(environmentId)
			.then((g) => (groups = g))
			.catch(() => {});
	}
	onMount(reload);

	function countPlants(units: EnvPlantsGroup['units']): number {
		return units.filter((u) => u.status === 'active').reduce((n, u) => n + u.quantity, 0);
	}
</script>

{#if groups.length}
	<section>
		<h2 class="mb-3 text-sm font-semibold uppercase tracking-wide text-rig-400">Occupants</h2>
		<div class="space-y-3">
			{#each groups as g (g.grow.id)}
				<div class="rounded-lg border border-rig-800 bg-rig-950/40 p-4">
					<div class="flex items-center justify-between">
						<a href="/grows/{g.grow.id}" class="font-medium hover:text-leaf">{g.grow.name}</a>
						<div class="flex items-center gap-2 text-sm text-rig-400">
							<span class="rounded-full bg-rig-800 px-2 py-0.5 text-xs capitalize text-leaf">{g.grow.stage}</span>
							<span class="inline-flex items-center gap-1"><Sprout size={13} /> {countPlants(g.units)}</span>
						</div>
					</div>
					<div class="mt-3 flex flex-wrap gap-2">
						{#each g.units as u (u.id)}
							<a
								href="/plants/{u.id}"
								class="rounded-md border border-rig-800 bg-rig-900/40 px-2.5 py-1 text-xs text-rig-300 transition-colors hover:border-rig-600"
							>
								{u.label || 'Plant'}
								{#if u.tracking === 'group'}<span class="text-rig-500"> ×{u.quantity}</span>{/if}
								{#if u.status !== 'active'}<span class="ml-1 text-rig-600">· {titleCase(u.status)}</span>{/if}
							</a>
						{/each}
					</div>
				</div>
			{/each}
		</div>
	</section>
{/if}
