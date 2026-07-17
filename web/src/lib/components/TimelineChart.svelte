<script lang="ts">
	import { LayerCake, Svg } from 'layercake';
	import { scaleLinear } from 'd3-scale';
	import type {
		ControlState,
		DeviceSeries,
		LightSchedule,
		StageLightDefaults,
		Reading,
		Weather
	} from '$lib/types';
	import { lightIntervals } from '$lib/photoperiod';
	import { fmtChartHourMinute } from '$lib/datetime';
	import TimelineBody, {
		type SeriesDef,
		type HoverInfo,
		type TargetBand,
		type Annotation,
		type StageBand
	} from './timeline/TimelineBody.svelte';

	interface Props {
		readings: Reading[];
		deviceSeries?: DeviceSeries[];
		controls?: ControlState[];
		weather?: Weather;
		schedule?: LightSchedule;
		stage: string;
		defaults: StageLightDefaults;
		hours?: number;
		futureHours?: number;
		/** Target "ok bands" (drawn only in Climate mode). */
		targetBands?: TargetBand[];
		/** Event markers (stage changes, care, overrides, warnings). */
		annotations?: Annotation[];
		/** Background stage regions (grow graph). */
		stageBands?: StageBand[];
		/** Called when the user picks a range preset, so the parent can widen its
		 *  history fetch to match. */
		onRangeChange?: (hours: number) => void;
		/** Hide the built-in range presets (when the parent owns the control). */
		hideRange?: boolean;
	}
	let {
		readings,
		deviceSeries = [],
		controls = [],
		weather,
		schedule,
		stage,
		defaults,
		hours = 72,
		futureHours = 12,
		targetBands = [],
		annotations = [],
		stageBands = [],
		onRangeChange,
		hideRange = false
	}: Props = $props();

	type Mode = 'climate' | 'equipment' | 'outside' | 'custom';
	const modes: { id: Mode; label: string }[] = [
		{ id: 'climate', label: 'Climate' },
		{ id: 'equipment', label: 'Equipment' },
		{ id: 'outside', label: 'Indoor vs outside' },
		{ id: 'custom', label: 'Custom' }
	];
	let mode = $state<Mode>('climate');

	const rangeOptions = [
		{ h: 6, label: '6h' },
		{ h: 24, label: '24h' },
		{ h: 72, label: '3d' },
		{ h: 168, label: '7d' },
		{ h: 720, label: '30d' }
	];

	// Default-on set (Custom mode's starting point).
	const DEFAULT_ON = new Set(['tempC', 'humidity', 'co2', 'wx:temp', 'wx:rh', 'wx:pressure']);
	let overrides = $state<Record<string, boolean>>({});
	let showLight = $state(true);

	// Which series a given mode shows. Custom defers to the user's overrides.
	function modeOn(key: string): boolean {
		switch (mode) {
			case 'climate':
				return ['tempC', 'humidity', 'vpd', 'co2'].includes(key);
			case 'equipment':
				return key.startsWith('rpm:') || key.startsWith('pw:');
			case 'outside':
				return ['tempC', 'humidity', 'wx:temp', 'wx:rh'].includes(key);
			default:
				return overrides[key] ?? DEFAULT_ON.has(key);
		}
	}
	const isOn = (key: string) => modeOn(key);

	// Toggling a chip drops into Custom mode, seeding the current selection so the
	// one click the user made is the only change.
	function toggleSeries(key: string) {
		if (mode !== 'custom') {
			const seed: Record<string, boolean> = {};
			for (const s of allSeries) seed[s.key] = modeOn(s.key);
			overrides = seed;
			mode = 'custom';
		}
		overrides[key] = !overrides[key];
	}

	const fanPalette = ['#22d3ee', '#14b8a6', '#0ea5e9', '#818cf8'];
	const lightPalette = ['#fbbf24', '#fb923c', '#eab308'];

	const now = $derived(
		readings.length ? Math.max(Date.now(), new Date(readings.at(-1)!.time).getTime()) : Date.now()
	);
	const start = $derived(now - hours * 3_600_000);
	const end = $derived(now + futureHours * 3_600_000);

	function climatePoints(key: 'tempC' | 'humidity' | 'vpd' | 'co2') {
		return readings.map((r) => ({ t: new Date(r.time).getTime(), value: r[key] }));
	}

	// A climate series only counts as "available" if it carries real data — a
	// sensor that isn't configured reports a constant 0, which we hide rather
	// than draw as a flat line.
	const hasData = (pts: { value: number }[]) =>
		pts.some((p) => p.value != null && !Number.isNaN(p.value) && p.value !== 0);

	// The full catalogue of series (climate + per device), each with its data.
	const allSeries = $derived.by<SeriesDef[]>(() => {
		const climate: SeriesDef[] = [
			{ key: 'tempC', label: 'Temp', unit: '°C', color: '#f97316', scaleGroup: 'temp', points: climatePoints('tempC') },
			{ key: 'humidity', label: 'Humidity', unit: '%', color: '#38bdf8', scaleGroup: 'rh', points: climatePoints('humidity') },
			{ key: 'vpd', label: 'VPD', unit: 'kPa', color: '#4ade80', zeroBaseline: true, points: climatePoints('vpd') },
			{ key: 'co2', label: 'CO₂', unit: 'ppm', color: '#a78bfa', zeroBaseline: true, points: climatePoints('co2') }
		].filter((s) => hasData(s.points));
		const nameOf = new Map(controls.map((c) => [c.id, c.name] as const));
		let fanN = 0;
		let lightN = 0;
		// Devices are explicitly configured, so keep them even when idle at 0;
		// zeroBaseline keeps that idle line pinned to the bottom.
		const devices: SeriesDef[] = deviceSeries.map((ds) => {
			const pts = ds.points.map((p) => ({ t: new Date(p.time).getTime(), value: p.value }));
			const name = nameOf.get(ds.bindingId) ?? ds.bindingId;
			if (ds.metric === 'rpm') {
				return { key: `rpm:${ds.bindingId}`, label: name, unit: 'rpm', color: fanPalette[fanN++ % fanPalette.length], zeroBaseline: true, points: pts };
			}
			return { key: `pw:${ds.bindingId}`, label: name, unit: 'W', color: lightPalette[lightN++ % lightPalette.length], zeroBaseline: true, points: pts };
		});
		// Outdoor weather overlays: same hues/scale as indoor climate but faint,
		// solid in the past and dashed into the forecast.
		const outdoor: SeriesDef[] = [];
		if (weather?.temp?.length) {
			outdoor.push({ key: 'wx:temp', label: 'Out °C', unit: '°C', color: '#f97316', opacity: 0.5, dashFrom: now, scaleGroup: 'temp', points: weather.temp.map((p) => ({ t: new Date(p.time).getTime(), value: p.value })) });
		}
		if (weather?.humidity?.length) {
			outdoor.push({ key: 'wx:rh', label: 'Out RH', unit: '%', color: '#38bdf8', opacity: 0.5, dashFrom: now, scaleGroup: 'rh', points: weather.humidity.map((p) => ({ t: new Date(p.time).getTime(), value: p.value })) });
		}
		if (weather?.pressure?.length) {
			outdoor.push({ key: 'wx:pressure', label: 'Pressure', unit: 'hPa', color: '#f472b6', opacity: 0.6, dashFrom: now, points: weather.pressure.map((p) => ({ t: new Date(p.time).getTime(), value: p.value })) });
		}
		return [...climate, ...devices, ...outdoor];
	});

	const activeSeries = $derived(allSeries.filter((s) => isOn(s.key)));
	const intervals = $derived(lightIntervals(schedule, stage, defaults, start, end));

	const padding = { top: 10, right: 14, bottom: 26, left: 14 };

	function fmt(v: number | undefined | null, unit: string): string {
		if (v == null || Number.isNaN(v)) return '—';
		const d = unit === 'kPa' ? 2 : unit === '%' || unit === 'ppm' || unit === 'rpm' || unit === 'W' || unit === 'hPa' ? 0 : 1;
		return `${v.toFixed(d)}${unit === '°C' ? '°' : ''}`;
	}
	const lastOf = (s: SeriesDef) => s.points.at(-1)?.value;

	let hover = $state<HoverInfo | null>(null);
	let wrapW = $state(0);
	const tipStyle = $derived.by(() => {
		if (!hover) return '';
		const cursor = padding.left + hover.x;
		const left = cursor > wrapW / 2 ? cursor - 12 : cursor + 12;
		const anchor = cursor > wrapW / 2 ? 'translateX(-100%)' : 'none';
		return `left:${left}px;transform:${anchor};`;
	});
	const litColor = 'var(--color-warn)';
</script>

<section>
	<div class="mb-3 flex flex-wrap items-center justify-between gap-3">
		<h2 class="text-sm font-semibold uppercase tracking-wide text-rig-400">Timeline</h2>
		<div class="flex flex-wrap items-center gap-2">
			<!-- named modes -->
			<div class="flex rounded-lg border border-rig-800 p-0.5">
				{#each modes as m (m.id)}
					<button
						onclick={() => (mode = m.id)}
						class="rounded-md px-2.5 py-1 text-xs font-medium transition-colors {mode === m.id
							? 'bg-rig-800 text-rig-50'
							: 'text-rig-400 hover:text-rig-100'}"
					>
						{m.label}
					</button>
				{/each}
			</div>
			{#if !hideRange}
				<!-- range presets -->
				<div class="flex rounded-lg border border-rig-800 p-0.5">
					{#each rangeOptions as r (r.h)}
						<button
							onclick={() => onRangeChange?.(r.h)}
							class="rounded-md px-2 py-1 text-xs font-medium tabular-nums transition-colors {hours === r.h
								? 'bg-rig-800 text-rig-50'
								: 'text-rig-400 hover:text-rig-100'}"
						>
							{r.label}
						</button>
					{/each}
				</div>
			{/if}
		</div>
	</div>
	<div class="rounded-xl border border-rig-800 bg-rig-950/40 p-4">
		<div class="relative" bind:clientWidth={wrapW} style="height:240px">
			<LayerCake data={[]} x={(d: { t: number }) => d.t} xScale={scaleLinear()} xDomain={[start, end]} yDomain={[0, 1]} {padding}>
				<Svg>
					{#snippet defs()}
						<pattern id="lc-hatch" width="6" height="6" patternUnits="userSpaceOnUse" patternTransform="rotate(45)">
							<line x1="0" y1="0" x2="0" y2="6" stroke={litColor} stroke-width="1.4" />
						</pattern>
					{/snippet}
					<TimelineBody
						series={activeSeries}
						{intervals}
						{showLight}
						{now}
						targetBands={mode === 'climate' ? targetBands : []}
						{annotations}
						{stageBands}
						onHover={(h) => (hover = h)}
					/>
				</Svg>
			</LayerCake>

			{#if hover}
				<div
					class="pointer-events-none absolute top-2 z-10 min-w-[9rem] rounded-lg border border-rig-700 bg-rig-900/95 px-3 py-2 text-xs shadow-xl backdrop-blur"
					style={tipStyle}
				>
					<div class="mb-1 flex items-center justify-between gap-3 text-rig-400">
						<span>{fmtChartHourMinute(hover.time)}</span>
						<span class="inline-flex items-center gap-1" style="color:{hover.lit ? litColor : 'var(--color-rig-500)'}">
							<span class="inline-block h-2 w-2 rounded-full" style="background:{hover.lit ? litColor : 'var(--color-rig-600)'}"></span>
							{hover.lit ? 'lit' : 'dark'}
						</span>
					</div>
					{#each hover.values as v (v.label + v.unit)}
						<div class="flex items-center justify-between gap-3">
							<span class="inline-flex items-center gap-1.5">
								<span class="inline-block h-2 w-2 rounded-full" style="background:{v.color}"></span>
								{v.label}
							</span>
							<span class="tabular-nums text-rig-100">{fmt(v.value, v.unit)}<span class="text-rig-500"> {v.unit === '°C' ? 'C' : v.unit}</span></span>
						</div>
					{/each}
				</div>
			{/if}
		</div>

		<!-- controls -->
		<div class="mt-3 flex flex-wrap items-center gap-2">
			{#each allSeries as s (s.key)}
				<button
					onclick={() => toggleSeries(s.key)}
					class="inline-flex items-center gap-1.5 rounded-full border px-2.5 py-1 text-xs transition-colors {isOn(s.key) ? 'border-rig-600 bg-rig-800/60 text-rig-100' : 'border-rig-800 text-rig-500 hover:border-rig-700'}"
				>
					<span class="inline-block h-2.5 w-2.5 rounded-full" style="background:{isOn(s.key) ? s.color : 'var(--color-rig-700)'}"></span>
					{s.label}
					<span class="tabular-nums text-rig-400">{fmt(lastOf(s), s.unit)}</span>
				</button>
			{/each}
			<button
				onclick={() => (showLight = !showLight)}
				class="ml-auto inline-flex items-center gap-1.5 rounded-full border px-2.5 py-1 text-xs transition-colors {showLight ? 'border-warn/50 bg-warn/10 text-warn' : 'border-rig-800 text-rig-500 hover:border-rig-700'}"
			>
				<span class="inline-block h-2.5 w-2.5 rounded-sm" style="background:{showLight ? litColor : 'var(--color-rig-700)'}"></span>
				Light
			</button>
		</div>
	</div>
</section>
