<script lang="ts">
	import { onMount } from 'svelte';
	import { live } from '$lib/live.svelte';
	import { auth } from '$lib/auth.svelte';
	import { getStagePresets } from '$lib/api';
	import type { GrowView, StagePresets } from '$lib/types';
	import { titleCase } from '$lib/format';
	import GrowFormModal from '$lib/components/GrowFormModal.svelte';
	import Sprout from '@lucide/svelte/icons/sprout';
	import MapPin from '@lucide/svelte/icons/map-pin';
	import Plus from '@lucide/svelte/icons/plus';

	const snap = $derived(live.snapshot);
	const grows = $derived(snap?.grows ?? []);
	const active = $derived(grows.filter((g) => g.status === 'active'));
	const inactive = $derived(grows.filter((g) => g.status !== 'active'));

	let presets = $state<StagePresets>({});
	let creating = $state(false);

	onMount(() => {
		getStagePresets().then((p) => (presets = p)).catch(() => {});
	});
</script>

<div class="mb-6 flex items-center justify-between">
	<div>
		<h1 class="text-2xl font-semibold">Grows</h1>
		<p class="text-sm text-rig-400">Cultivation runs and the plants they track, across your environments.</p>
	</div>
	{#if auth.isAdmin}
		<button
			onclick={() => (creating = true)}
			class="inline-flex items-center gap-1.5 rounded-md bg-rig-500 px-4 py-2 text-sm font-medium text-rig-950 transition-colors hover:bg-rig-400"
		>
			<Plus size={15} /> New grow
		</button>
	{/if}
</div>

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
				class="rounded-md bg-rig-500 px-5 py-2 text-sm font-medium text-rig-950 transition-colors hover:bg-rig-400"
			>
				Start a grow
			</button>
		{/if}
	</div>
{:else}
	<div class="space-y-8">
		{#snippet growRow(g: GrowView)}
			<a
				href="/grows/{g.id}"
				class="block rounded-xl border border-rig-800 bg-rig-900/50 p-4 transition-colors hover:border-rig-600"
			>
				<div class="mb-2 flex items-center justify-between gap-2">
					<h3 class="font-semibold">{g.name}</h3>
					<span class="rounded-full bg-rig-800 px-2 py-0.5 text-xs capitalize {g.status === 'active' ? 'text-leaf' : 'text-rig-400'}">
						{g.status === 'active' ? g.stage || '—' : titleCase(g.status)}
					</span>
				</div>
				<div class="flex items-center justify-between text-sm text-rig-400">
					<span>{titleCase(g.species) || 'No species set'}</span>
					<span class="tabular-nums">day {g.totalDays}</span>
				</div>
				<div class="mt-2 flex flex-wrap items-center gap-x-3 gap-y-1 text-xs text-rig-500">
					<span class="inline-flex items-center gap-1"><Sprout size={12} /> {g.plantCount} plants</span>
					<span>·</span>
					<span>{g.stageDays}d in {g.stage}</span>
					{#if g.environments.length}
						<span>·</span>
						<span class="inline-flex items-center gap-1"><MapPin size={11} /> {g.environments.map((e) => e.name).join(', ')}</span>
					{/if}
				</div>
			</a>
		{/snippet}

		<section>
			<h2 class="mb-3 text-sm font-semibold uppercase tracking-wide text-leaf">Active</h2>
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

{#if auth.isAdmin}
	<GrowFormModal bind:open={creating} {presets} />
{/if}
