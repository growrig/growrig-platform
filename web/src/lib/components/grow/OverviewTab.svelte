<script lang="ts">
	import { goto } from '$app/navigation';
	import { live } from '$lib/live.svelte';
	import type {
		Alert,
		CareAction,
		CareHistory,
		GrowAnalytics,
		GrowDetail,
		GrowPhoto,
		Task
	} from '$lib/types';
	import { growPhotoImageURL } from '$lib/api';
	import { careVisual } from '$lib/care';
	import { vpdZone, toneClass, plantDisplayName, plantNumbersById, daysSince } from '$lib/format';
	import CareHeatmap from './CareHeatmap.svelte';
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
		alerts: Alert[];
		tasks: Task[];
		onMoveStage: (stage: string) => void;
		onPhotoUploaded: () => void;
	}
	let {
		grow,
		isAdmin,
		photos,
		care,
		careActions,
		analytics,
		alerts,
		tasks,
		onMoveStage,
		onPhotoUploaded
	}: Props = $props();

	const latestPhoto = $derived(photos[0]);

	// The grow's current environment (first active plant's placement), for a
	// compact live readout beside the graph.
	const env = $derived.by(() => {
		const envId = grow.plants.find((p) => p.status === 'active')?.currentEnvironmentId;
		return envId ? live.snapshot?.environments?.find((e) => e.id === envId) : undefined;
	});

	const hasToday = $derived(alerts.length > 0 || tasks.length > 0);

	const plantNumbers = $derived(plantNumbersById(grow.plants));
	const statusTone = (s: string) =>
		s === 'active' ? 'text-leaf' : s === 'harvested' ? 'text-warn' : 'text-rig-500';
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

	<!-- Grow timeline (care heatmap, full width) -->
	<CareHeatmap {grow} events={care?.events ?? []} stages={analytics?.stageDurations ?? []} />

	<!-- Plants -->
	{#if grow.plants.length}
		<section>
			<div class="mb-3 flex items-center justify-between">
				<h2 class="text-sm font-semibold uppercase tracking-wide text-rig-400">Plants · {grow.plantCount} active</h2>
				<a href="/grows/{grow.id}?tab=plants" class="text-xs text-rig-400 hover:text-rig-100">Manage plants →</a>
			</div>
			<div class="overflow-x-auto rounded-xl border border-rig-800">
				<table class="w-full min-w-[36rem] text-sm">
					<thead class="border-b border-rig-800 text-left text-xs uppercase tracking-wide text-rig-500">
						<tr>
							<th class="px-4 py-2 font-medium">Plant</th>
							<th class="px-4 py-2 font-medium">Cultivar</th>
							<th class="px-4 py-2 font-medium">Status</th>
							<th class="px-4 py-2 font-medium">Location</th>
							<th class="px-4 py-2 font-medium">Pot</th>
							<th class="px-4 py-2 font-medium">Age</th>
						</tr>
					</thead>
					<tbody>
						{#each grow.plants as p (p.id)}
							<tr
								class="cursor-pointer border-b border-rig-800/60 transition-colors last:border-0 hover:bg-rig-800/40"
								onclick={() => goto(`/plants/${p.id}`)}
							>
								<td class="px-4 py-2"><a href="/plants/{p.id}" class="font-medium hover:text-rig-50" onclick={(e) => e.stopPropagation()}>{plantDisplayName(p, plantNumbers.get(p.id))}</a>{#if p.tracking === 'group' && p.quantity > 1}<span class="ml-1 text-xs text-rig-500">×{p.quantity}</span>{/if}</td>
								<td class="px-4 py-2 text-rig-300">{p.cultivar || '—'}</td>
								<td class="px-4 py-2 capitalize {statusTone(p.status)}">{p.status}</td>
								<td class="px-4 py-2 text-rig-300">{#if p.currentEnvironmentId}<a href="/env/{p.currentEnvironmentId}" class="hover:text-rig-100 hover:underline" onclick={(e) => e.stopPropagation()}>{p.currentEnvironmentName || p.currentEnvironmentId}</a>{:else}—{/if}</td>
								<td class="px-4 py-2 tabular-nums text-rig-300">{p.currentPot ? `${p.currentPot.size} ${p.currentPot.unit}${p.currentPot.type ? ` ${p.currentPot.type}` : ''}` : '—'}</td>
								<td class="px-4 py-2 tabular-nums text-rig-400">{daysSince(p.createdAt)}d</td>
							</tr>
						{/each}
					</tbody>
				</table>
			</div>
		</section>
	{/if}

	<!-- Care summaries + environment -->
	<div class="grid gap-6 lg:grid-cols-2">
		<div class="space-y-6">
			{#if care && careActions.length > 0 && grow.plantCount > 0}
				<CareSummary {care} />
			{/if}
		</div>
		<div class="space-y-6">
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
