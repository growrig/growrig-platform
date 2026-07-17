<script lang="ts">
	import type { EnvironmentView, LightSchedule, StageLightDefaults } from '$lib/types';
	import { nextTransition } from '$lib/photoperiod';
	import { relTime } from '$lib/format';
	import Lightbulb from '@lucide/svelte/icons/lightbulb';
	import Wind from '@lucide/svelte/icons/wind';
	import Droplets from '@lucide/svelte/icons/droplets';

	interface Props {
		env: EnvironmentView;
		schedule?: LightSchedule;
		defaults: StageLightDefaults;
		stage: string;
	}
	let { env, schedule, defaults, stage }: Props = $props();

	// Ticks so the "scheduled off in …" countdown stays live.
	let nowMs = $state(Date.now());
	$effect(() => {
		const t = setInterval(() => (nowMs = Date.now()), 30_000);
		return () => clearInterval(t);
	});

	const lights = $derived(env.controls?.filter((c) => c.kind === 'light') ?? []);
	const primaryLight = $derived(lights.find((l) => l.primary) ?? lights[0]);
	const lightOn = $derived(!!primaryLight?.on);
	const lightPct = $derived.by(() => {
		if (!primaryLight?.wattage) return null;
		const power = primaryLight.power ?? (primaryLight.on ? primaryLight.wattage : 0);
		return Math.round((power / primaryLight.wattage) * 100);
	});
	const transition = $derived(nextTransition(schedule, stage, defaults, nowMs));

	const lightingLine = $derived.by(() => {
		if (!lights.length) return 'No light assigned';
		const parts = [lightOn ? 'On' : 'Off'];
		if (lightPct != null) parts.push(`${lightPct}%`);
		if (transition) parts.push(`${transition.on ? 'on' : 'off'} in ${relTime(transition.at - nowMs)}`);
		return parts.join(' · ');
	});

	const fans = $derived(env.controls?.filter((c) => c.kind === 'fan') ?? []);
	const exhaust = $derived(fans.filter((f) => f.role === 'exhaust' || f.role === 'intake'));
	const circulation = $derived(fans.filter((f) => f.role === 'circulation'));
	const maxSpeed = (list: typeof fans) => list.reduce((m, f) => Math.max(m, f.desiredSpeed), 0);
	const airLine = $derived.by(() => {
		if (!fans.length) return 'No fans assigned';
		const parts = ['Auto'];
		if (exhaust.length) parts.push(`Exhaust ${maxSpeed(exhaust)}%`);
		if (circulation.length) parts.push(`Circulation ${maxSpeed(circulation)}%`);
		return parts.join(' · ');
	});

	const irrigation = $derived(env.irrigation ?? []);
	const irrigationTypeLabel: Record<string, string> = {
		autopot: 'AutoPot',
		drip: 'Drip',
		wick: 'Wick',
		ebb_flow: 'Ebb & flow',
		hand: 'Hand-watering'
	};
	// A passive setup (AutoPot today) auto-waters the grow: no manual watering.
	const autoWatered = $derived(irrigation.length > 0);
	const irrigationLine = $derived.by(() => {
		if (!irrigation.length) return 'Not configured';
		const unit = irrigation[0];
		const parts = [irrigationTypeLabel[unit.type] ?? unit.type, unit.mode === 'controlled' ? 'controlled' : 'passive'];
		if (unit.reservoirL) parts.push(`${unit.reservoirL} L reservoir`);
		if (irrigation.length > 1) parts.push(`+${irrigation.length - 1} more`);
		return parts.join(' · ');
	});
</script>

<section>
	<h2 class="mb-3 text-sm font-semibold uppercase tracking-wide text-rig-400">Operating now</h2>
	<div class="space-y-2.5">
		<div class="flex items-start gap-3 rounded-lg border border-rig-800 bg-rig-950/40 p-3">
			<Lightbulb size={16} class="mt-0.5 shrink-0 {lightOn ? 'text-leaf' : 'text-rig-500'}" />
			<div class="min-w-0">
				<div class="text-sm font-medium">Lighting</div>
				<div class="text-xs text-rig-400">{lightingLine}</div>
			</div>
		</div>
		<div class="flex items-start gap-3 rounded-lg border border-rig-800 bg-rig-950/40 p-3">
			<Wind size={16} class="mt-0.5 shrink-0 text-rig-400" />
			<div class="min-w-0">
				<div class="text-sm font-medium">Air exchange</div>
				<div class="text-xs text-rig-400">{airLine}</div>
			</div>
		</div>
		<div class="flex items-start gap-3 rounded-lg border border-rig-800 bg-rig-950/40 p-3">
			<Droplets size={16} class="mt-0.5 shrink-0 {autoWatered ? 'text-sky-400' : 'text-rig-500'}" />
			<div class="min-w-0">
				<div class="text-sm font-medium">Irrigation</div>
				<div class="text-xs {autoWatered ? 'text-rig-400' : 'text-rig-500'}">{irrigationLine}</div>
				{#if autoWatered}
					<div class="mt-0.5 text-[11px] text-sky-400/80">Auto-watered — top up the reservoir instead of hand-watering</div>
				{/if}
			</div>
		</div>
	</div>
</section>
