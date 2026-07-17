<script lang="ts">
	import type {
		Alert,
		CameraRef,
		DeviceSeries,
		EnvironmentView,
		Grow,
		Location,
		Reading,
		StageLightDefaults,
		Weather
	} from '$lib/types';
	import type { MetricDescriptor } from '$lib/components/MetricModal.svelte';
	import type { TargetBand, Annotation } from '$lib/components/timeline/TimelineBody.svelte';
	import { cameraProxyURL } from '$lib/api';
	import TimelineChart from '$lib/components/TimelineChart.svelte';
	import CameraPreview from '$lib/components/CameraPreview.svelte';
	import CameraDetailModal from '$lib/components/CameraDetailModal.svelte';
	import ActivityLog from '$lib/components/ActivityLog.svelte';
	import GrowingHere from './GrowingHere.svelte';
	import AttentionRow from './AttentionRow.svelte';
	import ClimateNow from './ClimateNow.svelte';
	import OperatingNow from './OperatingNow.svelte';
	import NextTransition from './NextTransition.svelte';
	import Camera from '@lucide/svelte/icons/camera';

	interface Props {
		env: EnvironmentView;
		readings: Reading[];
		rangeReadings: Reading[];
		deviceSeries: DeviceSeries[];
		weatherData?: Weather;
		grows: Grow[];
		defaults: StageLightDefaults;
		locations: Location[];
		alerts: Alert[];
		timelineHours: number;
		targetBands: TargetBand[];
		annotations: Annotation[];
		onRangeChange: (hours: number) => void;
		onMetric: (descriptor: MetricDescriptor, title: string, unit: string) => void;
	}
	let {
		env,
		readings,
		rangeReadings,
		deviceSeries,
		weatherData,
		grows,
		defaults,
		alerts,
		timelineHours,
		targetBands,
		annotations,
		onRangeChange,
		onMetric
	}: Props = $props();

	const stage = $derived(env.grow?.stage ?? '');
	const firstCamera = $derived<CameraRef | undefined>(env.cameras?.[0]);

	let cameraOpen = $state(false);
</script>

<div class="space-y-6">
	<GrowingHere {env} />

	<AttentionRow {alerts} />

	{#if env.kind === 'tent'}
		<ClimateNow {env} {readings} {onMetric} />
	{/if}

	<!-- Main area: timeline + notable events (8) · camera + system response (4). -->
	<div class="grid gap-6 lg:grid-cols-12">
		<div class="space-y-6 lg:col-span-8">
			<TimelineChart
				readings={rangeReadings}
				{deviceSeries}
				controls={env.controls ?? []}
				weather={weatherData}
				schedule={env.schedule}
				{stage}
				{defaults}
				hours={timelineHours}
				{targetBands}
				{annotations}
				{onRangeChange}
			/>
			<ActivityLog environmentId={env.id} limit={8} title="Recent activity" />
		</div>

		<div class="space-y-6 lg:col-span-4">
			{#if firstCamera}
				<section>
					<h2 class="mb-3 text-sm font-semibold uppercase tracking-wide text-rig-400">Camera</h2>
					<div
						role="button" tabindex="0"
						onclick={() => (cameraOpen = true)}
						onkeydown={(e) => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); cameraOpen = true; } }}
						class="group cursor-pointer overflow-hidden rounded-lg border border-rig-800 bg-rig-950/40 transition-colors hover:border-rig-600 focus-visible:border-rig-500 focus-visible:outline-none"
					>
						<CameraPreview
							url={firstCamera.cameraType === 'rtsp' || firstCamera.entity || !firstCamera.streamUrl ? cameraProxyURL(firstCamera.id) : firstCamera.streamUrl}
							type={firstCamera.cameraType === 'rtsp' ? 'snapshot' : firstCamera.streamUrl ? firstCamera.cameraType : 'snapshot'}
							refreshSeconds={firstCamera.cameraType === 'rtsp' ? firstCamera.cameraCaptureInterval ?? 60 : 2}
							class="border-0"
							emptyLabel="Camera connecting…"
							errorLabel="Camera connecting…"
						/>
						<div class="flex items-center gap-2 px-3 py-2 text-sm">
							<Camera size={15} class="text-rig-400" />
							<span class="transition-colors group-hover:text-leaf">{firstCamera.name}</span>
						</div>
					</div>
				</section>
			{/if}

			<OperatingNow {env} schedule={env.schedule} {defaults} {stage} />
			<NextTransition schedule={env.schedule} {defaults} {stage} />
		</div>
	</div>
</div>

{#if firstCamera}
	<CameraDetailModal bind:open={cameraOpen} camera={firstCamera} />
{/if}
