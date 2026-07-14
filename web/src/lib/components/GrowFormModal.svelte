<script lang="ts">
	import { errMsg } from '$lib/errors';
	import type { Grow, StagePresets } from '$lib/types';
	import { createGrow, updateGrow } from '$lib/api';
	import { Button, Dialog, Select, DatePicker } from '$lib/components/ui';
	import { titleCase } from '$lib/format';

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

	// Reseed on open.
	$effect(() => {
		if (!open) return;
		name = grow?.name ?? '';
		species = grow?.species ?? '';
		startDate = (grow?.startedAt ?? new Date().toISOString()).slice(0, 10);
		notes = grow?.notes ?? '';
	});

	const presetKeys = $derived(Object.keys(presets));
	// Stages are derived from the chosen species, not entered by hand.
	const derivedStages = $derived(presets[species.trim().toLowerCase()] ?? []);
	const canSave = $derived(!!name.trim() && derivedStages.length > 0);

	async function save() {
		if (!canSave) return;
		busy = true;
		err = '';
		try {
			const input = {
				name: name.trim(),
				species: species.trim().toLowerCase(),
				startedAt: startDate,
				notes
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
		'w-full rounded-md border border-rig-700 bg-rig-950 px-3 py-1.5 text-sm focus:border-rig-500 focus:outline-none';
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
