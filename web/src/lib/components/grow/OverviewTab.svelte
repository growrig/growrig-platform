<script lang="ts">
	import { live } from '$lib/live.svelte';
	import type {
		Alert,
		CareAction,
		CareHistory,
		DeviceSeries,
		GrowAnalytics,
		GrowDetail,
		GrowPhoto,
		Reading,
		StageLightDefaults,
		Task,
		Weather
	} from '$lib/types';
	import type { StageBand, Annotation } from '$lib/components/timeline/TimelineBody.svelte';
	import { growPhotoImageURL } from '$lib/api';
	import { careVisual } from '$lib/care';
	import { titleCase, vpdZone, toneClass } from '$lib/format';
	import TimelineChart from '$lib/components/TimelineChart.svelte';
	import CareSummary from '$lib/components/CareSummary.svelte';
	import GrowJourney from './GrowJourney.svelte';
	import PhotoUpload from './PhotoUpload.svelte';
	import AttentionRow from '$lib/components/env/AttentionRow.svelte';
	import CircleCheck from '@lucide/svelte/icons/circle-check';
	import ImageIcon from '@lucide/svelte/icons/image';
	import Check from '@lucide/svelte/icons/check';

	interface Props {
		grow: GrowDetail;
		isAdmin: boolean;
		photos: GrowPhoto[];
		care: CareHistory | null;
		careActions: CareAction[];
		analytics: GrowAnalytics | null;
		rangeReadings: Reading[];
		deviceSeries: DeviceSeries[];
		weatherData?: Weather;
		defaults: StageLightDefaults;
		timelineHours: number;
		alerts: Alert[];
		tasks: Task[];
		onRangeChange: (hours: number) => void;
		onMoveStage: (stage: string) => void;
		onPhotoUploaded: () => void;
		onLogCare: () => void;
		onQuickCare: (key: string) => void;
	}
	let {
		grow,
		isAdmin,
		photos,
		care,
		careActions,
		analytics,
		rangeReadings,
		deviceSeries,
		weatherData,
		defaults,
		timelineHours,
		alerts,
		tasks,
		onRangeChange,
		onMoveStage,
		onPhotoUploaded,
		onLogCare,
		onQuickCare
	}: Props = $props();

	const latestPhoto = $derived(photos[0]);

	// The grow's current environment (first active plant's placement), for a
	// compact live readout beside the graph.
	const env = $derived.by(() => {
		const envId = grow.plants.find((p) => p.status === 'active')?.currentEnvironmentId;
		return envId ? live.snapshot?.environments?.find((e) => e.id === envId) : undefined;
	});

	const stagePalette = ['#4ade80', '#38bdf8', '#a78bfa', '#f97316', '#f472b6', '#facc15'];
	const stageBands = $derived.by<StageBand[]>(() => {
		if (!analytics) return [];
		const order = grow.stages;
		return analytics.stageDurations.map((sd) => ({
			from: new Date(sd.from).getTime(),
			to: sd.to ? new Date(sd.to).getTime() : Date.now(),
			label: titleCase(sd.stage),
			color: stagePalette[Math.max(0, order.indexOf(sd.stage)) % stagePalette.length]
		}));
	});
	const careTotalMl = (apps: { amountMl?: number }[] = []) =>
		apps.reduce((n, a) => n + (a.amountMl ?? 0), 0);
	const careAnnotations = $derived.by<Annotation[]>(() =>
		(care?.events ?? []).map((e) => {
			const ml = careTotalMl(e.applications);
			return {
				t: new Date(e.occurredAt).getTime(),
				label: `${careVisual(e.type).label}${ml ? ` · ${ml} ml` : ''}`,
				color: 'var(--color-leaf)'
			};
		})
	);

	const hasToday = $derived(alerts.length > 0 || tasks.length > 0);
</script>

<div class="space-y-6">
	<!-- Hero: latest photo + journey -->
	<div class="grid gap-4 lg:grid-cols-2">
		<div class="overflow-hidden rounded-xl border border-rig-800 bg-rig-950/40">
			{#if latestPhoto}
				<img
					src={growPhotoImageURL(grow.id, latestPhoto.id)}
					alt={latestPhoto.caption || 'Latest grow photo'}
					class="h-full max-h-80 w-full object-cover"
				/>
			{:else}
				<div class="flex h-full min-h-56 flex-col items-center justify-center gap-3 p-6 text-center">
					<ImageIcon size={36} class="text-rig-700" />
					<p class="text-sm text-rig-500">No photos yet</p>
					{#if isAdmin}
						<PhotoUpload
							growId={grow.id}
							onUploaded={onPhotoUploaded}
							triggerClass="inline-flex items-center gap-1.5 rounded-md bg-rig-50 px-4 py-1.5 text-sm font-medium text-rig-950 transition-colors hover:bg-rig-200"
						/>
					{/if}
				</div>
			{/if}
		</div>
		<GrowJourney {grow} {isAdmin} {onMoveStage} />
	</div>

	<!-- Today / attention -->
	<section>
		<h2 class="mb-3 text-sm font-semibold uppercase tracking-wide text-rig-400">Today</h2>
		{#if !hasToday}
			<div class="flex items-center gap-3 rounded-xl border border-rig-800 bg-rig-900/30 p-4 text-sm text-rig-400">
				<CircleCheck size={18} class="text-leaf" /> Everything looks good — nothing scheduled today.
			</div>
		{:else}
			<div class="space-y-2">
				<AttentionRow {alerts} />
				{#each tasks as t (t.id)}
					{@const v = careVisual(t.actionType || 'inspect')}
					<div class="flex items-center gap-3 rounded-lg border border-rig-800 bg-rig-900/40 px-3 py-2.5">
						<span class="grid h-8 w-8 shrink-0 place-items-center rounded-md bg-rig-800 text-rig-300"><v.icon size={16} /></span>
						<div class="min-w-0 flex-1">
							<p class="truncate text-sm text-rig-100">{t.title}</p>
							{#if t.dueAt}<p class="text-xs text-rig-500">Due {new Date(t.dueAt).toLocaleDateString(undefined, { month: 'short', day: 'numeric' })}</p>{/if}
						</div>
						<Check size={15} class="text-rig-600" />
					</div>
				{/each}
			</div>
		{/if}
	</section>

	<!-- Grow data: graph + summaries -->
	<div class="grid gap-6 lg:grid-cols-12">
		<div class="lg:col-span-8">
			<TimelineChart
				readings={rangeReadings}
				{deviceSeries}
				controls={env?.controls ?? []}
				weather={weatherData}
				schedule={env?.schedule}
				stage={grow.stage}
				{defaults}
				hours={timelineHours}
				{stageBands}
				annotations={careAnnotations}
				{onRangeChange}
			/>
		</div>
		<div class="space-y-6 lg:col-span-4">
			{#if care && careActions.length > 0 && grow.plantCount > 0}
				<CareSummary {care} actions={careActions} canWrite={isAdmin} onQuick={onQuickCare} onLog={onLogCare} />
			{/if}
			{#if env}
				<section>
					<h2 class="mb-3 text-sm font-semibold uppercase tracking-wide text-rig-400">Environment</h2>
					<a href="/env/{env.id}" class="block rounded-lg border border-rig-800 bg-rig-950/40 p-4 transition-colors hover:border-rig-600">
						<div class="font-medium">{env.name}</div>
						{#if env.hasClimate}
							<div class="mt-1 text-sm tabular-nums text-rig-300">
								{env.tempC.toFixed(1)}°C · {env.humidity.toFixed(0)}% RH ·
								<span class={toneClass[vpdZone(env.vpd).tone]}>{env.vpd.toFixed(2)} VPD</span>
							</div>
						{:else}
							<div class="mt-1 text-sm text-rig-500">no climate data</div>
						{/if}
						{#if analytics?.pctInTarget != null}
							<div class="mt-1 text-xs text-rig-500">Climate within target {analytics.pctInTarget.toFixed(0)}% of this grow</div>
						{/if}
					</a>
				</section>
			{/if}
		</div>
	</div>
</div>
