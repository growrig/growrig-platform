<script lang="ts">
	import type { GrowDetail } from '$lib/types';
	import { titleCase } from '$lib/format';
	import ArrowRight from '@lucide/svelte/icons/arrow-right';

	interface Props {
		grow: GrowDetail;
		isAdmin: boolean;
		/** Advance to the next stage. */
		onMoveStage: (stage: string) => void;
	}
	let { grow, isAdmin, onMoveStage }: Props = $props();

	const currentIndex = $derived(grow.stages.indexOf(grow.stage));
	const nextStage = $derived(currentIndex >= 0 && currentIndex < grow.stages.length - 1 ? grow.stages[currentIndex + 1] : '');
</script>

<div class="flex h-full flex-col rounded-xl border border-rig-800 bg-rig-900/40 p-5">
	<div class="text-xs uppercase tracking-wide text-rig-500">Grow journey</div>
	<div class="mt-1 text-lg font-semibold capitalize">
		{titleCase(grow.stage) || '—'}
		<span class="text-sm font-normal text-rig-400">· Day {grow.stageDays} in stage</span>
	</div>

	<!-- Stage progression (informational). -->
	<div class="mt-4 flex flex-wrap gap-1.5">
		{#each grow.stages as st, i (st)}
			<span
				class="rounded-full px-2.5 py-0.5 text-xs capitalize {st === grow.stage
					? 'bg-leaf/20 text-leaf'
					: i < currentIndex
						? 'bg-rig-800 text-rig-400'
						: 'bg-rig-800/50 text-rig-500'}"
			>
				{i + 1}. {st}
			</span>
		{/each}
	</div>

	<div class="mt-auto pt-5">
		{#if nextStage}
			<div class="flex items-center justify-between gap-3 border-t border-rig-800 pt-4">
				<div>
					<div class="text-xs text-rig-500">Next stage</div>
					<div class="text-sm font-medium capitalize">{titleCase(nextStage)}</div>
				</div>
				{#if isAdmin && grow.status === 'active'}
					<button
						onclick={() => onMoveStage(nextStage)}
						class="inline-flex items-center gap-1.5 rounded-md border border-rig-700 px-3 py-1.5 text-sm text-rig-200 transition-colors hover:border-leaf/60 hover:text-leaf"
					>
						Move to {titleCase(nextStage)} <ArrowRight size={14} />
					</button>
				{/if}
			</div>
		{/if}
		<div class="mt-3 text-xs text-rig-500">Day {grow.totalDays} overall · {grow.plantCount} plant{grow.plantCount === 1 ? '' : 's'}</div>
	</div>
</div>
