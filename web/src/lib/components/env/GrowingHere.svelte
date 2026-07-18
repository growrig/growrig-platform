<script lang="ts">
	import { onMount } from 'svelte';
	import { live } from '$lib/live.svelte';
	import { getEnvironmentPlants, getCultivars } from '$lib/api';
	import { daysSince, titleCase, vpdZone, toneClass } from '$lib/format';
	import type { EnvPlantsGroup, Cultivar, GrowCultivarRef, EnvironmentView } from '$lib/types';
	import CultivarThumbnails from '$lib/components/CultivarThumbnails.svelte';
	import Sprout from '@lucide/svelte/icons/sprout';
	import Zap from '@lucide/svelte/icons/zap';

	interface Props {
		env: EnvironmentView;
	}
	let { env }: Props = $props();

	let groups = $state<EnvPlantsGroup[]>([]);
	let cultivars = $state<Cultivar[]>([]);

	onMount(() => {
		getEnvironmentPlants(env.id).then((g) => (groups = g)).catch(() => {});
		getCultivars().then((c) => (cultivars = c)).catch(() => {});
	});

	// Rooms don't hold plants directly; they connect to tents they feed air to.
	const connectedTents = $derived(
		env.kind === 'room'
			? (live.snapshot?.environments ?? []).filter((e) => e.airSourceId === env.id)
			: []
	);

	function countPlants(units: EnvPlantsGroup['units']): number {
		return units.filter((u) => u.status === 'active').reduce((n, u) => n + u.quantity, 0);
	}
	function cultivarRefs(units: EnvPlantsGroup['units']): GrowCultivarRef[] {
		const refs: GrowCultivarRef[] = [];
		const idx = new Map<string, number>();
		for (const u of units) {
			if (u.status !== 'active') continue;
			if (idx.has(u.cultivar)) refs[idx.get(u.cultivar)!].count += u.quantity;
			else {
				idx.set(u.cultivar, refs.length);
				refs.push({ cultivar: u.cultivar, count: u.quantity });
			}
		}
		return refs;
	}
</script>

<section>
	<h2 class="mb-3 text-sm font-semibold uppercase tracking-wide text-rig-400">
		{env.kind === 'room' ? 'Connected environments' : 'Growing here'}
	</h2>

	{#if env.kind === 'room'}
		{#if connectedTents.length}
			<div class="grid gap-3 sm:grid-cols-2">
				{#each connectedTents as tent (tent.id)}
					<a
						href="/env/{tent.id}"
						class="group flex items-center justify-between rounded-lg border border-rig-800 bg-rig-950/40 p-4 transition-colors hover:border-rig-600"
					>
						<span class="font-medium">{tent.name}</span>
						{#if tent.hasClimate}
							<span class="text-sm tabular-nums text-rig-300">
								{tent.tempC.toFixed(1)}° · {tent.humidity.toFixed(0)}% ·
								<span class={toneClass[vpdZone(tent.vpd).tone]}>{tent.vpd.toFixed(2)}</span>
							</span>
						{:else}
							<span class="text-sm text-rig-500">no data</span>
						{/if}
					</a>
				{/each}
			</div>
		{:else}
			<div class="rounded-xl border border-dashed border-rig-800 p-6 text-center text-sm text-rig-500">
				No grow boxes draw air from this room yet.
			</div>
		{/if}
	{:else if groups.length}
		<div class="grid gap-3 {groups.length > 1 ? 'sm:grid-cols-2' : ''}">
			{#each groups as g (g.grow.id)}
				{@const isControl = g.grow.id === env.controlGrowId}
				<a
					href="/grows/{g.grow.id}"
					class="group rounded-lg border border-rig-800 bg-rig-950/40 p-4 transition-colors hover:border-rig-600"
				>
					<div class="flex items-start justify-between gap-3">
						<div class="min-w-0">
							<div class="flex items-center gap-2">
								<span class="truncate font-medium transition-colors group-hover:text-rig-50">{g.grow.name}</span>
								<span class="shrink-0 rounded-full bg-rig-800 px-2 py-0.5 text-xs capitalize text-leaf">
									{titleCase(g.grow.stage)}
								</span>
							</div>
							<div class="mt-1 flex items-center gap-2 text-xs text-rig-500">
								<span class="inline-flex items-center gap-1"><Sprout size={12} /> {countPlants(g.units)} plant{countPlants(g.units) === 1 ? '' : 's'}</span>
								<span>·</span>
								<span>Day {daysSince(g.grow.startedAt)}</span>
							</div>
						</div>
						{#if isControl}
							<span class="inline-flex shrink-0 items-center gap-1 text-[11px] text-leaf" title="This grow's stage drives the environment's automation">
								<Zap size={11} /> Automation follows this grow
							</span>
						{/if}
					</div>
					<div class="mt-3">
						<CultivarThumbnails refs={cultivarRefs(g.units)} {cultivars} />
					</div>
				</a>
			{/each}
		</div>
	{:else}
		<div class="flex flex-wrap items-center justify-between gap-3 rounded-xl border border-dashed border-rig-800 p-6">
			<span class="text-sm text-rig-400">Nothing is growing here yet</span>
			<a
				href="/grows"
				class="inline-flex items-center gap-1.5 rounded-md bg-rig-50 px-4 py-1.5 text-sm font-medium text-rig-950 transition-colors hover:bg-rig-200"
			>
				<Sprout size={15} /> Place plants
			</a>
		</div>
	{/if}
</section>
