<script lang="ts">
	import type { EnvironmentView, Reading } from '$lib/types';
	import type { MetricDescriptor } from '$lib/components/MetricModal.svelte';
	import ClimateDetail from './ClimateDetail.svelte';

	interface Props {
		env: EnvironmentView;
		readings: Reading[];
		rangeReadings: Reading[];
		timelineHours: number;
		onRangeChange: (hours: number) => void;
		onMetric: (descriptor: MetricDescriptor, title: string, unit: string) => void;
	}
	let { env, readings, rangeReadings, timelineHours, onRangeChange, onMetric }: Props = $props();

	const detailReadings = $derived.by(() => {
		if (rangeReadings.length < 2) return readings;
		const cutoff = Date.now() - timelineHours * 3_600_000;
		const sliced = rangeReadings.filter((r) => new Date(r.time).getTime() >= cutoff);
		return sliced.length >= 2 ? sliced : rangeReadings;
	});
</script>

<ClimateDetail
	{env}
	readings={detailReadings}
	{timelineHours}
	{onRangeChange}
	{onMetric}
/>
