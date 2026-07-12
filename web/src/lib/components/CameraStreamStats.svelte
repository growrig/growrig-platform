<script lang="ts">
	import { onMount } from 'svelte';
	import { getCameraStats, type CameraStats } from '$lib/api';
	interface Props { cameraId: string; class?: string; }
	let { cameraId, class: className = '' }: Props = $props();
	let stats = $state<CameraStats | null>(null);
	function refresh() { getCameraStats(cameraId).then((value) => (stats = value)).catch(() => {}); }
	onMount(() => { refresh(); const timer = setInterval(refresh, 2000); return () => clearInterval(timer); });
	function bitrate(value: number): string {
		if (value >= 1_000_000) return `${(value / 1_000_000).toFixed(1)} Mbps`;
		if (value >= 1_000) return `${Math.round(value / 1_000)} kbps`;
		return `${value} bps`;
	}
</script>
<span class="tabular-nums {className}" title="Measured GrowCore MJPEG relay throughput">{#if stats?.online}{stats.fps.toFixed(1)} FPS · {bitrate(stats.bitrateBps)}{:else}—{/if}</span>
