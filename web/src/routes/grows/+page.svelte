<script lang="ts">
	import { onMount } from 'svelte';
	import { live } from '$lib/live.svelte';
	import { auth } from '$lib/auth.svelte';
	import {
		getStagePresets,
		getSpecies,
		getCultivars,
		deleteCultivar,
		cultivarImageURL,
		getFeedingPresets,
		deleteFeedingPreset
	} from '$lib/api';
	import type { GrowView, StagePresets, Species, Cultivar, FeedingPreset } from '$lib/types';
	import { titleCase } from '$lib/format';
	import GrowFormModal from '$lib/components/GrowFormModal.svelte';
	import CultivarFormModal from '$lib/components/CultivarFormModal.svelte';
	import FeedingPresetFormModal from '$lib/components/FeedingPresetFormModal.svelte';
	import GrowCard from '$lib/components/GrowCard.svelte';
	import Sprout from '@lucide/svelte/icons/sprout';
	import Plus from '@lucide/svelte/icons/plus';
	import Dna from '@lucide/svelte/icons/dna';
	import Pencil from '@lucide/svelte/icons/pencil';
	import Trash2 from '@lucide/svelte/icons/trash-2';
	import Copy from '@lucide/svelte/icons/copy';
	import FlaskConical from '@lucide/svelte/icons/flask-conical';

	const snap = $derived(live.snapshot);
	const grows = $derived(snap?.grows ?? []);
	const active = $derived(grows.filter((g) => g.status === 'active'));
	const inactive = $derived(grows.filter((g) => g.status !== 'active'));

	let presets = $state<StagePresets>({});
	let creating = $state(false);

	// Cultivars are reference data (not in the live snapshot), fetched over REST.
	let species = $state<Species[]>([]);
	let cultivars = $state<Cultivar[]>([]);
	let editingCultivar = $state<Cultivar | undefined>(undefined);
	let cultivarModalOpen = $state(false);

	// Feeding presets are reference data too (built-in + user), fetched over REST.
	let feedings = $state<FeedingPreset[]>([]);
	let editingFeeding = $state<FeedingPreset | undefined>(undefined);
	let feedingModalOpen = $state(false);

	const speciesById = $derived(new Map(species.map((s) => [s.id, s])));

	onMount(() => {
		getStagePresets().then((p) => (presets = p)).catch(() => {});
		getSpecies().then((s) => (species = s)).catch(() => {});
		refreshCultivars();
		refreshFeedings();
	});

	function refreshCultivars() {
		getCultivars().then((c) => (cultivars = c)).catch(() => {});
	}

	function refreshFeedings() {
		getFeedingPresets().then((f) => (feedings = f)).catch(() => {});
	}

	function newFeeding() {
		editingFeeding = undefined;
		feedingModalOpen = true;
	}
	// Edit a user preset, or duplicate a built-in one (the modal handles both).
	function editFeeding(f: FeedingPreset) {
		editingFeeding = f;
		feedingModalOpen = true;
	}
	async function removeFeeding(f: FeedingPreset) {
		if (!confirm(`Delete feeding preset “${f.name}”?`)) return;
		try {
			await deleteFeedingPreset(f.id);
			refreshFeedings();
		} catch {
			/* ignore */
		}
	}

	// Total week count across a preset's phases, for the card summary.
	function weekCount(f: FeedingPreset): number {
		return (f.phases ?? []).reduce((n, ph) => n + (ph.weeks?.length ?? 0), 0);
	}

	function newCultivar() {
		editingCultivar = undefined;
		cultivarModalOpen = true;
	}
	function editCultivar(c: Cultivar) {
		editingCultivar = c;
		cultivarModalOpen = true;
	}
	async function removeCultivar(c: Cultivar) {
		if (!confirm(`Delete cultivar “${c.name}”?`)) return;
		try {
			await deleteCultivar(c.id);
			refreshCultivars();
		} catch {
			/* ignore */
		}
	}

	// A short, human summary of a cultivar's attributes using its species schema.
	function attrSummary(c: Cultivar): { label: string; value: string }[] {
		const sp = speciesById.get(c.species);
		if (!sp?.cultivarAttributes) return [];
		const out: { label: string; value: string }[] = [];
		for (const attr of sp.cultivarAttributes) {
			const v = c.attributes?.[attr.key];
			if (!v) continue;
			const value = attr.type === 'percent' ? `${v}%` : attr.unit ? `${v} ${attr.unit}` : titleCase(v);
			out.push({ label: attr.label, value });
		}
		return out;
	}
</script>

<div class="mb-6 flex items-start justify-between gap-4">
	<div>
		<h1 class="text-2xl font-semibold">Grows</h1>
		<p class="text-sm text-rig-400">Cultivation runs, the plants they track, and your cultivar library.</p>
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
					class="rounded-md bg-rig-500 px-5 py-2 text-sm font-medium text-rig-950 transition-colors hover:bg-rig-400"
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

	<!-- Cultivars: a library that lives under the grows on the same page. -->
	<section>
		<div class="mb-3 flex items-center justify-between gap-4">
			<h2 class="text-sm font-semibold uppercase tracking-wide text-leaf">
				Cultivars{cultivars.length ? ` · ${cultivars.length}` : ''}
			</h2>
			{#if auth.isAdmin && cultivars.length}
				<button
					onclick={newCultivar}
					class="inline-flex items-center gap-1.5 rounded-md border border-rig-700 px-3 py-1.5 text-xs font-medium text-rig-200 transition-colors hover:border-rig-500 hover:text-white"
				>
					<Plus size={14} /> New cultivar
				</button>
			{/if}
		</div>
		{#if cultivars.length === 0}
			<div class="rounded-xl border border-dashed border-rig-800 p-10 text-center">
				<div class="mb-3 flex justify-center text-rig-500"><Dna size={40} /></div>
				<h3 class="mb-1 text-lg font-semibold">No cultivars yet</h3>
				<p class="mb-5 text-sm text-rig-400">Build a library of strains and varieties, then bind them to your plants.</p>
				{#if auth.isAdmin}
					<button
						onclick={newCultivar}
						class="rounded-md bg-rig-500 px-5 py-2 text-sm font-medium text-rig-950 transition-colors hover:bg-rig-400"
					>
						Add a cultivar
					</button>
				{/if}
			</div>
		{:else}
			<div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
			{#each cultivars as c (c.id)}
				<div class="group relative flex flex-col overflow-hidden rounded-xl border border-rig-800 bg-rig-900/50 transition-colors hover:border-rig-600">
					<!-- Image occupies the top half, shown in full (no crop). -->
					<div class="flex h-48 items-center justify-center overflow-hidden border-b border-rig-800 bg-rig-950">
						{#if c.imageType}
							<img src={cultivarImageURL(c.id)} alt={c.name} class="max-h-full max-w-full object-contain" />
						{:else}
							<div class="text-rig-700"><Dna size={40} /></div>
						{/if}
					</div>
					<div class="min-w-0 flex-1 p-3">
						<div class="flex items-center justify-between gap-2">
							<h3 class="truncate font-semibold">{c.name}</h3>
							<span class="shrink-0 rounded-full bg-rig-800 px-2 py-0.5 text-[11px] capitalize text-rig-300">
								{speciesById.get(c.species)?.label ?? c.species}
							</span>
						</div>
						{#if c.creator}<p class="truncate text-xs text-rig-500">by {c.creator}</p>{/if}
						{#if attrSummary(c).length}
							<div class="mt-1.5 flex flex-wrap gap-1">
								{#each attrSummary(c) as a (a.label)}
									<span class="rounded bg-rig-800/70 px-1.5 py-0.5 text-[11px] text-rig-300">
										<span class="text-rig-500">{a.label}:</span> {a.value}
									</span>
								{/each}
							</div>
						{/if}
						{#if c.description}<p class="mt-1.5 line-clamp-2 text-xs text-rig-400">{c.description}</p>{/if}
					</div>
					{#if auth.isAdmin}
						<div class="absolute right-2 top-2 flex gap-1 opacity-0 transition-opacity group-hover:opacity-100">
							<button
								onclick={() => editCultivar(c)}
								aria-label="Edit cultivar"
								class="rounded bg-rig-950/80 p-1.5 text-rig-400 hover:text-rig-100"
							>
								<Pencil size={13} />
							</button>
							<button
								onclick={() => removeCultivar(c)}
								aria-label="Delete cultivar"
								class="rounded bg-rig-950/80 p-1.5 text-rig-400 hover:text-danger"
							>
								<Trash2 size={13} />
							</button>
						</div>
					{/if}
				</div>
			{/each}
		</div>
		{/if}
	</section>

	<!-- Feeding presets: nutrient schedules (built-in + user), below cultivars. -->
	<section>
		<div class="mb-3 flex items-center justify-between gap-4">
			<h2 class="text-sm font-semibold uppercase tracking-wide text-leaf">
				Feeding presets{feedings.length ? ` · ${feedings.length}` : ''}
			</h2>
			{#if auth.isAdmin && feedings.length}
				<button
					onclick={newFeeding}
					class="inline-flex items-center gap-1.5 rounded-md border border-rig-700 px-3 py-1.5 text-xs font-medium text-rig-200 transition-colors hover:border-rig-500 hover:text-white"
				>
					<Plus size={14} /> New preset
				</button>
			{/if}
		</div>
		{#if feedings.length === 0}
			<div class="rounded-xl border border-dashed border-rig-800 p-10 text-center">
				<div class="mb-3 flex justify-center text-rig-500"><FlaskConical size={40} /></div>
				<h3 class="mb-1 text-lg font-semibold">No feeding presets yet</h3>
				<p class="mb-5 text-sm text-rig-400">Build nutrient schedules — products dosed per week across each phase of a grow.</p>
				{#if auth.isAdmin}
					<button
						onclick={newFeeding}
						class="rounded-md bg-rig-500 px-5 py-2 text-sm font-medium text-rig-950 transition-colors hover:bg-rig-400"
					>
						Add a preset
					</button>
				{/if}
			</div>
		{:else}
			<div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
				{#each feedings as f (f.id)}
					<div class="group relative flex flex-col overflow-hidden rounded-xl border border-rig-800 bg-rig-900/50 p-4 transition-colors hover:border-rig-600">
						<div class="flex items-start justify-between gap-2">
							<div class="min-w-0">
								<h3 class="truncate font-semibold">{f.name}</h3>
								{#if f.brand}<p class="truncate text-xs text-rig-500">{f.brand}</p>{/if}
							</div>
							<div class="flex shrink-0 items-center gap-1">
								{#if f.source === 'builtin'}
									<span class="rounded-full bg-rig-800 px-2 py-0.5 text-[11px] text-rig-300">Built-in</span>
								{/if}
								<span class="rounded-full bg-rig-800 px-2 py-0.5 text-[11px] capitalize text-rig-300">
									{speciesById.get(f.species)?.label ?? f.species}
								</span>
							</div>
						</div>
						<div class="mt-2 flex flex-wrap gap-1">
							<span class="rounded bg-rig-800/70 px-1.5 py-0.5 text-[11px] text-rig-300">
								<span class="text-rig-500">Products:</span> {f.products?.length ?? 0}
							</span>
							<span class="rounded bg-rig-800/70 px-1.5 py-0.5 text-[11px] text-rig-300">
								<span class="text-rig-500">Phases:</span> {f.phases?.length ?? 0}
							</span>
							<span class="rounded bg-rig-800/70 px-1.5 py-0.5 text-[11px] text-rig-300">
								<span class="text-rig-500">Weeks:</span> {weekCount(f)}
							</span>
						</div>
						{#if f.description}<p class="mt-2 line-clamp-2 text-xs text-rig-400">{f.description}</p>{/if}
						{#if (f.phases ?? []).length}
							<div class="mt-2 flex flex-wrap gap-1">
								{#each f.phases as ph (ph.name)}
									<span class="rounded bg-rig-950 px-1.5 py-0.5 text-[10px] text-rig-400">
										{ph.name}{ph.weeks?.length ? ` ·${ph.weeks.length}w` : ''}
									</span>
								{/each}
							</div>
						{/if}
						{#if auth.isAdmin}
							<div class="absolute right-2 top-2 flex gap-1 opacity-0 transition-opacity group-hover:opacity-100">
								<button
									onclick={() => editFeeding(f)}
									aria-label={f.source === 'builtin' ? 'Duplicate preset' : 'Edit preset'}
									class="rounded bg-rig-950/80 p-1.5 text-rig-400 hover:text-rig-100"
								>
									{#if f.source === 'builtin'}<Copy size={13} />{:else}<Pencil size={13} />{/if}
								</button>
								{#if f.source === 'user'}
									<button
										onclick={() => removeFeeding(f)}
										aria-label="Delete preset"
										class="rounded bg-rig-950/80 p-1.5 text-rig-400 hover:text-danger"
									>
										<Trash2 size={13} />
									</button>
								{/if}
							</div>
						{/if}
					</div>
				{/each}
			</div>
		{/if}
	</section>
</div>

{#if auth.isAdmin}
	<GrowFormModal bind:open={creating} {presets} />
	<CultivarFormModal
		bind:open={cultivarModalOpen}
		cultivar={editingCultivar}
		{species}
		onSaved={refreshCultivars}
	/>
	<FeedingPresetFormModal
		bind:open={feedingModalOpen}
		preset={editingFeeding}
		{species}
		onSaved={refreshFeedings}
	/>
{/if}
