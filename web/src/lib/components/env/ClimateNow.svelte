<script lang="ts">
	import type { EnvironmentView, Reading } from '$lib/types';
	import { climateTone, titleCase, vpdZone } from '$lib/format';
	import type { MetricDescriptor } from '$lib/components/MetricModal.svelte';
	import ClimateCard from './ClimateCard.svelte';

	interface Props {
		env: EnvironmentView;
		readings: Reading[];
		onMetric: (descriptor: MetricDescriptor, title: string, unit: string) => void;
	}
	let { env, readings, onMetric }: Props = $props();

	// Provenance: when a control grow drives this environment, its (occupancy-
	// sourced) stage explains where the targets come from — and keeps the stage
	// label consistent across the page.
	const provenance = $derived.by(() => {
		if (!env.grow) return '';
		const stage = env.grow.stage ? ` · ${titleCase(env.grow.stage)}` : '';
		return `Target from ${env.grow.name}${stage}`;
	});
</script>

<section>
	<div class="mb-3 flex items-center justify-between gap-3">
		<h2 class="text-sm font-semibold uppercase tracking-wide text-rig-400">Climate now</h2>
		{#if provenance}<span class="text-xs text-rig-500">{provenance}</span>{/if}
	</div>

	<div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
		<ClimateCard
			label="Temperature"
			value={env.hasTemp ? env.tempC.toFixed(1) : '—'}
			unit="°C"
			current={env.hasTemp ? env.tempC : undefined}
			target={env.targetTempC}
			rangeMin={env.targetTempMinC}
			rangeMax={env.targetTempMaxC}
			tone={env.hasTemp ? climateTone(env.tempC, env.targetTempC, env.emergencyTempC) : 'muted'}
			spark={env.hasTemp ? readings.map((r) => r.tempC) : undefined}
			sparkColor="#f97316"
			onclick={() => onMetric({ kind: 'sensor', measurement: 'temperature' }, 'Temperature', '°C')}
		/>
		<ClimateCard
			label="Humidity"
			value={env.hasHum ? env.humidity.toFixed(0) : '—'}
			unit="%"
			current={env.hasHum ? env.humidity : undefined}
			target={env.targetHumidity}
			rangeMin={env.targetHumidityMin}
			rangeMax={env.targetHumidityMax}
			spark={env.hasHum ? readings.map((r) => r.humidity) : undefined}
			sparkColor="#38bdf8"
			onclick={() => onMetric({ kind: 'sensor', measurement: 'humidity' }, 'Humidity', '%')}
		/>
		{#if env.hasCO2}
			<ClimateCard
				label="CO₂"
				value={env.co2.toFixed(0)}
				unit="ppm"
				current={env.co2}
				rangeMin={env.targetCo2Min}
				rangeMax={env.targetCo2Max}
				spark={readings.map((r) => r.co2)}
				sparkColor="#a78bfa"
				onclick={() => onMetric({ kind: 'sensor', measurement: 'co2' }, 'CO₂', 'ppm')}
			/>
		{/if}
		<ClimateCard
			label="VPD"
			value={env.hasClimate ? env.vpd.toFixed(2) : '—'}
			unit="kPa"
			current={env.hasClimate ? env.vpd : undefined}
			rangeMin={env.targetVpdMin}
			rangeMax={env.targetVpdMax}
			tone={env.hasClimate ? vpdZone(env.vpd).tone : 'muted'}
			onclick={() => onMetric({ kind: 'vpd' }, 'VPD', 'kPa')}
		/>
	</div>
</section>
