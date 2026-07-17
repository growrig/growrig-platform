<script lang="ts">
	import { errMsg } from '$lib/errors';
	import type { Environment } from '$lib/types';
	import { updateEnvironment } from '$lib/api';
	import { Button, Slider } from '$lib/components/ui';

	interface Props {
		env: Environment;
		/** When true the values are shown pre-filled but disabled, with no Save. */
		readOnly?: boolean;
		onSaved?: () => void;
		flash?: (kind: 'ok' | 'err', text: string) => void;
	}
	let { env, readOnly = false, onSaved, flash }: Props = $props();

	let temp = $state(24);
	let humidity = $state(55);
	let co2 = $state(0);
	let emergency = $state(35);
	let leafOffset = $state(-2);
	let busy = $state(false);

	// Optional display ranges (0 = unset); shown as the "ok band" on climate cards.
	let tMin = $state(0);
	let tMax = $state(0);
	let hMin = $state(0);
	let hMax = $state(0);
	let vMin = $state(0);
	let vMax = $state(0);
	let cMin = $state(0);
	let cMax = $state(0);

	$effect(() => {
		temp = env.targetTempC;
		humidity = env.targetHumidity;
		co2 = env.targetCO2;
		emergency = env.emergencyTempC;
		leafOffset = env.leafTempOffsetC ?? -2;
		tMin = env.targetTempMinC ?? 0;
		tMax = env.targetTempMaxC ?? 0;
		hMin = env.targetHumidityMin ?? 0;
		hMax = env.targetHumidityMax ?? 0;
		vMin = env.targetVpdMin ?? 0;
		vMax = env.targetVpdMax ?? 0;
		cMin = env.targetCo2Min ?? 0;
		cMax = env.targetCo2Max ?? 0;
	});

	const rangeInput =
		'w-full min-w-0 rounded-md border border-rig-700 bg-rig-950 px-2 py-1.5 text-sm tabular-nums focus:border-rig-500 focus:outline-none';

	async function saveTargets() {
		busy = true;
		try {
			await updateEnvironment(env.id, {
				name: env.name, kind: env.kind, model: env.model, airSourceId: env.airSourceId,
				locationId: env.locationId,
				widthCm: env.widthCm, depthCm: env.depthCm, heightCm: env.heightCm,
				targetTempC: temp, targetHumidity: humidity, targetCO2: co2, emergencyTempC: emergency, leafTempOffsetC: leafOffset,
				targetTempMinC: tMin, targetTempMaxC: tMax,
				targetHumidityMin: hMin, targetHumidityMax: hMax,
				targetVpdMin: vMin, targetVpdMax: vMax,
				targetCo2Min: cMin, targetCo2Max: cMax
			});
			flash?.('ok', 'Targets and safety limits saved');
			onSaved?.();
		} catch (e) {
			flash?.('err', errMsg(e, 'Save failed'));
		} finally { busy = false; }
	}
</script>

<div class="grid gap-6 sm:grid-cols-2 lg:grid-cols-5">
	<div><span class="text-sm text-rig-400">Temperature — {temp}°C</span><Slider min={15} max={35} step={0.5} bind:value={temp} disabled={readOnly} class="mt-3" /></div>
	<div><span class="text-sm text-rig-400">Humidity — {humidity}%</span><Slider min={20} max={90} step={1} bind:value={humidity} disabled={readOnly} class="mt-3" /></div>
	<div><span class="text-sm text-rig-400">CO₂ — {co2 ? `${co2} ppm` : 'off'}</span><Slider min={0} max={1500} step={50} bind:value={co2} disabled={readOnly} class="mt-3" /></div>
	<div><span class="text-sm text-rig-400">Emergency temperature — {emergency}°C</span><Slider min={28} max={45} step={0.5} bind:value={emergency} tone="warn" disabled={readOnly} class="mt-3" /></div>
	<div><span class="text-sm text-rig-400">Leaf temperature offset — {leafOffset > 0 ? '+' : ''}{leafOffset}°C</span><Slider min={-5} max={5} step={0.5} bind:value={leafOffset} disabled={readOnly} class="mt-3" /><p class="mt-2 text-xs text-rig-500">Estimated leaf temp relative to air; −2°C is a common starting point.</p></div>
</div>

<!-- Optional display ranges: the "ok band" shown on climate cards and the
     timeline. Leave a pair at 0 to fall back to the single target above. -->
<div class="mt-6 border-t border-rig-800 pt-5">
	<div class="mb-3 text-xs font-medium uppercase tracking-wide text-rig-500">Target ranges (optional)</div>
	<div class="grid gap-x-6 gap-y-3 sm:grid-cols-2 lg:grid-cols-4">
		{#snippet rangeField(label: string, min: number, max: number, step: number, setMin: (v: number) => void, setMax: (v: number) => void)}
			<div>
				<span class="text-sm text-rig-400">{label}</span>
				<div class="mt-1.5 flex items-center gap-2">
					<input type="number" {step} value={min} oninput={(e) => setMin(+e.currentTarget.value)} disabled={readOnly} placeholder="min" class="{rangeInput}" />
					<span class="text-rig-600">–</span>
					<input type="number" {step} value={max} oninput={(e) => setMax(+e.currentTarget.value)} disabled={readOnly} placeholder="max" class="{rangeInput}" />
				</div>
			</div>
		{/snippet}
		{@render rangeField('Temperature °C', tMin, tMax, 0.5, (v) => (tMin = v), (v) => (tMax = v))}
		{@render rangeField('Humidity %', hMin, hMax, 1, (v) => (hMin = v), (v) => (hMax = v))}
		{@render rangeField('VPD kPa', vMin, vMax, 0.1, (v) => (vMin = v), (v) => (vMax = v))}
		{@render rangeField('CO₂ ppm', cMin, cMax, 50, (v) => (cMin = v), (v) => (cMax = v))}
	</div>
</div>

{#if !readOnly}<div class="mt-5"><Button onclick={saveTargets} disabled={busy}>Save targets</Button></div>{/if}
