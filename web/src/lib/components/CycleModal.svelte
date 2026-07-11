<script lang="ts">
	import type { Cycle, LightSchedule, LightScheduleMode, Phase, PhotoperiodDefaults } from '$lib/types';
	import { setCycle, clearCycle, setSchedule } from '$lib/api';
	import { Button, Dialog, Select, type SelectItem } from '$lib/components/ui';

	interface Props {
		open?: boolean;
		environmentId: string;
		cycle?: Cycle;
		schedule?: LightSchedule;
		phases: Phase[];
		defaults: PhotoperiodDefaults;
		hasPrimaryLight: boolean;
		onSaved?: () => void;
	}
	let {
		open = $bindable(false),
		environmentId,
		cycle,
		schedule,
		phases,
		defaults,
		hasPrimaryLight,
		onSaved
	}: Props = $props();

	// --- cycle fields ---
	let strain = $state('');
	let startDate = $state(new Date().toISOString().slice(0, 10));
	let phase = $state<Phase>('vegetative');
	let notes = $state('');

	// --- lighting fields ---
	let mode = $state<LightScheduleMode>('off');
	let lightsOnAt = $state('06:00');
	let customHours = $state(18);
	// Per-phase hours for phase mode, seeded from overrides or recommended defaults.
	let phaseHours = $state<Record<string, number>>({});

	let busy = $state(false);
	let err = $state('');

	// Reseed the whole form whenever the modal opens.
	$effect(() => {
		if (!open) return;
		strain = cycle?.strain ?? '';
		startDate = (cycle?.startedAt ?? new Date().toISOString()).slice(0, 10);
		phase = cycle?.phase ?? 'vegetative';
		notes = cycle?.notes ?? '';

		mode = schedule?.mode ?? 'off';
		lightsOnAt = schedule?.lightsOnAt || '06:00';
		customHours = schedule?.onHours ?? 18;
		const seeded: Record<string, number> = {};
		for (const p of phases) seeded[p] = schedule?.phaseOnHours?.[p] ?? defaults[p] ?? 18;
		phaseHours = seeded;
	});

	const modeItems: SelectItem[] = [
		{ value: 'off', label: 'Manual only' },
		{ value: 'phase', label: 'Follow phase' },
		{ value: 'custom', label: 'Custom' }
	];
	const cap = (s: string) => s[0].toUpperCase() + s.slice(1);
	const phaseItems = $derived(phases.map((p) => ({ value: p, label: cap(p) })));

	function offHours(on: number): string {
		const off = Math.max(0, 24 - on);
		return `${on}/${off}`;
	}

	const cycleChanged = $derived(
		!cycle ||
			strain !== cycle.strain ||
			startDate !== cycle.startedAt.slice(0, 10) ||
			phase !== cycle.phase ||
			notes !== cycle.notes
	);

	async function save() {
		busy = true;
		err = '';
		try {
			// Only write the cycle when something changed, so we don't reset the
			// "days in phase" counter on a lighting-only edit.
			if (strain.trim() && cycleChanged) {
				await setCycle(environmentId, { strain, startedAt: startDate, phase, notes });
			}
			const phaseOnHours: Partial<Record<Phase, number>> = {};
			if (mode === 'phase') {
				for (const p of phases) {
					// Persist an override only when it differs from the recommendation.
					if (phaseHours[p] !== (defaults[p] ?? 18)) phaseOnHours[p as Phase] = phaseHours[p];
				}
			}
			await setSchedule(environmentId, { mode, lightsOnAt, onHours: customHours, phaseOnHours });
			open = false;
			onSaved?.();
		} catch (e) {
			err = e instanceof Error ? e.message : 'Save failed';
		} finally {
			busy = false;
		}
	}

	async function endCycle() {
		if (!confirm('End this cycle?')) return;
		try {
			await clearCycle(environmentId);
			open = false;
			onSaved?.();
		} catch (e) {
			err = e instanceof Error ? e.message : 'Failed';
		}
	}

	const field =
		'w-full rounded-md border border-rig-700 bg-rig-950 px-3 py-1.5 text-sm focus:border-rig-500 focus:outline-none';
</script>

<Dialog bind:open title={cycle ? 'Edit cycle' : 'Start a cycle'} description="Grow details and the light photoperiod for this tent.">
	<div class="space-y-6">
		{#if err}<p class="text-xs text-danger">{err}</p>{/if}

		<!-- Cycle -->
		<section class="space-y-3">
			<h3 class="text-xs font-semibold uppercase tracking-wide text-rig-400">Cycle</h3>
			<label class="block">
				<span class="text-xs text-rig-400">Strain</span>
				<input bind:value={strain} placeholder="e.g. Blue Dream" class="{field} mt-1" />
			</label>
			<div class="grid gap-3 sm:grid-cols-2">
				<label class="block">
					<span class="text-xs text-rig-400">Start date</span>
					<input type="date" bind:value={startDate} class="{field} mt-1" />
				</label>
				<label class="block">
					<span class="text-xs text-rig-400">Phase</span>
					<Select value={phase} onValueChange={(v) => (phase = v as Phase)} items={phaseItems} class="mt-1" />
				</label>
			</div>
			<label class="block">
				<span class="text-xs text-rig-400">Notes</span>
				<textarea bind:value={notes} rows="2" class="{field} mt-1"></textarea>
			</label>
		</section>

		<!-- Lighting -->
		<section class="space-y-3 border-t border-rig-800 pt-4">
			<h3 class="text-xs font-semibold uppercase tracking-wide text-rig-400">Lighting (photoperiod)</h3>
			{#if !hasPrimaryLight}
				<p class="rounded-md border border-warn/30 bg-warn/10 px-3 py-2 text-xs text-warn">
					No primary grow light is set for this tent. Mark a light as primary in Settings & Devices for the schedule to control it.
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
					<p class="text-xs text-rig-500">Hours of light per phase. Defaults are recommendations — override any of them.</p>
					{#each phases as p (p)}
						<div class="flex items-center gap-3">
							<span class="w-24 text-sm capitalize text-rig-300">{p}</span>
							<input
								type="number"
								min="0"
								max="24"
								step="0.5"
								bind:value={phaseHours[p]}
								class="w-20 rounded-md border border-rig-700 bg-rig-950 px-2 py-1 text-sm focus:border-rig-500 focus:outline-none"
							/>
							<span class="text-xs text-rig-500">
								{offHours(phaseHours[p] ?? 0)}
								{#if phaseHours[p] === (defaults[p] ?? 18)}· recommended{/if}
							</span>
						</div>
					{/each}
				</div>
			{/if}
		</section>

		<div class="flex justify-end gap-2 border-t border-rig-800 pt-4">
			{#if cycle}
				<Button variant="ghost" onclick={endCycle}>End cycle</Button>
			{/if}
			<Button variant="ghost" onclick={() => (open = false)}>Cancel</Button>
			<Button onclick={save} disabled={busy || !strain.trim()}>Save</Button>
		</div>
	</div>
</Dialog>
