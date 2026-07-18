<script lang="ts">
	import { slide } from 'svelte/transition';
	import { errMsg } from '$lib/errors';
	import type { EnvironmentView, Grow, StageLightDefaults, ControlMode } from '$lib/types';
	import { setControl, setSwitch } from '$lib/api';
	import { nextTransition } from '$lib/photoperiod';
	import { relTime, titleCase } from '$lib/format';
	import ControlModeToggle from './ControlModeToggle.svelte';
	import { Switch } from '$lib/components/ui';
	import Lightbulb from '@lucide/svelte/icons/lightbulb';
	import LightbulbOff from '@lucide/svelte/icons/lightbulb-off';
	import Sun from '@lucide/svelte/icons/sun';
	import ChevronDown from '@lucide/svelte/icons/chevron-down';

	interface Props {
		env: EnvironmentView;
		canWrite: boolean;
		grows: Grow[];
		defaults: StageLightDefaults;
		flash?: (kind: 'ok' | 'err', text: string) => void;
	}
	let { env, canWrite, grows, defaults, flash }: Props = $props();

	let open = $state(false);

	const lights = $derived(env.controls?.filter((c) => c.kind === 'light') ?? []);
	const hasPrimaryLight = $derived(lights.length > 0);
	const mode = $derived<ControlMode>(env.control.lighting.mode);
	const stage = $derived(env.grow?.stage ?? '');
	const controlGrowName = $derived(grows.find((g) => g.id === env.controlGrowId)?.name ?? 'None');

	// Live countdown to the next scheduled transition.
	let nowMs = $state(Date.now());
	$effect(() => {
		const t = setInterval(() => (nowMs = Date.now()), 30_000);
		return () => clearInterval(t);
	});

	// Read-only view of the effective photoperiod (Automatic mode).
	const scheduleStages = $derived.by(() => {
		const g = grows.find((gr) => gr.id === env.controlGrowId);
		return g?.stages?.length ? g.stages : Object.keys(defaults);
	});
	const effectiveHours = $derived.by(() => {
		if (!env.schedule || env.schedule.mode === 'off') return null;
		if (env.schedule.mode === 'custom') return env.schedule.onHours;
		return env.schedule.stageOnHours?.[stage] ?? defaults[stage] ?? 18;
	});
	const stageHours = (st: string) => env.schedule?.stageOnHours?.[st] ?? defaults[st] ?? 18;

	const summary = $derived.by(() => {
		if (mode === 'manual') return 'Manual — you control the light';
		const h = effectiveHours ?? 0;
		const follows = env.schedule?.mode === 'phase' ? ' · follows stage' : '';
		const next = nextTransition(env.schedule, stage, defaults, nowMs);
		const countdown = next ? ` · ${next.on ? 'on' : 'off'} in ${relTime(next.at - nowMs)}` : '';
		return `${h}/${Math.max(0, 24 - h)} · on at ${env.schedule?.lightsOnAt}${follows}${countdown}`;
	});

	async function setMode(next: ControlMode) {
		try {
			await setControl(env.id, { lighting: next });
		} catch (e) {
			flash?.('err', errMsg(e, 'Could not change lighting mode'));
		}
	}
	async function toggleLight(id: string, on: boolean) {
		try {
			await setSwitch(id, on);
		} catch {
			/* reconciles via live feed */
		}
	}

	const ro = 'rounded-md border border-rig-800 bg-rig-950/40 px-3 py-2 text-sm text-rig-200';
</script>

<section class="rounded-xl border border-rig-800 bg-rig-900/40">
	<!-- Header (click to expand/collapse) -->
	<div class="flex flex-wrap items-center justify-between gap-3 p-5">
		<button type="button" onclick={() => (open = !open)} class="flex min-w-0 flex-1 items-center gap-2.5 text-left" aria-expanded={open}>
			<ChevronDown size={16} class="shrink-0 text-rig-500 transition-transform duration-200 {open ? 'rotate-0' : '-rotate-90'}" />
			<Sun size={18} class={mode === 'auto' ? 'text-warn' : 'text-rig-500'} />
			<div class="min-w-0">
				<h2 class="text-base font-semibold">Lighting</h2>
				<p class="truncate text-xs text-rig-400">{summary}</p>
			</div>
		</button>
		{#if canWrite}
			<ControlModeToggle value={mode} onChange={setMode} disabled={!hasPrimaryLight} />
		{/if}
	</div>

	{#if open}
		<div transition:slide={{ duration: 200 }} class="border-t border-rig-800 p-5 pt-4">
			{#if !hasPrimaryLight}
				<p class="rounded-md border border-warn/30 bg-warn/10 px-3 py-2 text-xs text-warn">
					No grow light assigned. Add one under <a href="/env/{env.id}?tab=equipment" class="underline">Equipment</a> to control the photoperiod.
				</p>
			{:else if mode === 'auto'}
				<!-- Read-only photoperiod (automation owns these values). -->
				<div class="grid gap-3 sm:grid-cols-3">
					<div><span class="text-xs text-rig-400">Control grow</span><div class="mt-1 {ro}">{controlGrowName}</div></div>
					<div><span class="text-xs text-rig-400">Schedule</span><div class="mt-1 {ro}">{env.schedule?.mode === 'custom' ? 'Custom' : 'Follow stage'}</div></div>
					<div><span class="text-xs text-rig-400">Lights on at</span><div class="mt-1 {ro}">{env.schedule?.lightsOnAt ?? '—'}</div></div>
				</div>
				{#if env.schedule?.mode === 'custom'}
					<div class="mt-4 text-sm text-rig-300">{env.schedule.onHours}/{Math.max(0, 24 - env.schedule.onHours)} hours of light.</div>
				{:else}
					<div class="mt-4 space-y-1">
						<p class="text-xs text-rig-500">Hours of light per stage (recommended for the control grow).</p>
						{#each scheduleStages as st (st)}
							<div class="flex items-center gap-3 text-sm">
								<span class="w-24 capitalize text-rig-300">{titleCase(st)}</span>
								<span class="tabular-nums text-rig-200">{stageHours(st)}/{Math.max(0, 24 - stageHours(st))}</span>
								{#if st === stage}<span class="rounded-full bg-warn/15 px-2 py-0.5 text-[10px] uppercase tracking-wide text-warn">current</span>{/if}
							</div>
						{/each}
					</div>
				{/if}
			{:else}
				<!-- Manual: direct light switches -->
				<div class="grid gap-3 sm:grid-cols-2">
					{#each lights as light (light.id)}
						<div class="flex items-center justify-between rounded-lg border border-rig-800 bg-rig-950/40 p-3">
							<div class="flex items-center gap-2 text-sm font-medium">
								{#if light.on}<Lightbulb size={16} class="text-leaf" />{:else}<LightbulbOff size={16} class="text-rig-500" />{/if}
								{light.name}
							</div>
							<div class="flex items-center gap-2">
								<span class="text-xs tabular-nums {light.on ? 'text-leaf' : 'text-rig-400'}">{light.on ? 'On' : 'Off'}</span>
								{#if canWrite}<Switch checked={light.on} onCheckedChange={(v) => toggleLight(light.id, v)} />{/if}
							</div>
						</div>
					{/each}
				</div>
			{/if}
		</div>
	{/if}
</section>
