<script lang="ts">
	import { errMsg } from '$lib/errors';
	import { untrack } from 'svelte';
	import type { Grow, LightSchedule, LightScheduleMode, StageLightDefaults } from '$lib/types';
	import { setControlGrow, setSchedule } from '$lib/api';
	import { titleCase } from '$lib/format';
	import { Button, Dialog, Select, type SelectItem } from '$lib/components/ui';

	interface Props {
		open?: boolean;
		environmentId: string;
		controlGrowId: string;
		schedule?: LightSchedule;
		grows: Grow[];
		defaults: StageLightDefaults;
		hasPrimaryLight: boolean;
		onSaved?: () => void;
	}
	let {
		open = $bindable(false),
		environmentId,
		controlGrowId,
		schedule,
		grows,
		defaults,
		hasPrimaryLight,
		onSaved
	}: Props = $props();

	// --- control grow ---
	let growId = $state('');

	// --- lighting fields ---
	let mode = $state<LightScheduleMode>('off');
	let lightsOnAt = $state('06:00');
	let customHours = $state(18);
	// Per-stage hours for phase mode, seeded from overrides or recommended defaults.
	let stageHours = $state<Record<string, number>>({});

	let busy = $state(false);
	let err = $state('');

	// The active grows selectable as control grow.
	const activeGrows = $derived(grows.filter((g) => g.status === 'active'));
	// Stage list driving the per-stage editor: the selected grow's stages, else
	// the known default stage names.
	const selectedGrow = $derived(grows.find((g) => g.id === growId));
	const stages = $derived(selectedGrow?.stages?.length ? selectedGrow.stages : Object.keys(defaults));

	// Reseed the whole form whenever the modal opens. This must depend ONLY on
	// `open`: the body writes `growId`, and `stages` derives from `growId`, so
	// reading either here would re-run the effect and clobber the user's choice.
	$effect(() => {
		if (!open) return;
		untrack(() => {
			growId = controlGrowId ?? '';
			mode = schedule?.mode ?? 'off';
			lightsOnAt = schedule?.lightsOnAt || '06:00';
			customHours = schedule?.onHours ?? 18;
			// Seed from the control grow's stages (not the reactive `stages`).
			const seedGrow = grows.find((g) => g.id === (controlGrowId ?? ''));
			const seedStages = seedGrow?.stages?.length ? seedGrow.stages : Object.keys(defaults);
			const seeded: Record<string, number> = {};
			for (const st of seedStages) seeded[st] = schedule?.stageOnHours?.[st] ?? defaults[st] ?? 18;
			stageHours = seeded;
		});
	});

	// Keep the per-stage editor populated when the user switches control grow: any
	// stage missing an entry gets its recommended default. Never overwrites edits.
	$effect(() => {
		const current = stages;
		untrack(() => {
			for (const st of current) {
				if (stageHours[st] === undefined) stageHours[st] = defaults[st] ?? 18;
			}
		});
	});

	const growItems: SelectItem[] = $derived([
		{ value: '__none__', label: 'None (manual)' },
		...activeGrows.map((g) => ({ value: g.id, label: g.name }))
	]);
	const modeItems: SelectItem[] = [
		{ value: 'off', label: 'Manual only' },
		{ value: 'phase', label: 'Follow stage' },
		{ value: 'custom', label: 'Custom' }
	];

	function offHours(on: number): string {
		return `${on}/${Math.max(0, 24 - on)}`;
	}

	async function save() {
		busy = true;
		err = '';
		try {
			if (growId !== (controlGrowId ?? '')) {
				await setControlGrow(environmentId, growId);
			}
			const stageOnHours: Record<string, number> = {};
			if (mode === 'phase') {
				for (const st of stages) {
					// Persist an override only when it differs from the recommendation.
					if (stageHours[st] !== (defaults[st] ?? 18)) stageOnHours[st] = stageHours[st];
				}
			}
			await setSchedule(environmentId, { mode, lightsOnAt, onHours: customHours, stageOnHours });
			open = false;
			onSaved?.();
		} catch (e) {
			err = errMsg(e, 'Save failed');
		} finally {
			busy = false;
		}
	}

	const field =
		'w-full rounded-md border border-rig-700 bg-rig-950 px-3 py-1.5 text-sm focus:border-rig-500 focus:outline-none';
</script>

<Dialog bind:open title="Control grow & lighting" description="Choose the grow whose stage drives this tent, and the light photoperiod.">
	<div class="space-y-6">
		{#if err}<p class="text-xs text-danger">{err}</p>{/if}

		<!-- Control grow -->
		<section class="space-y-3">
			<h3 class="text-xs font-semibold uppercase tracking-wide text-rig-400">Control grow</h3>
			<p class="text-xs text-rig-500">
				When several grows share this tent, the control grow's current stage supplies the automation
				presets. Choose "None" to keep lighting on defaults.
			</p>
			<Select
				value={growId || '__none__'}
				onValueChange={(v) => (growId = v === '__none__' ? '' : v)}
				items={growItems}
			/>
			{#if activeGrows.length === 0}
				<p class="rounded-md border border-rig-700 bg-rig-900/40 px-3 py-2 text-xs text-rig-400">
					No active grows yet. Create one under <a href="/grows" class="text-leaf hover:underline">Grows</a>.
				</p>
			{/if}
		</section>

		<!-- Lighting -->
		<section class="space-y-3 border-t border-rig-800 pt-4">
			<h3 class="text-xs font-semibold uppercase tracking-wide text-rig-400">Lighting (photoperiod)</h3>
			{#if !hasPrimaryLight}
				<p class="rounded-md border border-warn/30 bg-warn/10 px-3 py-2 text-xs text-warn">
					No primary grow light is set for this tent. Mark a light as primary in Settings &amp; Devices for the schedule to control it.
				</p>
			{/if}
			<div class="grid gap-3 sm:grid-cols-2">
				<label class="block">
					<span class="text-xs text-rig-400">Mode</span>
					<Select value={mode} onValueChange={(v) => (mode = v as LightScheduleMode)} items={modeItems} class="mt-1" />
				</label>
				{#if mode !== 'off'}
					<label class="block">
						<span class="text-xs text-rig-400">Lights on at</span>
						<input type="time" bind:value={lightsOnAt} class="{field} mt-1" />
					</label>
				{/if}
			</div>

			{#if mode === 'custom'}
				<label class="block">
					<span class="text-xs text-rig-400">Hours of light ({offHours(customHours)})</span>
					<input type="number" min="0" max="24" step="0.5" bind:value={customHours} class="{field} mt-1" />
				</label>
			{:else if mode === 'phase'}
				<div class="space-y-1.5">
					<p class="text-xs text-rig-500">Hours of light per stage. Defaults are recommendations — override any of them.</p>
					{#each stages as st (st)}
						<div class="flex items-center gap-3">
							<span class="w-24 text-sm capitalize text-rig-300">{titleCase(st)}</span>
							<input
								type="number"
								min="0"
								max="24"
								step="0.5"
								bind:value={stageHours[st]}
								class="w-20 rounded-md border border-rig-700 bg-rig-950 px-2 py-1 text-sm focus:border-rig-500 focus:outline-none"
							/>
							<span class="text-xs text-rig-500">
								{offHours(stageHours[st] ?? 0)}
								{#if stageHours[st] === (defaults[st] ?? 18)}· recommended{/if}
							</span>
						</div>
					{/each}
				</div>
			{/if}
		</section>

		<div class="flex justify-end gap-2 border-t border-rig-800 pt-4">
			<Button variant="ghost" onclick={() => (open = false)}>Cancel</Button>
			<Button onclick={save} disabled={busy}>Save</Button>
		</div>
	</div>
</Dialog>
