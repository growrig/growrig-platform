<script lang="ts">
	import type { Environment, Location } from '$lib/types';
	import { updateEnvironment } from '$lib/api';
	import { formatDimensions, volumeM3 } from '$lib/format';
	import { Button, Slider } from '$lib/components/ui';
	import EnvironmentDetailsDialog from '$lib/components/EnvironmentDetailsDialog.svelte';
	import Pencil from '@lucide/svelte/icons/pencil';

	interface Props {
		env: Environment;
		rooms: Environment[];
		locations: Location[];
		onChanged: () => void;
		flash: (kind: 'ok' | 'err', text: string) => void;
	}
	let { env, rooms, locations, onChanged, flash }: Props = $props();

	let editOpen = $state(false);
	let temp = $state(24);
	let humidity = $state(55);
	let co2 = $state(0);
	let emergency = $state(35);
	let leafOffset = $state(-2);
	let busy = $state(false);

	$effect(() => {
		temp = env.targetTempC;
		humidity = env.targetHumidity;
		co2 = env.targetCO2;
		emergency = env.emergencyTempC;
		leafOffset = env.leafTempOffsetC ?? -2;
	});

	const roomName = $derived(rooms.find((room) => room.id === env.airSourceId)?.name);
	const locationName = $derived(locations.find((l) => l.id === env.locationId)?.name);
	const dimensions = $derived(formatDimensions(env.widthCm, env.depthCm, env.heightCm));

	function onDetailsSaved() {
		flash('ok', 'Environment details saved');
		onChanged();
	}

	async function saveTargets() {
		busy = true;
		try {
			await updateEnvironment(env.id, {
				name: env.name, kind: env.kind, model: env.model, airSourceId: env.airSourceId,
				locationId: env.locationId,
				widthCm: env.widthCm, depthCm: env.depthCm, heightCm: env.heightCm,
				targetTempC: temp, targetHumidity: humidity, targetCO2: co2, emergencyTempC: emergency, leafTempOffsetC: leafOffset
			});
			flash('ok', 'Targets and safety limits saved');
			onChanged();
		} catch (e) {
			flash('err', e instanceof Error ? e.message : 'Save failed');
		} finally { busy = false; }
	}
</script>

<section class="space-y-3">
	<div class="flex items-center justify-between">
		<h2 class="text-sm font-semibold uppercase tracking-wide text-rig-400">Environment</h2>
		<Button variant="secondary" size="sm" onclick={() => (editOpen = true)}><Pencil size={14} /> Edit</Button>
	</div>
	<div class="rounded-xl border border-rig-800 bg-rig-900/40 p-5">
		<div class="grid gap-x-8 gap-y-4 sm:grid-cols-2 lg:grid-cols-4">
			<div><div class="text-xs text-rig-500">Name</div><div class="mt-1 text-sm font-medium">{env.name}</div></div>
			<div><div class="text-xs text-rig-500">Type</div><div class="mt-1 text-sm capitalize">{env.kind}</div></div>
			<div><div class="text-xs text-rig-500">{env.kind === 'tent' ? 'Tent model' : 'Model'}</div><div class="mt-1 text-sm">{env.model || '—'}</div></div>
			<div><div class="text-xs text-rig-500">Location</div><div class="mt-1 text-sm">{locationName || 'None'}</div></div>
			<div><div class="text-xs text-rig-500">ID</div><div class="mt-1 truncate font-mono text-xs text-rig-300">{env.id}</div></div>
			{#if env.kind === 'tent'}
				<div><div class="text-xs text-rig-500">Dimensions</div><div class="mt-1 text-sm">{dimensions || '—'}</div></div>
				<div><div class="text-xs text-rig-500">Volume</div><div class="mt-1 text-sm">{volumeM3(env.widthCm, env.depthCm, env.heightCm) ? `${volumeM3(env.widthCm, env.depthCm, env.heightCm).toFixed(2)} m³` : '—'}</div></div>
				<div><div class="text-xs text-rig-500">Air source</div><div class="mt-1 text-sm">{roomName || 'None'}</div></div>
			{/if}
		</div>
	</div>
</section>

{#if env.kind === 'tent'}
	<section class="space-y-3">
		<div><h2 class="text-sm font-semibold uppercase tracking-wide text-rig-400">Targets & Safety</h2><p class="mt-1 text-xs text-rig-500">Climate targets and emergency protection limits.</p></div>
		<div class="rounded-xl border border-rig-800 bg-rig-900/40 p-5">
			<div class="grid gap-6 sm:grid-cols-2 lg:grid-cols-5">
				<div><span class="text-sm text-rig-400">Temperature — {temp}°C</span><Slider min={15} max={35} step={0.5} bind:value={temp} class="mt-3" /></div>
				<div><span class="text-sm text-rig-400">Humidity — {humidity}%</span><Slider min={20} max={90} step={1} bind:value={humidity} class="mt-3" /></div>
				<div><span class="text-sm text-rig-400">CO₂ — {co2 ? `${co2} ppm` : 'off'}</span><Slider min={0} max={1500} step={50} bind:value={co2} class="mt-3" /></div>
				<div><span class="text-sm text-rig-400">Emergency temperature — {emergency}°C</span><Slider min={28} max={45} step={0.5} bind:value={emergency} tone="warn" class="mt-3" /></div>
				<div><span class="text-sm text-rig-400">Leaf temperature offset — {leafOffset > 0 ? '+' : ''}{leafOffset}°C</span><Slider min={-5} max={5} step={0.5} bind:value={leafOffset} class="mt-3" /><p class="mt-2 text-xs text-rig-500">Estimated leaf temp relative to air; −2°C is a common starting point.</p></div>
			</div>
			<div class="mt-5"><Button onclick={saveTargets} disabled={busy}>Save targets</Button></div>
		</div>
	</section>
{/if}

<EnvironmentDetailsDialog {env} {rooms} {locations} bind:open={editOpen} onSaved={onDetailsSaved} onLocationCreated={onChanged} />
