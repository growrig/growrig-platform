<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/state';
	import { live } from '$lib/live.svelte';
	import { history, setSwitch, getPhases } from '$lib/api';
	import type { Phase, Reading } from '$lib/types';
	import { climateTone, toneClass, vpdZone, volumeM3, formatDimensions } from '$lib/format';
	import StatTile from '$lib/components/StatTile.svelte';
	import VpdGauge from '$lib/components/VpdGauge.svelte';
	import Sparkline from '$lib/components/Sparkline.svelte';
	import CycleCard from '$lib/components/CycleCard.svelte';
	import SensorsDialog from '$lib/components/SensorsDialog.svelte';
	import ActivityLog from '$lib/components/ActivityLog.svelte';
	import { Switch } from '$lib/components/ui';
	import ArrowLeft from '@lucide/svelte/icons/arrow-left';
	import Settings from '@lucide/svelte/icons/settings';
	import Lightbulb from '@lucide/svelte/icons/lightbulb';
	import LightbulbOff from '@lucide/svelte/icons/lightbulb-off';
	import Camera from '@lucide/svelte/icons/camera';
	import Ruler from '@lucide/svelte/icons/ruler';
	import Box from '@lucide/svelte/icons/box';
	import Wind from '@lucide/svelte/icons/wind';
	import Star from '@lucide/svelte/icons/star';
	import Zap from '@lucide/svelte/icons/zap';

	const id = $derived(page.params.id);
	const env = $derived(live.snapshot?.environments?.find((e) => e.id === id));

	const fans = $derived(env?.controls?.filter((c) => c.kind === 'fan') ?? []);
	const lights = $derived(env?.controls?.filter((c) => c.kind === 'light') ?? []);
	// Primary grow light first, then the rest.
	const primaryLight = $derived(lights.find((l) => l.primary) ?? lights[0]);
	const orderedLights = $derived(
		primaryLight ? [primaryLight, ...lights.filter((l) => l.id !== primaryLight.id)] : []
	);

	const dims = $derived(env ? formatDimensions(env.widthCm, env.depthCm, env.heightCm) : '');
	const vol = $derived(env ? volumeM3(env.widthCm, env.depthCm, env.heightCm) : 0);
	const hasInfo = $derived(!!(dims || env?.airSource));

	let readings = $state<Reading[]>([]);
	let phases = $state<Phase[]>([]);
	async function refreshHistory() {
		if (!id) return;
		try {
			readings = await history(id, 120);
		} catch {
			/* keep last */
		}
	}
	onMount(() => {
		refreshHistory();
		getPhases().then((p) => (phases = p)).catch(() => {});
		const t = setInterval(refreshHistory, 5000);
		return () => clearInterval(t);
	});

	async function toggleLight(bindingId: string, on: boolean) {
		try {
			await setSwitch(bindingId, on);
		} catch {
			/* ignore; state reconciles via live feed */
		}
	}

	const healthTone = (h: string) =>
		h === 'online' ? 'bg-leaf/15 text-leaf' : h === 'stale' ? 'bg-warn/15 text-warn' : 'bg-danger/15 text-danger';
	const roleLabel: Record<string, string> = {
		exhaust: 'Exhaust',
		intake: 'Intake',
		circulation: 'Circulation',
		unassigned: 'Unassigned'
	};
</script>

<a href="/" class="mb-4 inline-flex items-center gap-1 text-sm text-rig-400 hover:text-rig-100">
	<ArrowLeft size={15} /> All environments
</a>

{#if !live.snapshot}
	<p class="text-rig-400">Connecting to Grow Core…</p>
{:else if !env}
	<p class="text-rig-400">Environment not found. <a href="/" class="text-leaf hover:underline">Go back</a></p>
{:else}
	<div class="space-y-6">
		<div class="flex items-start justify-between">
			<div class="flex items-center gap-3">
				<div>
					<h1 class="text-2xl font-semibold">{env.name}</h1>
					{#if env.model}<p class="text-sm text-rig-400">{env.model}</p>{/if}
				</div>
				<span class="rounded-full bg-rig-800 px-2 py-0.5 text-[10px] uppercase tracking-wide text-rig-400">{env.kind}</span>
				<span class="rounded-full px-2 py-0.5 text-xs {healthTone(env.health)}">{env.health}</span>
			</div>
			<div class="flex items-center gap-3">
				{#if env.kind === 'tent'}
					<span class="hidden text-sm text-rig-400 sm:inline">target {env.targetTempC}°C · {env.targetHumidity}% RH{#if env.targetCO2 > 0} · {env.targetCO2} ppm{/if}</span>
				{/if}
				<SensorsDialog sensors={env.sensors ?? []} />
				<a href="/env/{id}/settings" class="inline-flex items-center gap-1.5 rounded-md border border-rig-700 px-3 py-1.5 text-sm text-rig-300 transition-colors hover:border-rig-500 hover:text-rig-100">
					<Settings size={15} /> Settings & Devices
				</a>
			</div>
		</div>

		{#if env.kind === 'tent'}
			<CycleCard environmentId={env.id} cycle={env.cycle} {phases} />
		{/if}

		<!-- Basic info -->
		{#if hasInfo}
			<div class="flex flex-wrap items-center gap-x-8 gap-y-3 rounded-xl border border-rig-800 bg-rig-900/40 px-5 py-4">
				{#if dims}
					<div class="flex items-center gap-2">
						<Ruler size={16} class="text-rig-500" />
						<div>
							<div class="text-[11px] uppercase tracking-wide text-rig-500">Dimensions</div>
							<div class="text-sm text-rig-100">{dims}</div>
						</div>
					</div>
					{#if vol}
						<div class="flex items-center gap-2">
							<Box size={16} class="text-rig-500" />
							<div>
								<div class="text-[11px] uppercase tracking-wide text-rig-500">Volume</div>
								<div class="text-sm text-rig-100 tabular-nums">{vol.toFixed(2)} m³</div>
							</div>
						</div>
					{/if}
				{/if}
				{#if env.airSource}
					<div class="flex items-center gap-2">
						<Wind size={16} class="text-rig-500" />
						<div>
							<div class="text-[11px] uppercase tracking-wide text-rig-500">Air source</div>
							<div class="text-sm text-rig-100">{env.airSource.name}</div>
						</div>
					</div>
				{/if}
			</div>
		{/if}

		<!-- Climate tiles -->
		<div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
			<StatTile
				label="Temperature"
				value={env.hasTemp ? env.tempC.toFixed(1) : '—'}
				unit="°C"
				tone={env.hasTemp ? climateTone(env.tempC, env.targetTempC, env.emergencyTempC) : 'muted'}
			/>
			<StatTile label="Humidity" value={env.hasHum ? env.humidity.toFixed(0) : '—'} unit="%" />
			{#if env.hasCO2}
				<StatTile label="CO₂" value={env.co2.toFixed(0)} unit="ppm" />
			{/if}
			<VpdGauge vpd={env.vpd} ok={env.hasClimate} />
		</div>

		<!-- History -->
		{#if readings.length > 1}
			<div class="grid gap-4 sm:grid-cols-3">
				<div class="rounded-lg border border-rig-800 bg-rig-950/40 p-4">
					<span class="text-sm text-rig-400">Temperature</span>
					<Sparkline values={readings.map((r) => r.tempC)} target={env.targetTempC} unit="°C" />
				</div>
				<div class="rounded-lg border border-rig-800 bg-rig-950/40 p-4">
					<span class="text-sm text-rig-400">Humidity</span>
					<Sparkline values={readings.map((r) => r.humidity)} target={env.targetHumidity} color="var(--color-rig-300)" unit="%" />
				</div>
				<div class="rounded-lg border border-rig-800 bg-rig-950/40 p-4">
					<span class="text-sm text-rig-400">VPD</span>
					<Sparkline values={readings.map((r) => r.vpd)} color="var(--color-leaf)" unit="" />
				</div>
			</div>
		{/if}

		<!-- Light -->
		{#if env.kind === 'tent' || lights.length}
			<section>
				<h2 class="mb-3 text-sm font-semibold uppercase tracking-wide text-rig-400">Light</h2>
				{#if lights.length === 0}
					<div class="flex flex-wrap items-center justify-between gap-3 rounded-xl border border-dashed border-warn/40 bg-warn/5 p-4">
						<div class="flex items-center gap-3">
							<LightbulbOff size={22} class="text-warn" />
							<div>
								<div class="text-sm font-medium">No grow light assigned</div>
								<div class="text-xs text-rig-400">A grow box needs a light to run its photoperiod. Assign one to get started.</div>
							</div>
						</div>
						<a
							href="/env/{id}/settings#devices"
							class="inline-flex items-center gap-1.5 rounded-md bg-rig-500 px-4 py-1.5 text-sm font-medium text-rig-950 transition-colors hover:bg-rig-400"
						>
							<Lightbulb size={15} /> Assign a light
						</a>
					</div>
				{:else}
					<div class="grid gap-3 sm:grid-cols-2">
						{#each orderedLights as light (light.id)}
							<div class="flex items-center justify-between rounded-lg border p-3 {light.primary ? 'border-warn/40 bg-warn/5' : 'border-rig-800 bg-rig-950/40'}">
								<div class="flex items-center gap-2">
									{#if light.on}
										<Lightbulb size={18} class="text-leaf" />
									{:else}
										<LightbulbOff size={18} class="text-rig-500" />
									{/if}
									<div>
										<div class="flex items-center gap-2 text-sm font-medium">
											{light.name}
											{#if light.primary}
												<span class="inline-flex items-center gap-1 rounded-full bg-warn/15 px-2 py-0.5 text-[10px] font-medium uppercase tracking-wide text-warn">
													<Star size={10} fill="currentColor" /> Primary
												</span>
											{/if}
										</div>
										{#if light.wattage}
											<div class="flex items-center gap-1 text-xs text-rig-500">
												<Zap size={12} /> {light.wattage} W
											</div>
										{/if}
									</div>
								</div>
								<div class="flex items-center gap-2">
									<span class="text-xs font-medium tabular-nums {light.on ? 'text-leaf' : 'text-rig-400'}">
										{light.on ? 'On' : 'Off'}
									</span>
									<Switch checked={light.on} onCheckedChange={(v) => toggleLight(light.id, v)} />
								</div>
							</div>
						{/each}
					</div>
				{/if}
			</section>
		{/if}

		<!-- Controls -->
		{#if fans.length}
			<section>
				<h2 class="mb-3 text-sm font-semibold uppercase tracking-wide text-rig-400">Controls</h2>
				<div class="grid gap-3 sm:grid-cols-2">
					{#each fans as fan (fan.id)}
						<div class="rounded-lg border border-rig-800 bg-rig-950/40 p-3">
							<div class="mb-2 flex items-center justify-between text-sm">
								<span class="font-medium">{fan.name}</span>
								<span class="text-rig-400">{roleLabel[fan.role ?? 'unassigned']}</span>
							</div>
							<div class="mb-1 flex items-center justify-between text-xs text-rig-400">
								<span>speed</span>
								<span class="tabular-nums">{fan.desiredSpeed}% · {fan.rpm} rpm</span>
							</div>
							<div class="h-2 overflow-hidden rounded-full bg-rig-800">
								<div class="h-full rounded-full bg-rig-500 transition-all duration-500" style="width:{fan.desiredSpeed}%"></div>
							</div>
						</div>
					{/each}
				</div>
			</section>
		{/if}

		<!-- Cameras -->
		{#if env.cameras?.length}
			<section>
				<h2 class="mb-3 text-sm font-semibold uppercase tracking-wide text-rig-400">Cameras</h2>
				<div class="grid gap-3 sm:grid-cols-2">
					{#each env.cameras as cam (cam.id)}
						<div class="flex items-center gap-2 rounded-lg border border-rig-800 bg-rig-950/40 p-3 text-sm">
							<Camera size={18} class="text-rig-400" />
							<span>{cam.name}</span>
							<span class="ml-auto text-xs text-rig-500">{cam.entity}</span>
						</div>
					{/each}
				</div>
			</section>
		{/if}

		<!-- Air source (lung room) -->
		{#if env.airSource}
			<section>
				<h2 class="mb-3 text-sm font-semibold uppercase tracking-wide text-rig-400">Air source</h2>
				<a
					href="/env/{env.airSource.id}"
					class="flex items-center justify-between rounded-lg border border-rig-800 bg-rig-950/40 p-4 transition-colors hover:border-rig-600"
				>
					<span class="font-medium">{env.airSource.name}</span>
					{#if env.airSource.ok}
						<span class="text-sm tabular-nums text-rig-300">
							{env.airSource.tempC.toFixed(1)}°C · {env.airSource.humidity.toFixed(0)}% ·
							<span class={toneClass[vpdZone(env.airSource.vpd).tone]}>{env.airSource.vpd.toFixed(2)} kPa</span>
						</span>
					{:else}
						<span class="text-sm text-rig-500">no data</span>
					{/if}
				</a>
			</section>
		{/if}

		<section>
			<div class="mb-3 flex items-center justify-between"><h2 class="text-sm font-semibold uppercase tracking-wide text-rig-400">Activity Log</h2><a href="/activity" class="text-xs text-rig-500 hover:text-leaf">View all environments</a></div>
			<ActivityLog environmentId={env.id} limit={20} />
		</section>
	</div>
{/if}
