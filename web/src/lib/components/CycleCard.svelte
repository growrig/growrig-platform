<script lang="ts">
	import type { Cycle, Phase } from '$lib/types';
	import { setCycle, clearCycle } from '$lib/api';
	import { Select } from '$lib/components/ui';

	interface Props {
		environmentId: string;
		cycle?: Cycle;
		phases: Phase[];
	}
	let { environmentId, cycle, phases }: Props = $props();

	let editing = $state(false);
	let strain = $state('');
	let startDate = $state(new Date().toISOString().slice(0, 10));
	let phase = $state<Phase>('vegetative');
	let notes = $state('');
	let busy = $state(false);
	let err = $state('');

	function beginEdit() {
		strain = cycle?.strain ?? '';
		startDate = (cycle?.startedAt ?? new Date().toISOString()).slice(0, 10);
		phase = cycle?.phase ?? 'vegetative';
		notes = cycle?.notes ?? '';
		editing = true;
	}

	function daysSince(iso?: string): number {
		if (!iso) return 0;
		return Math.max(0, Math.floor((Date.now() - new Date(iso).getTime()) / 86400000));
	}

	async function save() {
		busy = true;
		err = '';
		try {
			await setCycle(environmentId, { strain, startedAt: startDate, phase, notes });
			editing = false;
		} catch (e) {
			err = e instanceof Error ? e.message : 'Save failed';
		} finally {
			busy = false;
		}
	}

	async function clear() {
		if (!confirm('End this cycle?')) return;
		try {
			await clearCycle(environmentId);
			editing = false;
		} catch (e) {
			err = e instanceof Error ? e.message : 'Failed';
		}
	}

	const field = 'w-full rounded-md border border-rig-700 bg-rig-950 px-3 py-1.5 text-sm focus:border-rig-500 focus:outline-none';
</script>

<section>
	<h2 class="mb-3 text-sm font-semibold uppercase tracking-wide text-rig-400">Cycle</h2>
	<div class="rounded-lg border border-rig-800 bg-rig-950/40 p-4">
		{#if err}<p class="mb-2 text-xs text-danger">{err}</p>{/if}

		{#if editing}
			<div class="space-y-3">
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
						<Select value={phase} onValueChange={(value) => (phase = value as Phase)} items={phases.map((p) => ({ value: p, label: p[0].toUpperCase() + p.slice(1) }))} class="mt-1" />
					</label>
				</div>
				<div class="flex gap-2">
					<button onclick={save} disabled={busy || !strain.trim()} class="rounded-md bg-rig-500 px-4 py-1.5 text-sm font-medium text-rig-950 hover:bg-rig-400 disabled:opacity-50">Save</button>
					<button onclick={() => (editing = false)} class="rounded-md border border-rig-700 px-4 py-1.5 text-sm text-rig-300 hover:border-rig-500">Cancel</button>
					{#if cycle}
						<button onclick={clear} class="ml-auto rounded-md border border-rig-700 px-4 py-1.5 text-sm text-rig-300 hover:border-danger hover:text-danger">End cycle</button>
					{/if}
				</div>
			</div>
		{:else if cycle}
			<div class="flex items-center justify-between">
				<div>
					<div class="text-lg font-semibold">{cycle.strain || 'Unnamed strain'}</div>
					<div class="mt-1 flex items-center gap-2 text-sm text-rig-400">
						<span class="rounded-full bg-rig-800 px-2 py-0.5 text-xs capitalize text-leaf">{cycle.phase}</span>
						<span>day {daysSince(cycle.startedAt)}</span>
						<span class="text-rig-600">·</span>
						<span>{daysSince(cycle.phaseStarted)}d in {cycle.phase}</span>
					</div>
				</div>
				<button onclick={beginEdit} class="rounded-md border border-rig-700 px-3 py-1.5 text-sm text-rig-300 hover:border-rig-500">Edit</button>
			</div>
		{:else}
			<div class="flex items-center justify-between">
				<span class="text-sm text-rig-400">No active cycle.</span>
				<button onclick={beginEdit} class="rounded-md bg-rig-500 px-3 py-1.5 text-sm font-medium text-rig-950 hover:bg-rig-400">Start a cycle</button>
			</div>
		{/if}
	</div>
</section>
