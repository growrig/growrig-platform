<script lang="ts">
	import type { Grow, GrowSummary, LightSchedule, StageLightDefaults } from '$lib/types';
	import LightingModal from '$lib/components/LightingModal.svelte';
	import { nextTransition } from '$lib/photoperiod';
	import { relTime, titleCase } from '$lib/format';
	import Sun from '@lucide/svelte/icons/sun';
	import Sprout from '@lucide/svelte/icons/sprout';

	interface Props {
		environmentId: string;
		/** Live summary of this environment's control grow, if set. */
		grow?: GrowSummary;
		schedule?: LightSchedule;
		/** All grows, for the control-grow picker and its stage list. */
		grows: Grow[];
		defaults: StageLightDefaults;
		hasPrimaryLight: boolean;
		/** When false, everything is read-only. */
		canEdit?: boolean;
	}
	let { environmentId, grow, schedule, grows, defaults, hasPrimaryLight, canEdit = true }: Props = $props();

	let editing = $state(false);

	// Ticks so the countdown to the next light transition stays live.
	let nowMs = $state(Date.now());
	$effect(() => {
		const t = setInterval(() => (nowMs = Date.now()), 30_000);
		return () => clearInterval(t);
	});

	const stage = $derived(grow?.stage ?? '');

	const effectiveHours = $derived.by(() => {
		if (!schedule || schedule.mode === 'off') return null;
		if (schedule.mode === 'custom') return schedule.onHours;
		return schedule.stageOnHours?.[stage] ?? defaults[stage] ?? 18;
	});

	const lightingSummary = $derived.by(() => {
		if (!schedule || schedule.mode === 'off') return 'Lighting: manual';
		const h = effectiveHours ?? 0;
		const off = Math.max(0, 24 - h);
		const follows = schedule.mode === 'phase' ? ' · follows stage' : '';
		const next = nextTransition(schedule, stage, defaults, nowMs);
		const countdown = next ? ` · ${next.on ? 'on' : 'off'} in ${relTime(next.at - nowMs)}` : '';
		return `${h}/${off} · on at ${schedule.lightsOnAt}${follows}${countdown}`;
	});
</script>

<section>
	<div class="mb-3 flex items-center justify-between">
		<h2 class="text-sm font-semibold uppercase tracking-wide text-rig-400">Control grow &amp; lighting</h2>
	</div>
	<div class="rounded-lg border border-rig-800 bg-rig-950/40 p-4">
		{#if grow}
			<div class="flex items-center justify-between">
				<div>
					<a href="/grows/{grow.id}" class="text-lg font-semibold hover:text-leaf">{grow.name || 'Unnamed grow'}</a>
					<div class="mt-1 flex flex-wrap items-center gap-2 text-sm text-rig-400">
						<span class="rounded-full bg-rig-800 px-2 py-0.5 text-xs capitalize text-leaf">{grow.stage || '—'}</span>
						<span>day {grow.totalDays}</span>
						<span class="text-rig-600">·</span>
						<span>{grow.stageDays}d in {grow.stage}</span>
						<span class="text-rig-600">·</span>
						<span class="inline-flex items-center gap-1"><Sprout size={13} /> {grow.plantCount} plants</span>
					</div>
					{#if grow.species}
						<div class="mt-1 text-xs text-rig-500 capitalize">
							{titleCase(grow.species)}
						</div>
					{/if}
					<div class="mt-2 flex items-center gap-1.5 text-sm text-rig-400">
						<Sun size={14} class={schedule && schedule.mode !== 'off' ? 'text-warn' : 'text-rig-600'} />
						<span>{lightingSummary}</span>
					</div>
				</div>
				{#if canEdit}
					<button onclick={() => (editing = true)} class="rounded-md border border-rig-700 px-3 py-1.5 text-sm text-rig-300 hover:border-rig-500">Edit</button>
				{/if}
			</div>
		{:else}
			<div class="flex items-center justify-between">
				<div class="text-sm text-rig-400">
					No control grow selected.
					<span class="text-rig-500">Nominate a grow to drive this tent's photoperiod.</span>
				</div>
				{#if canEdit}
					<button onclick={() => (editing = true)} class="rounded-md bg-rig-500 px-3 py-1.5 text-sm font-medium text-rig-950 hover:bg-rig-400">Set control grow</button>
				{/if}
			</div>
		{/if}
	</div>
</section>

{#if canEdit}
	<LightingModal
		bind:open={editing}
		{environmentId}
		controlGrowId={grow?.id ?? ''}
		{schedule}
		{grows}
		{defaults}
		{hasPrimaryLight}
	/>
{/if}
