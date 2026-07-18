<script lang="ts">
	import { onMount } from 'svelte';
	import { live } from '$lib/live.svelte';
	import { auth } from '$lib/auth.svelte';
	import { getStagePresets, getCultivars } from '$lib/api';
	import type { GrowView, StagePresets, Cultivar } from '$lib/types';
	import GrowFormModal from '$lib/components/GrowFormModal.svelte';
	import GrowCard from '$lib/components/GrowCard.svelte';
	import Sprout from '@lucide/svelte/icons/sprout';
	import Plus from '@lucide/svelte/icons/plus';

	const snap = $derived(live.snapshot);
	const grows = $derived(snap?.grows ?? []);
	const active = $derived(grows.filter((g) => g.status === 'active'));
	const inactive = $derived(grows.filter((g) => g.status !== 'active'));

	let presets = $state<StagePresets>({});
	let creating = $state(false);

	// Cultivars are reference data (not in the live snapshot), fetched over REST
	// for thumbnail resolution on grow cards.
	let cultivars = $state<Cultivar[]>([]);

	onMount(() => {
		getStagePresets().then((p) => (presets = p)).catch(() => {});
		getCultivars().then((c) => (cultivars = c)).catch(() => {});
	});
</script>

<div class="mb-6 flex items-start justify-between gap-4">
	<div>
		<h1 class="text-2xl font-semibold">Grows</h1>
		<p class="text-sm text-rig-400">Cultivation runs and the plants they track across your environments.</p>
	</div>
	{#if auth.isAdmin}
		<button
			onclick={() => (creating = true)}
			class="inline-flex items-center gap-1.5 rounded-md bg-rig-50 px-4 py-2 text-sm font-medium text-rig-950 transition-colors hover:bg-rig-200"
		>
			<Plus size={15} /> New grow
		</button>
	{/if}
</div>

<div class="space-y-10">
	{#if !snap}
		<p class="text-rig-400">Connecting to Grow Core…</p>
	{:else if grows.length === 0}
		<div class="rounded-xl border border-dashed border-rig-800 p-10 text-center">
			<div class="mb-3 flex justify-center text-rig-500"><Sprout size={40} /></div>
			<h2 class="mb-1 text-lg font-semibold">No grows yet</h2>
			<p class="mb-5 text-sm text-rig-400">Start a grow to track plants and their placements across environments.</p>
			{#if auth.isAdmin}
				<button
					onclick={() => (creating = true)}
					class="rounded-md bg-rig-50 px-5 py-2 text-sm font-medium text-rig-950 transition-colors hover:bg-rig-200"
				>
					Start a grow
				</button>
			{/if}
		</div>
	{:else}
		<div class="space-y-8">
			{#snippet growRow(g: GrowView)}
				<GrowCard grow={g} {cultivars} />
			{/snippet}

			<section>
				<h2 class="mb-3 text-sm font-semibold uppercase tracking-wide text-rig-400">Active</h2>
				{#if active.length}
					<div class="grid gap-3 sm:grid-cols-2">
						{#each active as g (g.id)}{@render growRow(g)}{/each}
					</div>
				{:else}
					<p class="text-sm text-rig-500">No active grows.</p>
				{/if}
			</section>

			{#if inactive.length}
				<section>
					<h2 class="mb-3 text-sm font-semibold uppercase tracking-wide text-rig-400">Completed &amp; archived</h2>
					<div class="grid gap-3 sm:grid-cols-2">
						{#each inactive as g (g.id)}{@render growRow(g)}{/each}
					</div>
				</section>
			{/if}
		</div>
	{/if}
</div>

{#if auth.isAdmin}
	<GrowFormModal bind:open={creating} {presets} />
{/if}
