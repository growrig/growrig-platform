<script lang="ts">
	import { onMount } from 'svelte';
	import { Dialog, Button, DatePicker, Switch } from '$lib/components/ui';
	import { updateStageDates, updateGrow, getSpecies } from '$lib/api';
	import type { GrowDetail, StageEvent, Species, SpeciesStage } from '$lib/types';
	import { errMsg } from '$lib/errors';
	import { stageColorAt } from '$lib/stageColor';
	import X from '@lucide/svelte/icons/x';

	interface Props {
		open?: boolean;
		grow: GrowDetail;
		events: StageEvent[];
		/** Called with the fresh event list after saving. */
		onSaved: (events: StageEvent[]) => void;
	}
	let { open = $bindable(false), grow, events, onSaved }: Props = $props();

	// The species' full stage catalog — which stages are optional and their
	// canonical order (so an added stage lands in the right position).
	let allSpecies = $state<Species[]>([]);
	onMount(() => {
		getSpecies().then((s) => (allSpecies = s)).catch(() => {});
	});
	const speciesStages = $derived<SpeciesStage[]>(
		allSpecies.find((s) => s.id === grow.species)?.stages ?? []
	);
	const canon = $derived<SpeciesStage[]>(
		speciesStages.length ? speciesStages : grow.stages.map((n) => ({ name: n, lightHours: 0 }))
	);
	const currentCanonIdx = $derived(canon.findIndex((s) => s.name === grow.stage));

	// Which stages the grow will run, and a `YYYY-MM-DD` per stage ('' = predicted).
	let selected = $state<string[]>([]);
	let dates = $state<Record<string, string>>({});
	let busy = $state(false);
	let err = $state('');

	/** Local `YYYY-MM-DD` for an ISO datetime string. */
	function isoDay(value: string): string {
		const d = new Date(value);
		if (Number.isNaN(d.getTime())) return '';
		const y = d.getFullYear();
		const m = String(d.getMonth() + 1).padStart(2, '0');
		const day = String(d.getDate()).padStart(2, '0');
		return `${y}-${m}-${day}`;
	}

	// Reseed each time the editor opens.
	let lastOpen = false;
	$effect(() => {
		if (open && !lastOpen) {
			selected = [...grow.stages];
			const byStage = new Map(events.map((e) => [e.stage, isoDay(e.enteredAt)]));
			dates = Object.fromEntries(grow.stages.map((s) => [s, byStage.get(s) ?? '']));
			err = '';
		}
		lastOpen = open;
	});

	// Rows: every stage the grow runs, plus inactive optional stages that sit in
	// the future (so they can be switched on). Colors follow the selected order,
	// matching the timeline heatmap.
	interface Row {
		name: string;
		optional: boolean;
		future: boolean;
		togglable: boolean;
		current: boolean;
	}
	const rows = $derived.by<Row[]>(() => {
		const inGrow = new Set(grow.stages);
		return canon
			.map((s, ci) => {
				const future = currentCanonIdx < 0 ? true : ci > currentCanonIdx;
				return {
					name: s.name,
					optional: !!s.optional,
					future,
					togglable: grow.status === 'active' && !!s.optional && future,
					current: s.name === grow.stage
				};
			})
			.filter((r) => inGrow.has(r.name) || r.togglable);
	});
	// Stage → color, by position in the selected sequence.
	const colorByStage = $derived(
		new Map(canon.map((s) => s.name).filter((n) => selected.includes(n)).map((n, i) => [n, stageColorAt(i)]))
	);

	function toggle(name: string, on: boolean) {
		selected = on ? [...selected, name] : selected.filter((s) => s !== name);
	}

	async function save() {
		busy = true;
		err = '';
		try {
			const order = canon.map((s) => s.name);
			const newStages = order.filter((n) => selected.includes(n));
			// Persist stage membership first, so any date targets a stage that exists.
			if (newStages.join(',') !== grow.stages.join(',')) {
				await updateGrow(grow.id, {
					name: grow.name,
					species: grow.species,
					startedAt: grow.startedAt,
					notes: grow.notes,
					stages: newStages,
					setup: grow.setup
				});
			}
			const keep = new Set(newStages);
			const dueDates = Object.fromEntries(Object.entries(dates).filter(([s]) => keep.has(s)));
			onSaved(await updateStageDates(grow.id, dueDates));
			open = false;
		} catch (e) {
			err = errMsg(e, 'Failed to update stages');
		} finally {
			busy = false;
		}
	}
</script>

<Dialog
	bind:open
	title="Edit stages"
	description="Choose which optional stages this grow runs, and set when it entered each. Stages it hasn't reached are predicted from typical durations — leave the date blank, or fill one in to override."
	size="xl"
>
	<div class="space-y-4">
		{#if err}<p class="rounded-md border border-danger/40 bg-danger/10 px-3 py-2 text-sm text-danger">{err}</p>{/if}

		<div class="space-y-2">
			{#each rows as r (r.name)}
				{@const on = selected.includes(r.name)}
				<div class="flex items-center gap-3 {on ? '' : 'opacity-60'}">
					{#if on}
						<span class="h-2.5 w-2.5 shrink-0 rounded-full" style="background:{colorByStage.get(r.name)}"></span>
					{:else}
						<span class="h-2.5 w-2.5 shrink-0 rounded-full border border-dashed border-rig-600"></span>
					{/if}
					<span class="flex w-28 shrink-0 items-center gap-1.5 text-sm capitalize {r.current ? 'font-medium text-rig-100' : on ? 'text-rig-200' : 'text-rig-400'}">
						{r.name}
						{#if r.current}<span class="rounded-full bg-leaf/20 px-1.5 py-0.5 text-[9px] uppercase tracking-wide text-leaf">now</span>{/if}
					</span>

					{#if on}
						<DatePicker value={dates[r.name] ?? ''} onValueChange={(v) => (dates[r.name] = v)} class="flex-1" />
						<button
							onclick={() => (dates[r.name] = '')}
							disabled={busy || !dates[r.name]}
							class="rounded-md p-2 text-rig-500 transition-colors hover:bg-rig-800 hover:text-rig-200 disabled:opacity-30"
							aria-label="Clear date"
						>
							<X size={15} />
						</button>
					{:else}
						<span class="flex-1 text-xs text-rig-600">Not in this grow</span>
					{/if}

					{#if r.togglable}
						<Switch checked={on} disabled={busy} onCheckedChange={(v) => toggle(r.name, v)} aria-label="Include {r.name} stage" />
					{:else}
						<span class="w-11 shrink-0 text-right text-[10px] uppercase tracking-wide text-rig-700">req</span>
					{/if}
				</div>
			{/each}
		</div>

		<p class="text-xs text-rig-500">Required stages are always included. Only upcoming stages can be added or removed — past and current stages are locked. Dates should follow the stage order; the current stage's date sets how long it has been running.</p>

		<div class="flex justify-end gap-2 border-t border-rig-800 pt-4">
			<Button variant="ghost" onclick={() => (open = false)} disabled={busy}>Cancel</Button>
			<Button onclick={save} disabled={busy}>Save</Button>
		</div>
	</div>
</Dialog>
