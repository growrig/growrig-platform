<script lang="ts">
	import type { CameraType } from '$lib/types';
	import CameraOff from '@lucide/svelte/icons/camera-off';

	interface Props {
		url: string;
		/** Optional MJPEG feed layered over the initial snapshot once ready. */
		liveUrl?: string;
		type?: CameraType;
		/** Snapshot refresh interval in seconds. */
		refreshSeconds?: number;
		class?: string;
		emptyLabel?: string;
		errorLabel?: string;
	}
	let { url, liveUrl = '', type = 'snapshot', refreshSeconds = 2, class: className = '', emptyLabel = 'No stream URL', errorLabel = 'No signal' }: Props = $props();

	let tick = $state(0);
	let failed = $state(false);
	let liveReady = $state(false);

	// Snapshot cameras return a single JPEG, so we re-request on an interval with a
	// cache-busting param. MJPEG streams play continuously, so no refresh is needed.
	$effect(() => {
		if (type !== 'snapshot' || !url) return;
		const h = setInterval(() => (tick = tick + 1), Math.max(1, refreshSeconds) * 1000);
		return () => clearInterval(h);
	});

	// Reset the error state whenever the source changes.
	$effect(() => {
		void url;
		void type;
		// A cached RTSP camera may not have produced its first frame yet. Retry
		// snapshot failures on the next scheduled refresh instead of remaining in
		// the no-signal state forever.
		if (type === 'snapshot') void tick;
		failed = false;
	});
	$effect(() => { void liveUrl; liveReady = false; });

	const src = $derived(
		type === 'snapshot' && tick > 0 ? `${url}${url.includes('?') ? '&' : '?'}_t=${tick}` : url
	);
</script>

<div
	class="relative aspect-video w-full overflow-hidden rounded-lg border border-rig-800 bg-rig-950 {className}"
>
	{#if url && !failed}
		<!-- svelte-ignore a11y_missing_attribute -->
		<img src={src} class="h-full w-full object-cover" onerror={() => (failed = true)} />
	{:else}
		<div class="flex h-full w-full flex-col items-center justify-center gap-1 text-rig-600">
			<CameraOff size={22} />
			<span class="text-xs">{url ? errorLabel : emptyLabel}</span>
		</div>
	{/if}
	{#if liveUrl}
		<!-- Keep the cached frame visible until the MJPEG response produces its first frame. -->
		<img src={liveUrl} alt="Live camera" onload={() => (liveReady = true)} class="absolute inset-0 h-full w-full object-cover transition-opacity duration-300 {liveReady ? 'opacity-100' : 'opacity-0'}" />
	{/if}
</div>
