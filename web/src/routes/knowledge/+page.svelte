<script lang="ts">
	import { onMount } from 'svelte';
	import { auth } from '$lib/auth.svelte';
	import {
		getSpecies,
		getCultivars,
		deleteCultivar,
		cultivarImageURL,
		getFeedingPresets,
		getFeedingTemplates,
		deleteFeedingPreset
	} from '$lib/api';
	import type { Species, Cultivar, FeedingPreset } from '$lib/types';
	import { titleCase } from '$lib/format';
	import CultivarFormModal from '$lib/components/CultivarFormModal.svelte';
	import FeedingPresetFormModal from '$lib/components/FeedingPresetFormModal.svelte';
	import Plus from '@lucide/svelte/icons/plus';
	import Dna from '@lucide/svelte/icons/dna';
	import Pencil from '@lucide/svelte/icons/pencil';
	import Trash2 from '@lucide/svelte/icons/trash-2';
	import FlaskConical from '@lucide/svelte/icons/flask-conical';

	let species = $state<Species[]>([]);
	let cultivars = $state<Cultivar[]>([]);
	let editingCultivar = $state<Cultivar | undefined>(undefined);
	let cultivarModalOpen = $state(false);

	// Feeding presets: the user's own (shown in the table) plus built-in
	// templates (offered only inside the create form).
	let feedings = $state<FeedingPreset[]>([]);
	let feedingTemplates = $state<FeedingPreset[]>([]);
	let editingFeeding = $state<FeedingPreset | undefined>(undefined);
	let feedingModalOpen = $state(false);

	const speciesById = $derived(new Map(species.map((s) => [s.id, s])));

	onMount(() => {
		getSpecies().then((s) => (species = s)).catch(() => {});
		refreshCultivars();
		refreshFeedings();
		getFeedingTemplates().then((t) => (feedingTemplates = t)).catch(() => {});
	});

	function refreshCultivars() {
		getCultivars().then((c) => (cultivars = c)).catch(() => {});
	}

	// Enter/Space activates a clickable cultivar card / preset row.
	function activateOnKey(e: KeyboardEvent, fn: () => void) {
		if (e.key === 'Enter' || e.key === ' ') {
			e.preventDefault();
			fn();
		}
	}

	function refreshFeedings() {
		getFeedingPresets().then((f) => (feedings = f)).catch(() => {});
	}

	function newFeeding() {
		editingFeeding = undefined;
		feedingModalOpen = true;
	}
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

<div class="mb-6">
	<h1 class="text-2xl font-semibold">Knowledge</h1>
	<p class="text-sm text-rig-400">Feeding schedules, cultivar library, and other reference data for your grows.</p>
</div>

<div class="space-y-10">
	<!-- Feeding presets: nutrient schedules (built-in + user). -->
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
			<div class="overflow-x-auto rounded-xl border border-rig-800">
				<table class="w-full border-collapse text-sm">
					<thead>
						<tr class="border-b border-rig-800 bg-rig-900/50 text-left text-xs uppercase tracking-wide text-rig-500">
							<th class="px-4 py-2.5 font-medium">Name</th>
							<th class="px-4 py-2.5 font-medium">Brand</th>
							<th class="px-4 py-2.5 font-medium">Species</th>
							<th class="px-4 py-2.5 text-center font-medium">Products</th>
							<th class="px-4 py-2.5 text-center font-medium">Phases</th>
							<th class="px-4 py-2.5 text-center font-medium">Weeks</th>
							{#if auth.isAdmin}<th class="px-4 py-2.5"></th>{/if}
						</tr>
					</thead>
					<tbody>
						{#each feedings as f (f.id)}
							<!-- Row click opens the editor for admins; the Edit button keeps keyboard access. -->
							<!-- svelte-ignore a11y_click_events_have_key_events a11y_no_static_element_interactions -->
							<tr
								class="border-b border-rig-800/60 last:border-0 hover:bg-rig-900/30"
								class:cursor-pointer={auth.isAdmin}
								onclick={() => auth.isAdmin && editFeeding(f)}
							>
								<td class="px-4 py-2.5">
									<div class="font-medium text-rig-100">{f.name}</div>
									{#if f.description}<div class="line-clamp-1 max-w-md text-xs text-rig-500">{f.description}</div>{/if}
								</td>
								<td class="px-4 py-2.5 text-rig-300">{f.brand || '—'}</td>
								<td class="px-4 py-2.5">
									<span class="rounded-full bg-rig-800 px-2 py-0.5 text-[11px] capitalize text-rig-300">
										{speciesById.get(f.species)?.label ?? f.species}
									</span>
								</td>
								<td class="px-4 py-2.5 text-center text-rig-300">{f.products?.length ?? 0}</td>
								<td class="px-4 py-2.5 text-center text-rig-300">{f.phases?.length ?? 0}</td>
								<td class="px-4 py-2.5 text-center text-rig-300">{weekCount(f)}</td>
								{#if auth.isAdmin}
									<td class="px-4 py-2.5">
										<div class="flex justify-end gap-1">
											<button
												onclick={(e) => { e.stopPropagation(); editFeeding(f); }}
												aria-label="Edit preset"
												class="rounded p-1.5 text-rig-400 hover:text-rig-100"
											>
												<Pencil size={14} />
											</button>
											<button
												onclick={(e) => { e.stopPropagation(); removeFeeding(f); }}
												aria-label="Delete preset"
												class="rounded p-1.5 text-rig-400 hover:text-danger"
											>
												<Trash2 size={14} />
											</button>
										</div>
									</td>
								{/if}
							</tr>
						{/each}
					</tbody>
				</table>
			</div>
		{/if}
	</section>

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
				<!-- Whole card opens the editor for admins (buttons below stop propagation). -->
				<div
					class="group relative flex flex-col overflow-hidden rounded-xl border border-rig-800 bg-rig-900/50 transition-colors hover:border-rig-600"
					class:cursor-pointer={auth.isAdmin}
					role="button"
					tabindex={auth.isAdmin ? 0 : -1}
					onclick={() => auth.isAdmin && editCultivar(c)}
					onkeydown={(e) => auth.isAdmin && activateOnKey(e, () => editCultivar(c))}
				>
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
								onclick={(e) => { e.stopPropagation(); editCultivar(c); }}
								aria-label="Edit cultivar"
								class="rounded bg-rig-950/80 p-1.5 text-rig-400 hover:text-rig-100"
							>
								<Pencil size={13} />
							</button>
							<button
								onclick={(e) => { e.stopPropagation(); removeCultivar(c); }}
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
</div>

{#if auth.isAdmin}
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
		templates={feedingTemplates}
		onSaved={refreshFeedings}
	/>
{/if}
