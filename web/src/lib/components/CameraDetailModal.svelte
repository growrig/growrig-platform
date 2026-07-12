<script lang="ts">
	import { getCameraSnapshots, cameraArchiveURL, cameraProxyURL, type CameraSnapshot } from '$lib/api';
	import type { CameraRef } from '$lib/types';
	import { Dialog } from '$lib/components/ui';
	import CameraPreview from './CameraPreview.svelte';
	import Radio from '@lucide/svelte/icons/radio';

	interface Props { open?: boolean; camera: CameraRef; }
	let { open = $bindable(false), camera }: Props = $props();
	let snapshots = $state<CameraSnapshot[]>([]);
	let selected = $state<CameraSnapshot | null>(null);
	let loading = $state(false);
	const canUseRecordedView = $derived(camera.cameraType === 'rtsp' || (!camera.streamUrl && !camera.entity));

	$effect(() => {
		if (!open || !canUseRecordedView) return;
		void camera.id;
		selected = null;
		loading = true;
		getCameraSnapshots(camera.id)
			.then((items) => (snapshots = items))
			.catch(() => (snapshots = []))
			.finally(() => (loading = false));
	});

	const selectedURL = $derived(selected ? cameraArchiveURL(camera.id, selected.id) : cameraProxyURL(camera.id));
	const formatTime = (value: string) => new Intl.DateTimeFormat(undefined, { month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit', second: '2-digit' }).format(new Date(value));
</script>

<Dialog bind:open title={camera.name} description="Live camera and recorded snapshots" size="3xl">
	<div class="space-y-4">
		<div class="relative">
			<CameraPreview url={selectedURL} liveUrl={!selected && canUseRecordedView ? cameraProxyURL(camera.id, true) : ''} type={selected ? 'mjpeg' : canUseRecordedView ? 'snapshot' : camera.cameraType} class="border-rig-700" emptyLabel="Connecting to camera…" errorLabel="Connecting to camera…" />
			{#if selected}
				<button type="button" onclick={() => (selected = null)} class="absolute right-3 top-3 inline-flex items-center gap-1.5 rounded-md bg-rig-950/85 px-3 py-1.5 text-xs font-medium text-rig-100 shadow hover:bg-rig-800"><Radio size={14} class="text-leaf" /> Return to live</button>
			{:else if canUseRecordedView}
				<span class="absolute right-3 top-3 inline-flex items-center gap-1.5 rounded-md bg-rig-950/75 px-2.5 py-1 text-xs text-leaf"><span class="h-1.5 w-1.5 animate-pulse rounded-full bg-leaf"></span> Live</span>
			{/if}
		</div>

		<div>
			<div class="mb-2 flex items-center justify-between"><h3 class="text-sm font-semibold uppercase tracking-wide text-rig-400">Snapshot timeline</h3><span class="text-xs text-rig-500">{snapshots.length} most recent</span></div>
			{#if loading}
				<div class="py-6 text-center text-sm text-rig-500">Loading snapshots…</div>
			{:else if snapshots.length}
				<div class="flex gap-2 overflow-x-auto pb-2">
					{#each snapshots as snapshot (snapshot.id)}
						<button type="button" onclick={() => (selected = snapshot)} class="w-32 shrink-0 overflow-hidden rounded-lg border text-left transition-colors {selected?.id === snapshot.id ? 'border-leaf ring-1 ring-leaf/40' : 'border-rig-800 hover:border-rig-500'}">
							<img src={cameraArchiveURL(camera.id, snapshot.id)} alt="Snapshot at {formatTime(snapshot.time)}" loading="lazy" class="aspect-video w-full object-cover" />
							<span class="block truncate px-2 py-1.5 text-[10px] text-rig-400">{formatTime(snapshot.time)}</span>
						</button>
					{/each}
				</div>
			{:else}
				<div class="rounded-lg border border-dashed border-rig-800 py-6 text-center text-sm text-rig-500">No archived snapshots yet.</div>
			{/if}
		</div>
	</div>
</Dialog>
