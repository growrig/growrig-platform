<script lang="ts">
	import type { EnvironmentView, Reading } from '$lib/types';
	import type { Tone } from '$lib/format';
	import { climateTone, titleCase, vpdZone } from '$lib/format';
	import type { MetricDescriptor } from '$lib/components/MetricModal.svelte';
	import ClimateMetricRow from './ClimateMetricRow.svelte';
	import Info from '@lucide/svelte/icons/info';
	import Leaf from '@lucide/svelte/icons/leaf';

	interface Props {
		env: EnvironmentView;
		readings: Reading[];
		timelineHours: number;
		onRangeChange: (hours: number) => void;
		onMetric: (descriptor: MetricDescriptor, title: string, unit: string) => void;
	}
	let { env, readings, timelineHours, onRangeChange, onMetric }: Props = $props();

	const rangeOptions = [
		{ h: 6, label: '6h' },
		{ h: 24, label: '24h' },
		{ h: 72, label: '3d' },
		{ h: 168, label: '7d' },
		{ h: 720, label: '30d' }
	];

	const provenance = $derived.by(() => {
		if (!env.grow) return '';
		const stage = env.grow.stage ? ` · ${titleCase(env.grow.stage)}` : '';
		return `Target from ${env.grow.name}${stage}`;
	});

	function pointsOf(pick: (r: Reading) => number, ok: (r: Reading) => boolean) {
		return readings
			.filter(ok)
			.map((r) => ({ t: new Date(r.time).getTime(), v: pick(r) }))
			.filter((p) => Number.isFinite(p.v));
	}

	function rangeTone(v: number, min?: number, max?: number): Tone {
		if (!min || !max || max <= min) return 'muted';
		if (v >= min && v <= max) return 'good';
		const span = max - min;
		if (v < min - span * 0.5 || v > max + span * 0.5) return 'danger';
		return 'warn';
	}

	function statusLabel(tone: Tone, current: number | undefined, min?: number, max?: number, target?: number): string {
		if (current == null || tone === 'muted') return '';
		if (tone === 'good') return 'Good';
		if (min && max && max > min) {
			if (current > max) return 'High';
			if (current < min) return 'Low';
		}
		if (target && target > 0) {
			if (current > target) return 'High';
			if (current < target) return 'Low';
		}
		return tone === 'danger' ? 'Alert' : 'Moderate';
	}

	function setpointTone(v: number, target: number, soft: number, hard: number): Tone {
		if (!target) return 'muted';
		const d = Math.abs(v - target);
		if (d <= soft) return 'good';
		if (d <= hard) return 'warn';
		return 'danger';
	}

	const tempClassify = (v: number): Tone => {
		if (env.emergencyTempC > 0 && v >= env.emergencyTempC) return 'danger';
		if (env.targetTempMinC && env.targetTempMaxC)
			return rangeTone(v, env.targetTempMinC, env.targetTempMaxC);
		return climateTone(v, env.targetTempC, env.emergencyTempC);
	};
	const humClassify = (v: number): Tone => {
		if (env.targetHumidityMin && env.targetHumidityMax)
			return rangeTone(v, env.targetHumidityMin, env.targetHumidityMax);
		return setpointTone(v, env.targetHumidity, 5, 15);
	};
	const co2Classify = (v: number): Tone => {
		if (env.targetCo2Min && env.targetCo2Max) return rangeTone(v, env.targetCo2Min, env.targetCo2Max);
		return 'muted';
	};
	const vpdClassify = (v: number): Tone => {
		if (env.targetVpdMin && env.targetVpdMax) return rangeTone(v, env.targetVpdMin, env.targetVpdMax);
		return vpdZone(v).tone;
	};

	const tempPts = $derived(pointsOf((r) => r.tempC, () => true));
	const humPts = $derived(pointsOf((r) => r.humidity, () => true));
	const co2Pts = $derived(pointsOf((r) => r.co2, (r) => r.co2 > 0));
	const vpdPts = $derived(pointsOf((r) => r.vpd, (r) => r.vpd > 0));

	const tempTone = $derived(env.hasTemp ? tempClassify(env.tempC) : ('muted' as Tone));
	const humTone = $derived(env.hasHum ? humClassify(env.humidity) : ('muted' as Tone));
	const co2Tone = $derived(env.hasCO2 ? co2Classify(env.co2) : ('muted' as Tone));
	const vpdTone = $derived(env.hasClimate ? vpdClassify(env.vpd) : ('muted' as Tone));

	const overall = $derived.by(() => {
		const tones = [tempTone, humTone, vpdTone];
		if (env.hasCO2) tones.push(co2Tone);
		if (tones.includes('danger')) return { label: 'Alert', tone: 'danger' as Tone };
		if (tones.includes('warn')) return { label: 'Moderate', tone: 'warn' as Tone };
		if (tones.every((t) => t === 'muted')) return { label: 'No data', tone: 'muted' as Tone };
		return { label: 'Good', tone: 'good' as Tone };
	});

	const advice = $derived.by(() => {
		const items: { title: string; detail: string }[] = [];
		if (env.hasTemp && tempTone !== 'good' && tempTone !== 'muted') {
			const detail =
				env.targetTempMinC && env.targetTempMaxC
					? `${env.tempC.toFixed(1)} °C · target ${env.targetTempMinC}–${env.targetTempMaxC} °C`
					: env.targetTempC
						? `${env.tempC.toFixed(1)} °C · target ${env.targetTempC} °C`
						: `${env.tempC.toFixed(1)} °C`;
			items.push({
				title: env.tempC > (env.targetTempC || env.targetTempMaxC || 0) ? 'Temperature high' : 'Temperature low',
				detail
			});
		}
		if (env.hasHum && humTone !== 'good' && humTone !== 'muted') {
			items.push({
				title: env.humidity > (env.targetHumidityMax || env.targetHumidity || 0) ? 'Humidity high' : 'Humidity low',
				detail:
					env.targetHumidityMin && env.targetHumidityMax
						? `${env.humidity.toFixed(0)} % · target ${env.targetHumidityMin}–${env.targetHumidityMax} %`
						: `${env.humidity.toFixed(0)} %`
			});
		}
		if (env.hasCO2 && co2Tone !== 'good' && co2Tone !== 'muted') {
			items.push({
				title: env.co2 > (env.targetCo2Max || 0) ? 'CO₂ high — consider ventilating' : 'CO₂ low',
				detail:
					env.targetCo2Min && env.targetCo2Max
						? `${env.co2.toFixed(0)} ppm · target ${env.targetCo2Min}–${env.targetCo2Max} ppm`
						: `${env.co2.toFixed(0)} ppm`
			});
		}
		if (env.hasClimate && vpdTone !== 'good' && vpdTone !== 'muted') {
			const z = vpdZone(env.vpd);
			items.push({
				title: z.label,
				detail:
					env.targetVpdMin && env.targetVpdMax
						? `${env.vpd.toFixed(2)} kPa · target ${env.targetVpdMin}–${env.targetVpdMax} kPa`
						: `${env.vpd.toFixed(2)} kPa`
			});
		}
		return items[0] ?? null;
	});

	const overallPill: Record<Tone, string> = {
		good: 'border-leaf/40 text-leaf',
		warn: 'border-warn/50 text-warn',
		danger: 'border-danger/50 text-danger',
		muted: 'border-rig-700 text-rig-400'
	};
	const adviceBg: Record<Tone, string> = {
		good: 'bg-leaf/10 border-leaf/20',
		warn: 'bg-warn/10 border-warn/25',
		danger: 'bg-danger/10 border-danger/25',
		muted: 'bg-rig-900 border-rig-800'
	};
</script>

<section>
	<div class="mb-3 flex flex-wrap items-center justify-between gap-3">
		<div class="flex flex-wrap items-center gap-2.5">
			<h2 class="text-sm font-semibold uppercase tracking-wide text-rig-400">Climate now</h2>
			{#if provenance}
				<span class="text-xs text-rig-500">{provenance}</span>
			{/if}
			<span
				class="inline-flex items-center gap-1.5 rounded-full border px-2.5 py-1 text-xs font-medium {overallPill[overall.tone]}"
			>
				<Leaf size={13} />
				{overall.label}
			</span>
		</div>
		<div class="flex rounded-lg border border-rig-800 p-0.5">
			{#each rangeOptions as r (r.h)}
				<button
					onclick={() => onRangeChange(r.h)}
					class="rounded-md px-2.5 py-1 text-xs font-medium tabular-nums transition-colors {timelineHours === r.h
						? 'bg-rig-800 text-rig-50'
						: 'text-rig-400 hover:text-rig-100'}"
				>
					{r.label}
				</button>
			{/each}
		</div>
	</div>

	{#if advice}
		<div class="mb-3 flex items-start gap-3 rounded-xl border px-3.5 py-3 {adviceBg[overall.tone]}">
			<span
				class="mt-0.5 flex size-5 shrink-0 items-center justify-center rounded-full {overall.tone === 'danger'
					? 'bg-danger text-rig-950'
					: overall.tone === 'warn'
						? 'bg-warn text-rig-950'
						: 'bg-leaf text-rig-950'}"
			>
				<Info size={12} />
			</span>
			<div>
				<p class="text-sm font-semibold text-rig-100">{advice.title}</p>
				<p class="text-xs text-rig-400">{advice.detail}</p>
			</div>
		</div>
	{/if}

	<div class="space-y-3">
		<ClimateMetricRow
			label="Temperature"
			value={env.hasTemp ? env.tempC.toFixed(1) : '—'}
			unit="°C"
			tone={tempTone}
			status={statusLabel(tempTone, env.hasTemp ? env.tempC : undefined, env.targetTempMinC, env.targetTempMaxC, env.targetTempC)}
			points={tempPts}
			classify={tempClassify}
			onclick={() => onMetric({ kind: 'sensor', measurement: 'temperature' }, 'Temperature', '°C')}
		/>
		<ClimateMetricRow
			label="Humidity"
			value={env.hasHum ? env.humidity.toFixed(0) : '—'}
			unit="%"
			tone={humTone}
			status={statusLabel(humTone, env.hasHum ? env.humidity : undefined, env.targetHumidityMin, env.targetHumidityMax, env.targetHumidity)}
			points={humPts}
			classify={humClassify}
			onclick={() => onMetric({ kind: 'sensor', measurement: 'humidity' }, 'Humidity', '%')}
		/>
		{#if env.hasCO2}
			<ClimateMetricRow
				label="CO₂"
				value={env.co2.toFixed(0)}
				unit="ppm"
				tone={co2Tone}
				status={statusLabel(co2Tone, env.co2, env.targetCo2Min, env.targetCo2Max)}
				points={co2Pts}
				classify={co2Classify}
				onclick={() => onMetric({ kind: 'sensor', measurement: 'co2' }, 'CO₂', 'ppm')}
			/>
		{/if}
		<ClimateMetricRow
			label="VPD"
			value={env.hasClimate ? env.vpd.toFixed(2) : '—'}
			unit="kPa"
			tone={vpdTone}
			status={env.hasClimate ? (vpdTone === 'good' ? 'Good' : vpdZone(env.vpd).label) : ''}
			points={vpdPts}
			classify={vpdClassify}
			onclick={() => onMetric({ kind: 'vpd' }, 'VPD', 'kPa')}
		/>
	</div>
</section>
