<script lang="ts">
	import { onMount } from 'svelte';
	import { errMsg } from '$lib/errors';
	import type { Grow, Species, StagePresets } from '$lib/types';
	import { createGrow, updateGrow, getSpecies } from '$lib/api';
	import { Button, Dialog, Select, DatePicker } from '$lib/components/ui';
	import { titleCase } from '$lib/format';

	// Crop-neutral fallbacks, mirroring Grow Core's species defaults, used until
	// the species catalog loads or when a species curates no list of its own.
	const FALLBACK_MEDIA = ['soil', 'coco', 'soilless', 'hydroponic', 'aeroponic'];
	const FALLBACK_NUTRIENTS = ['organic', 'mineral', 'living-soil'];
	const POT_TYPES = ['fabric', 'plastic', 'terracotta', 'air-pot', 'other'];

	interface Props {
		open?: boolean;
		/** Provided in edit mode; omit to create. */
		grow?: Grow;
		presets: StagePresets;
		onSaved?: (grow: Grow) => void;
	}
	let { open = $bindable(false), grow, presets, onSaved }: Props = $props();

	let name = $state('');
	let species = $state('');
	let startDate = $state(new Date().toISOString().slice(0, 10));
	let notes = $state('');
	let busy = $state(false);
	let err = $state('');

	// Growing setup.
	let medium = $state('');
	let mediumDetails = $state('');
	let nutrientMethod = $state('');
	let potSize = $state<number | null>(null);
	let potUnit = $state('L');
	let potType = $state('');

	let catalog = $state<Species[]>([]);
	onMount(() => {
		getSpecies().then((s) => (catalog = s)).catch(() => {});
	});

	// Reseed on open.
	$effect(() => {
		if (!open) return;
		name = grow?.name ?? '';
		species = grow?.species ?? '';
		startDate = (grow?.startedAt ?? new Date().toISOString()).slice(0, 10);
		notes = grow?.notes ?? '';
		const s = grow?.setup;
		medium = s?.medium ?? '';
		mediumDetails = s?.mediumDetails ?? '';
		nutrientMethod = s?.nutrientMethod ?? '';
		potSize = s?.potSize ?? null;
		potUnit = s?.potUnit || 'L';
		potType = s?.potType ?? '';
	});

	const presetKeys = $derived(Object.keys(presets));
	// Stages are derived from the chosen species, not entered by hand.
	const derivedStages = $derived(presets[species.trim().toLowerCase()] ?? []);
	const canSave = $derived(!!name.trim() && derivedStages.length > 0);

	// The chosen species curates the media / nutrient lists; fall back to the
	// crop-neutral defaults until the catalog loads or when a species curates none.
	const selectedSpecies = $derived(catalog.find((s) => s.id === species.trim().toLowerCase()));
	const mediaOptions = $derived(selectedSpecies?.media?.options?.length ? selectedSpecies.media!.options : FALLBACK_MEDIA);
	const nutrientOptions = $derived(
		selectedSpecies?.nutrientMethods?.length ? selectedSpecies.nutrientMethods : FALLBACK_NUTRIENTS
	);
	const asItems = (vals: string[]) => vals.map((v) => ({ value: v, label: titleCase(v) }));

	async function save() {
		if (!canSave) return;
		busy = true;
		err = '';
		try {
			const input = {
				name: name.trim(),
				species: species.trim().toLowerCase(),
				startedAt: startDate,
				notes,
				setup: {
					medium,
					mediumDetails: mediumDetails.trim(),
					nutrientMethod,
					potSize: potSize && potSize > 0 ? potSize : 0,
					potUnit: potSize && potSize > 0 ? potUnit : '',
					potType: potSize && potSize > 0 ? potType : ''
				}
			};
			const saved = grow ? await updateGrow(grow.id, input) : await createGrow(input);
			open = false;
			onSaved?.(saved);
		} catch (e) {
			err = errMsg(e, 'Save failed');
		} finally {
			busy = false;
		}
	}

	const field =
		'w-full rounded-md border border-rig-700 bg-rig-950 px-3 py-1.5 text-sm focus:border-leaf focus:outline-none';
</script>

<Dialog bind:open title={grow ? 'Edit grow' : 'New grow'} description="A crop-neutral cultivation run with a configurable stage sequence.">
	<div class="space-y-4">
		{#if err}<p class="text-xs text-danger">{err}</p>{/if}
		<label class="block">
			<span class="text-xs text-rig-400">Name</span>
			<input bind:value={name} placeholder="e.g. Summer basil" class="{field} mt-1" />
		</label>
		<div class="grid gap-3 sm:grid-cols-2">
			<label class="block">
				<span class="text-xs text-rig-400">Species</span>
				<Select
					class="mt-1"
					bind:value={species}
					placeholder="Select a species…"
					items={presetKeys.map((k) => ({ value: k, label: titleCase(k) }))}
				/>
			</label>
			<label class="block">
				<span class="text-xs text-rig-400">Start date</span>
				<DatePicker class="mt-1" bind:value={startDate} />
			</label>
		</div>
		<div class="block">
			<span class="text-xs text-rig-400">Stages</span>
			{#if derivedStages.length}
				<div class="mt-1 flex flex-wrap gap-1.5">
					{#each derivedStages as st, i (st)}
						<span class="rounded-full bg-rig-800 px-2.5 py-0.5 text-xs capitalize text-rig-300">{i + 1}. {st}</span>
					{/each}
				</div>
				<span class="mt-1.5 block text-xs text-rig-500">Set automatically from the species — the first stage is the starting stage.</span>
			{:else}
				<p class="mt-1 text-xs text-rig-500">Pick a species to set the stage sequence.</p>
			{/if}
		</div>
		<div class="space-y-3 border-t border-rig-800 pt-4">
			<div class="text-xs font-medium uppercase tracking-wide text-rig-400">Growing setup</div>
			<div class="grid gap-3 sm:grid-cols-2">
				<label class="block">
					<span class="text-xs text-rig-400">Growing medium</span>
					<Select class="mt-1" bind:value={medium} placeholder="Select a medium…" items={asItems(mediaOptions)} />
				</label>
				<label class="block">
					<span class="text-xs text-rig-400">Nutrient method <span class="text-rig-600">(optional)</span></span>
					<Select class="mt-1" bind:value={nutrientMethod} placeholder="—" items={asItems(nutrientOptions)} />
				</label>
			</div>
			<label class="block">
				<span class="text-xs text-rig-400">Medium details <span class="text-rig-600">(optional)</span></span>
				<input bind:value={mediumDetails} placeholder="e.g. BioBizz Light-Mix + worm castings + perlite" class="{field} mt-1" />
			</label>
			<div class="grid gap-3 sm:grid-cols-3">
				<label class="block">
					<span class="text-xs text-rig-400">Default pot size <span class="text-rig-600">(optional)</span></span>
					<input type="number" min="0" step="any" bind:value={potSize} placeholder="e.g. 20" class="{field} mt-1" />
				</label>
				<label class="block">
					<span class="text-xs text-rig-400">Unit</span>
					<Select class="mt-1" bind:value={potUnit} items={[{ value: 'L', label: 'liters (L)' }, { value: 'gal', label: 'gallons' }]} />
				</label>
				<label class="block">
					<span class="text-xs text-rig-400">Pot type</span>
					<Select class="mt-1" bind:value={potType} placeholder="—" items={asItems(POT_TYPES)} />
				</label>
			</div>
			<p class="text-xs text-rig-500">The default container seeds each new plant's pot. Irrigation is configured on the environment.</p>
		</div>

		<label class="block">
			<span class="text-xs text-rig-400">Notes</span>
			<textarea bind:value={notes} rows="2" class="{field} mt-1"></textarea>
		</label>
		<div class="flex justify-end gap-2 border-t border-rig-800 pt-4">
			<Button variant="ghost" onclick={() => (open = false)}>Cancel</Button>
			<Button onclick={save} disabled={busy || !canSave}>Save</Button>
		</div>
	</div>
</Dialog>
