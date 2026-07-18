<script lang="ts" module>
	export type { MetricDescriptor, MetricListItem } from '$lib/components/MetricGraph.svelte';
</script>

<script lang="ts">
	import { Dialog } from '$lib/components/ui';
	import MetricGraph, { metricNote, type MetricDescriptor } from '$lib/components/MetricGraph.svelte';
	import type { ControlState, SensorReading } from '$lib/types';

	interface Props {
		open?: boolean;
		envId: string;
		title: string;
		unit: string;
		descriptor: MetricDescriptor;
		/** Live sensors of this measurement (labels + current readings). */
		sensors?: SensorReading[];
		/** Live controls (device current value). */
		controls?: ControlState[];
		vpdCurrent?: number | null;
		vpdTempC?: number | null;
		vpdHumidity?: number | null;
		vpdLeafTempOffsetC?: number;
	}
	let {
		open = $bindable(false),
		envId,
		title,
		unit,
		descriptor,
		sensors = [],
		controls = [],
		vpdCurrent = null,
		vpdTempC = null,
		vpdHumidity = null,
		vpdLeafTempOffsetC = -2
	}: Props = $props();

	const note = $derived(metricNote(descriptor, vpdLeafTempOffsetC));
</script>

<Dialog bind:open {title} description={note} size="2xl">
	<MetricGraph
		active={open}
		{envId}
		{unit}
		{descriptor}
		{sensors}
		{controls}
		{vpdCurrent}
		{vpdTempC}
		{vpdHumidity}
		{vpdLeafTempOffsetC}
	/>
</Dialog>
