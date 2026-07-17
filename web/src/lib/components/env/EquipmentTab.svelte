<script lang="ts">
	import type { CameraRef, EnvironmentView } from '$lib/types';
	import type { MetricDescriptor } from '$lib/components/MetricModal.svelte';
	import { cameraProxyURL, setSwitch } from '$lib/api';
	import { toneClass, vpdZone, measurementLabel, measurementUnit } from '$lib/format';
	import { measurementIcon } from '$lib/icons';
	import { Switch } from '$lib/components/ui';
	import CameraPreview from '$lib/components/CameraPreview.svelte';
	import CameraDetailModal from '$lib/components/CameraDetailModal.svelte';
	import CameraStreamStats from '$lib/components/CameraStreamStats.svelte';
	import Lightbulb from '@lucide/svelte/icons/lightbulb';
	import LightbulbOff from '@lucide/svelte/icons/lightbulb-off';
	import Star from '@lucide/svelte/icons/star';
	import Zap from '@lucide/svelte/icons/zap';
	import Settings from '@lucide/svelte/icons/settings';
	import Droplets from '@lucide/svelte/icons/droplets';
	import Camera from '@lucide/svelte/icons/camera';

	interface Props {
		env: EnvironmentView;
		canWrite: boolean;
		isAdmin: boolean;
		onMetric: (descriptor: MetricDescriptor, title: string, unit: string) => void;
	}
	let { env, canWrite, isAdmin, onMetric }: Props = $props();

	let cameraOpen = $state(false);
	let cameraDetail = $state<CameraRef | null>(null);
	function openCamera(cam: CameraRef) {
		cameraDetail = cam;
		cameraOpen = true;
	}

	const cameras = $derived(env.cameras ?? []);
	const hasCamera = $derived(cameras.length > 0);

	const lights = $derived(env.controls?.filter((c) => c.kind === 'light') ?? []);
	const primaryLight = $derived(lights.find((l) => l.primary) ?? lights[0]);
	const orderedLights = $derived(
		primaryLight ? [primaryLight, ...lights.filter((l) => l.id !== primaryLight.id)] : []
	);
	const showLight = $derived(env.kind === 'tent' || lights.length > 0);

	const fans = $derived(env.controls?.filter((c) => c.kind === 'fan') ?? []);
	const ventilationFans = $derived(fans.filter((f) => f.role === 'exhaust' || f.role === 'intake'));
	const otherFanSections = $derived(
		[
			{ label: 'Circulation', fans: fans.filter((f) => f.role === 'circulation') },
			{ label: 'Other fans', fans: fans.filter((f) => !f.role || f.role === 'unassigned') }
		].filter((s) => s.fans.length > 0)
	);
	const rpmProgress = (rpm: number, maxRpm?: number) =>
		Math.min(100, Math.max(0, (rpm / (maxRpm || 2500)) * 100));
	const roleLabel: Record<string, string> = {
		exhaust: 'Exhaust',
		intake: 'Intake',
		circulation: 'Circulation',
		unassigned: 'Unassigned'
	};

	const sensorOrder = ['temperature', 'humidity', 'co2'] as const;
	const sensorGroups = $derived(
		sensorOrder
			.map((m) => ({ measurement: m, items: (env.sensors ?? []).filter((s) => s.measurement === m) }))
			.filter((g) => g.items.length > 0)
	);

	const irrigation = $derived(env.irrigation ?? []);
	const irrigationTypeLabel: Record<string, string> = {
		autopot: 'AutoPot',
		drip: 'Drip',
		wick: 'Wick',
		ebb_flow: 'Ebb & flow',
		hand: 'Hand-watering'
	};

	async function toggleLight(bindingId: string, on: boolean) {
		try {
			await setSwitch(bindingId, on);
		} catch {
			/* reconciles via live feed */
		}
	}
</script>

{#snippet lightSection(compact: boolean)}
	{#if showLight}
		<section>
			<div class="mb-3 flex items-center justify-between">
				<h2 class="text-sm font-semibold uppercase tracking-wide text-rig-400">Light</h2>
				{#if isAdmin}
					<a href="/env/{env.id}/settings#devices" class="inline-flex items-center gap-1 text-xs text-rig-400 hover:text-rig-100">
						<Settings size={13} /> Manage
					</a>
				{/if}
			</div>
			{#if lights.length === 0}
				<div class="flex flex-wrap items-center justify-between gap-3 rounded-xl border border-dashed border-warn/40 bg-warn/5 p-4">
					<div class="flex items-center gap-3">
						<LightbulbOff size={22} class="text-warn" />
						<div>
							<div class="text-sm font-medium">No grow light assigned</div>
							<div class="text-xs text-rig-400">A grow box needs a light to run its photoperiod.</div>
						</div>
					</div>
					{#if isAdmin}
						<a href="/env/{env.id}/settings#devices" class="inline-flex items-center gap-1.5 rounded-md bg-rig-500 px-4 py-1.5 text-sm font-medium text-rig-950 transition-colors hover:bg-rig-400">
							<Lightbulb size={15} /> Assign a light
						</a>
					{/if}
				</div>
			{:else}
				<div class="grid gap-3 {compact ? '' : 'sm:grid-cols-2'}">
					{#each orderedLights as light (light.id)}
						{@const power = light.power ?? (light.on ? light.wattage ?? 0 : 0)}
						<div
							role="button" tabindex="0"
							onclick={() => onMetric({ kind: 'device', bindingId: light.id, metric: 'power' }, `${light.name} · power`, 'W')}
							onkeydown={(e) => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); onMetric({ kind: 'device', bindingId: light.id, metric: 'power' }, `${light.name} · power`, 'W'); } }}
							class="group cursor-pointer rounded-lg border border-rig-800 bg-rig-950/40 p-3 text-left transition-colors hover:border-rig-600 focus-visible:border-rig-500 focus-visible:outline-none"
						>
							<div class="flex items-center justify-between">
								<div class="flex items-center gap-2" role="presentation" onclick={(e) => e.stopPropagation()} onkeydown={(e) => e.stopPropagation()}>
									{#if light.on}<Lightbulb size={18} class="text-leaf" />{:else}<LightbulbOff size={18} class="text-rig-500" />{/if}
									<div class="flex items-center gap-2 text-sm font-medium">
										{light.name}
										{#if light.primary}
											<span class="inline-flex items-center gap-1 rounded-full bg-rig-800 px-2 py-0.5 text-[10px] font-medium uppercase tracking-wide text-rig-300">
												<Star size={10} fill="currentColor" class="text-warn" /> Primary
											</span>
										{/if}
									</div>
								</div>
								<div class="flex items-center gap-2">
									<span class="text-xs font-medium tabular-nums {light.on ? 'text-leaf' : 'text-rig-400'}">{light.on ? 'On' : 'Off'}</span>
									{#if canWrite}<Switch checked={light.on} onCheckedChange={(v) => toggleLight(light.id, v)} />{/if}
								</div>
							</div>
							{#if light.wattage}
								<div class="mb-1 mt-3 flex items-center justify-between text-xs text-rig-400">
									<span class="flex items-center gap-1"><Zap size={12} /> power</span>
									<span class="tabular-nums">{Math.round(power)} / {light.wattage} W</span>
								</div>
								<div class="h-2 overflow-hidden rounded-full bg-rig-800">
									<div class="h-full rounded-full bg-leaf transition-all duration-500" style="width:{light.wattage ? (power / light.wattage) * 100 : 0}%"></div>
								</div>
							{/if}
						</div>
					{/each}
				</div>
			{/if}
		</section>
	{/if}
{/snippet}

{#snippet fanSection(label: string, sectionFans: typeof fans, compact: boolean)}
	{#if sectionFans.length}
		<section>
			<h2 class="mb-3 text-sm font-semibold uppercase tracking-wide text-rig-400">{label}</h2>
			<div class="grid gap-3 {compact ? '' : 'sm:grid-cols-2'}">
				{#each sectionFans as fan (fan.id)}
					<div
						role="button" tabindex="0"
						onclick={() => onMetric({ kind: 'device', bindingId: fan.id, metric: 'rpm' }, `${fan.name} · speed`, 'rpm')}
						onkeydown={(e) => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); onMetric({ kind: 'device', bindingId: fan.id, metric: 'rpm' }, `${fan.name} · speed`, 'rpm'); } }}
						class="group cursor-pointer rounded-lg border border-rig-800 bg-rig-950/40 p-3 text-left transition-colors hover:border-rig-600 focus-visible:border-rig-500 focus-visible:outline-none"
					>
						<div class="mb-2 flex items-center justify-between text-sm">
							<span class="font-medium">{fan.name}</span>
							<span class="text-rig-400">{roleLabel[fan.role ?? 'unassigned']}</span>
						</div>
						<div class="mb-1 flex items-center justify-between text-xs text-rig-400">
							<span>speed</span>
							<span class="tabular-nums">{fan.desiredSpeed}% · {fan.rpm} rpm</span>
						</div>
						<div class="relative h-2 overflow-hidden rounded-full bg-rig-800" title="Speed setting {fan.desiredSpeed}% · RPM feedback {fan.rpm}">
							<div class="absolute inset-y-0 left-0 rounded-full bg-sky-500/45 transition-all duration-500" style="width:{fan.desiredSpeed}%"></div>
							<div class="absolute inset-y-0 left-0 rounded-full bg-leaf transition-all duration-500" style="width:{rpmProgress(fan.rpm, fan.maxRpm)}%"></div>
						</div>
					</div>
				{/each}
			</div>
		</section>
	{/if}
{/snippet}

{#snippet cameraCard(cam: CameraRef)}
	<div
		role="button" tabindex="0"
		onclick={() => openCamera(cam)}
		onkeydown={(e) => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); openCamera(cam); } }}
		class="group cursor-pointer overflow-hidden rounded-lg border border-rig-800 bg-rig-950/40 transition-colors hover:border-rig-600 focus-visible:border-rig-500 focus-visible:outline-none"
	>
		<CameraPreview
			url={cam.cameraType === 'rtsp' || cam.entity || !cam.streamUrl ? cameraProxyURL(cam.id) : cam.streamUrl}
			liveUrl={cam.cameraType === 'rtsp' || (!cam.streamUrl && !cam.entity) ? cameraProxyURL(cam.id, true) : ''}
			type={cam.cameraType === 'rtsp' ? 'snapshot' : cam.streamUrl ? cam.cameraType : 'snapshot'}
			refreshSeconds={cam.cameraType === 'rtsp' ? cam.cameraCaptureInterval ?? 60 : 2}
			class="border-0"
			emptyLabel="Connecting to camera…"
			errorLabel="Connecting to camera…"
		/>
		<div class="flex items-center gap-2 px-3 py-2 text-sm">
			<Camera size={16} class="text-rig-400" />
			<span class="transition-colors group-hover:text-leaf">{cam.name}</span>
			<span class="ml-auto flex items-center gap-2 text-xs text-rig-500">
				{#if cam.cameraType === 'rtsp'}
					<CameraStreamStats cameraId={cam.id} showProtocol />
				{:else}
					<span>{cam.entity ? 'Home Assistant' : cam.cameraType || 'Connecting…'}</span>
				{/if}
			</span>
		</div>
	</div>
{/snippet}

<div class="space-y-6">
	{#if hasCamera}
		<!-- Camera sits top-right beside light + ventilation. -->
		<div class="grid items-start gap-6 lg:grid-cols-2">
			<div class="space-y-6">
				{@render lightSection(true)}
				{@render fanSection('Ventilation', ventilationFans, true)}
			</div>
			<section>
				<h2 class="mb-3 text-sm font-semibold uppercase tracking-wide text-rig-400">
					{cameras.length === 1 ? 'Camera' : 'Cameras'}
				</h2>
				<div class="space-y-3">
					{#each cameras as cam (cam.id)}
						{@render cameraCard(cam)}
					{/each}
				</div>
			</section>
		</div>
	{:else}
		{@render lightSection(false)}
		{@render fanSection('Ventilation', ventilationFans, false)}
	{/if}

	{#each otherFanSections as section (section.label)}
		{@render fanSection(section.label, section.fans, false)}
	{/each}

	<!-- Sensors -->
	{#if (env.sensors ?? []).length}
		<section>
			<h2 class="mb-3 text-sm font-semibold uppercase tracking-wide text-rig-400">Sensors</h2>
			<div class="space-y-4">
				{#each sensorGroups as group (group.measurement)}
					{@const GroupIcon = measurementIcon[group.measurement]}
					<div>
						<h3 class="mb-2 flex items-center gap-1.5 text-xs font-semibold uppercase tracking-wide text-rig-400">
							<GroupIcon size={14} /> {measurementLabel[group.measurement]}
						</h3>
						<div class="grid gap-2 sm:grid-cols-2">
							{#each group.items as s (s.id)}
								<div class="flex items-center justify-between rounded-lg border border-rig-800 bg-rig-950/40 px-3 py-2 text-sm">
									<span class="truncate">{s.name}</span>
									<span class="tabular-nums {s.ok ? 'text-rig-200' : 'text-rig-600'}">
										{s.ok ? `${s.value.toFixed(group.measurement === 'temperature' ? 1 : 0)}${measurementUnit[group.measurement]}` : 'offline'}
									</span>
								</div>
							{/each}
						</div>
					</div>
				{/each}
			</div>
		</section>
	{/if}

	<!-- Irrigation -->
	{#if irrigation.length}
		<section>
			<h2 class="mb-3 text-sm font-semibold uppercase tracking-wide text-rig-400">Irrigation</h2>
			<div class="grid gap-3 sm:grid-cols-2">
				{#each irrigation as unit (unit.id)}
					<div class="rounded-lg border border-rig-800 bg-rig-950/40 p-3">
						<div class="flex items-center justify-between">
							<div class="flex items-center gap-2 text-sm font-medium">
								<Droplets size={16} class="text-sky-400" />
								{unit.name}
							</div>
							<span class="inline-flex items-center gap-1 rounded-full bg-rig-800 px-2 py-0.5 text-[10px] font-medium uppercase tracking-wide text-rig-300">
								{unit.mode === 'controlled' ? 'Controlled' : 'Passive'}
							</span>
						</div>
						<div class="mt-2 text-xs text-rig-400">
							{irrigationTypeLabel[unit.type] ?? unit.type}
							{#if unit.reservoirL}· {unit.reservoirL} L reservoir{/if}
							{#if unit.valveCount}· {unit.valveCount} valve{unit.valveCount === 1 ? '' : 's'}{/if}
						</div>
					</div>
				{/each}
			</div>
		</section>
	{/if}

	<!-- Air source (lung room) -->
	{#if env.airSource}
		<section>
			<h2 class="mb-3 text-sm font-semibold uppercase tracking-wide text-rig-400">Air source</h2>
			<a href="/env/{env.airSource.id}" class="flex items-center justify-between rounded-lg border border-rig-800 bg-rig-950/40 p-4 transition-colors hover:border-rig-600">
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
</div>

{#if cameraDetail}
	<CameraDetailModal bind:open={cameraOpen} camera={cameraDetail} />
{/if}
