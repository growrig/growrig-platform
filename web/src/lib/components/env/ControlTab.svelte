<script lang="ts">
	import { slide } from 'svelte/transition';
	import { errMsg } from '$lib/errors';
	import type { EnvironmentView, Grow, StageLightDefaults, ControlMode } from '$lib/types';
	import { setControl, type ControlInput } from '$lib/api';
	import { Switch } from '$lib/components/ui';
	import LightingPanel from './LightingPanel.svelte';
	import AirPanel from './AirPanel.svelte';
	import ControlModeToggle from './ControlModeToggle.svelte';
	import Zap from '@lucide/svelte/icons/zap';
	import Droplets from '@lucide/svelte/icons/droplets';
	import ChevronDown from '@lucide/svelte/icons/chevron-down';

	interface Props {
		env: EnvironmentView;
		canWrite: boolean;
		grows: Grow[];
		defaults: StageLightDefaults;
	}
	let { env, canWrite, grows, defaults }: Props = $props();

	let irrigationOpen = $state(false);
	let notice = $state<{ kind: 'ok' | 'err'; text: string } | null>(null);
	function flash(kind: 'ok' | 'err', text: string) {
		notice = { kind, text };
		setTimeout(() => (notice = null), 2500);
	}

	const lights = $derived(env.controls?.filter((c) => c.kind === 'light') ?? []);
	const fans = $derived(env.controls?.filter((c) => c.kind === 'fan') ?? []);
	const irrigation = $derived(env.irrigation ?? []);

	// A category is "applicable" (counts toward Full automatic, gets a live toggle)
	// only when it has hardware whose automatic mode means something.
	const applicable = $derived({
		lighting: env.kind === 'tent' && lights.length > 0,
		air: fans.length > 0,
		irrigation: irrigation.length > 0
	});
	const applicableKeys = $derived(
		(['lighting', 'air', 'irrigation'] as const).filter((k) => applicable[k])
	);
	const modeOf = $derived({
		lighting: env.control.lighting.mode,
		air: env.control.airExchange.mode,
		irrigation: env.control.irrigation.mode
	});
	const autoCount = $derived(applicableKeys.filter((k) => modeOf[k] === 'auto').length);
	const allAuto = $derived(applicableKeys.length > 0 && autoCount === applicableKeys.length);

	async function setAll(auto: boolean) {
		const mode: ControlMode = auto ? 'auto' : 'manual';
		const body: ControlInput = {};
		if (applicable.lighting) body.lighting = mode;
		if (applicable.air) body.air = mode;
		if (applicable.irrigation) body.irrigation = mode;
		try {
			await setControl(env.id, body);
		} catch (e) {
			flash('err', errMsg(e, 'Could not update automation'));
		}
	}

	async function setIrrigation(mode: ControlMode) {
		try {
			await setControl(env.id, { irrigation: mode });
		} catch (e) {
			flash('err', errMsg(e, 'Could not change irrigation mode'));
		}
	}

	const irrigationTypeLabel: Record<string, string> = {
		autopot: 'AutoPot',
		drip: 'Drip',
		wick: 'Wick',
		ebb_flow: 'Ebb & flow',
		hand: 'Hand-watering'
	};
	const masterSummary = $derived.by(() => {
		if (applicableKeys.length === 0) return 'No controllable equipment yet';
		if (allAuto) return 'Every system is running automatically';
		if (autoCount === 0) return 'Everything is under manual control';
		return `${autoCount} of ${applicableKeys.length} systems automatic`;
	});
</script>

<div class="space-y-4">
	{#if notice}
		<div class="rounded-lg px-4 py-2 text-sm {notice.kind === 'ok' ? 'bg-leaf/15 text-leaf' : 'bg-danger/15 text-danger'}">{notice.text}</div>
	{/if}

	<!-- Master automation -->
	<section class="rounded-xl border border-leaf/25 bg-leaf/5 p-5">
		<div class="flex flex-wrap items-center justify-between gap-3">
			<div class="flex items-center gap-2.5">
				<Zap size={18} class={allAuto ? 'text-leaf' : 'text-rig-400'} />
				<div>
					<h2 class="text-base font-semibold">Full automatic</h2>
					<p class="text-xs text-rig-400">{masterSummary}</p>
				</div>
			</div>
			{#if canWrite && applicableKeys.length > 0}
				<div class="flex items-center gap-2">
					<span class="text-xs font-medium tabular-nums {allAuto ? 'text-leaf' : 'text-rig-400'}">{allAuto ? 'On' : 'Off'}</span>
					<Switch checked={allAuto} onCheckedChange={setAll} />
				</div>
			{/if}
		</div>
	</section>

	<LightingPanel {env} {canWrite} {grows} {defaults} {flash} />

	<AirPanel {env} {canWrite} {flash} />

	<!-- Irrigation -->
	<section class="rounded-xl border border-rig-800 bg-rig-900/40">
		<div class="flex flex-wrap items-center justify-between gap-3 p-5">
			<button type="button" onclick={() => (irrigationOpen = !irrigationOpen)} class="flex min-w-0 flex-1 items-center gap-2.5 text-left" aria-expanded={irrigationOpen}>
				<ChevronDown size={16} class="shrink-0 text-rig-500 transition-transform duration-200 {irrigationOpen ? 'rotate-0' : '-rotate-90'}" />
				<Droplets size={18} class={modeOf.irrigation === 'auto' && irrigation.length ? 'text-sky-400' : 'text-rig-500'} />
				<div class="min-w-0">
					<h2 class="text-base font-semibold">Irrigation</h2>
					<p class="truncate text-xs text-rig-400">
						{#if irrigation.length === 0}
							No irrigation equipment
						{:else if modeOf.irrigation === 'auto'}
							Auto-watered by {irrigationTypeLabel[irrigation[0].type] ?? irrigation[0].type}
						{:else}
							Hand-watering
						{/if}
					</p>
				</div>
			</button>
			{#if canWrite && irrigation.length > 0}
				<ControlModeToggle value={modeOf.irrigation} onChange={setIrrigation} />
			{/if}
		</div>

		{#if irrigationOpen}
			<div transition:slide={{ duration: 200 }} class="border-t border-rig-800 p-5 pt-4 text-sm text-rig-400">
				{#if irrigation.length === 0}
					No irrigation equipment installed. Add AutoPot under <a href="/env/{env.id}?tab=equipment" class="text-leaf hover:underline">Equipment</a>. Watering is manual — log each watering in the grow journal.
				{:else if modeOf.irrigation === 'auto'}
					{@const unit = irrigation[0]}
					<p>
						<span class="text-rig-200">{unit.name}</span> handles watering for this grow
						{#if unit.reservoirL}· {unit.reservoirL} L reservoir{/if}{#if unit.valveCount}· {unit.valveCount} valve{unit.valveCount === 1 ? '' : 's'}{/if}.
					</p>
					<p class="mt-1 text-xs text-rig-500">Top up the reservoir and check pH/EC instead of hand-watering. The watering task is handled automatically.</p>
				{:else}
					<p>Hand-watering — log each watering in the grow journal. The installed {irrigation[0].name} is not being relied on.</p>
				{/if}
			</div>
		{/if}
	</section>
</div>
