<script lang="ts">
	import type { CareAction, CareHistory } from '$lib/types';
	import { Button } from '$lib/components/ui';
	import Droplet from '@lucide/svelte/icons/droplet';
	import TriangleAlert from '@lucide/svelte/icons/triangle-alert';

	interface Props {
		care: CareHistory;
		actions: CareAction[];
		canWrite?: boolean;
		/** Open the log-care dialog straight into this action (e.g. water/feed all). */
		onQuick?: (actionKey: string) => void;
		/** Open the full log-care dialog (action picker). */
		onLog?: () => void;
	}
	let { care, actions, canWrite = false, onQuick, onLog }: Props = $props();

	const label = (key: string) => actions.find((a) => a.key === key)?.label ?? key;

	// A compact "2d ago" / "5h ago" / "just now" from an ISO timestamp.
	function ago(iso: string): string {
		const ms = Date.now() - new Date(iso).getTime();
		if (ms < 60_000) return 'just now';
		const min = Math.floor(ms / 60_000);
		if (min < 60) return `${min}m ago`;
		const h = Math.floor(min / 60);
		if (h < 24) return `${h}h ago`;
		return `${Math.floor(h / 24)}d ago`;
	}

	// Show the most recent action per type, most-recent first. Water and feed
	// lead since they're the everyday actions.
	const lastEntries = $derived(
		Object.values(care.summary.lastByType).sort(
			(a, b) => new Date(b.occurredAt).getTime() - new Date(a.occurredAt).getTime()
		)
	);
	const hasWater = $derived(actions.some((a) => a.key === 'water'));
	const hasFeed = $derived(actions.some((a) => a.key === 'feed'));
	const skipped = $derived(care.summary.skipped);
</script>

<section class="rounded-xl border border-rig-800 bg-rig-900/40 p-4">
	<div class="mb-3 flex items-center justify-between">
		<h2 class="flex items-center gap-1.5 text-sm font-semibold uppercase tracking-wide text-rig-400">
			<Droplet size={14} class="text-sky-400" /> Care
		</h2>
		{#if canWrite}
			<div class="flex flex-wrap gap-2">
				{#if hasWater}<Button size="sm" variant="secondary" onclick={() => onQuick?.('water')}>Water all</Button>{/if}
				{#if hasFeed}<Button size="sm" variant="secondary" onclick={() => onQuick?.('feed')}>Feed all</Button>{/if}
				<Button size="sm" onclick={() => onLog?.()}>Log care</Button>
			</div>
		{/if}
	</div>

	{#if lastEntries.length === 0}
		<p class="text-sm text-rig-500">No care logged yet.{#if canWrite} Use the actions above to start the grow journal.{/if}</p>
	{:else}
		<div class="flex flex-wrap gap-x-6 gap-y-1.5 text-sm">
			{#each lastEntries as e (e.type)}
				<div class="flex items-baseline gap-1.5">
					<span class="text-rig-400">Last {label(e.type).toLowerCase()}</span>
					<span class="font-medium text-rig-100">{ago(e.occurredAt)}</span>
					{#if e.type === 'water' || e.type === 'feed'}
						{@const total = (e.applications ?? []).reduce((s, a) => s + (a.amountMl ?? 0), 0)}
						{#if total > 0}<span class="text-xs text-rig-500">· {total >= 1000 ? `${(total / 1000).toFixed(1)} L` : `${total} ml`}</span>{/if}
					{/if}
				</div>
			{/each}
		</div>
	{/if}

	{#if skipped.length > 0}
		<div class="mt-3 flex items-start gap-2 rounded-lg border border-warn/30 bg-warn/5 px-3 py-2 text-xs text-warn">
			<TriangleAlert size={14} class="mt-0.5 shrink-0" />
			<span>
				{skipped.length === 1 ? `${skipped[0].plantLabel} was` : `${skipped.length} plants were`}
				left out of the most recent care action.
			</span>
		</div>
	{/if}
</section>
