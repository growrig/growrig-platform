<script lang="ts">
	import type { LightSchedule, StageLightDefaults } from '$lib/types';
	import { nextTransition } from '$lib/photoperiod';
	import { relTime } from '$lib/format';
	import { fmtTime } from '$lib/datetime';
	import Sun from '@lucide/svelte/icons/sun';
	import Moon from '@lucide/svelte/icons/moon';

	interface Props {
		schedule?: LightSchedule;
		defaults: StageLightDefaults;
		stage: string;
	}
	let { schedule, defaults, stage }: Props = $props();

	let nowMs = $state(Date.now());
	$effect(() => {
		const t = setInterval(() => (nowMs = Date.now()), 30_000);
		return () => clearInterval(t);
	});

	const transition = $derived(nextTransition(schedule, stage, defaults, nowMs));
</script>

{#if transition}
	<section>
		<h2 class="mb-3 text-sm font-semibold uppercase tracking-wide text-rig-400">Next transition</h2>
		<div class="flex items-center gap-3 rounded-lg border border-rig-800 bg-rig-950/40 p-3">
			{#if transition.on}
				<Sun size={18} class="shrink-0 text-warn" />
			{:else}
				<Moon size={18} class="shrink-0 text-rig-400" />
			{/if}
			<div>
				<div class="text-sm font-medium">Lights {transition.on ? 'on' : 'off'}</div>
				<div class="text-xs text-rig-400">
					in {relTime(transition.at - nowMs)} · {fmtTime(new Date(transition.at))}
				</div>
			</div>
		</div>
	</section>
{/if}
