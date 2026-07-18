<script lang="ts">
	import type { GrowDetail } from '$lib/types';
	import { titleCase } from '$lib/format';
	import { Button } from '$lib/components/ui';
	import Pencil from '@lucide/svelte/icons/pencil';
	import Settings2 from '@lucide/svelte/icons/settings-2';
	import CircleCheck from '@lucide/svelte/icons/circle-check';

	interface Props {
		grow: GrowDetail;
		isAdmin: boolean;
		onEdit: () => void;
		onCareSettings: () => void;
		onComplete: () => void;
		onDelete: () => void;
	}
	let { grow, isAdmin, onEdit, onCareSettings, onComplete, onDelete }: Props = $props();

	// Compact one-line summary of the growing setup, or '' when nothing is set.
	const setupSummary = $derived.by(() => {
		const s = grow.setup ?? {};
		const container = s.potSize && s.potSize > 0 ? `${s.potSize} ${s.potUnit || 'L'}${s.potType ? ` ${s.potType}` : ''} pot` : '';
		return [s.medium && titleCase(s.medium), s.nutrientMethod && titleCase(s.nutrientMethod), container, s.mediumDetails]
			.filter(Boolean)
			.join(' · ');
	});
</script>

{#if !isAdmin}
	<p class="text-rig-400">You don't have permission to change this grow's settings.</p>
{:else}
	<div class="space-y-8">
		<!-- Grow details -->
		<section class="space-y-3">
			<h2 class="text-sm font-semibold uppercase tracking-wide text-rig-400">Grow details</h2>
			<div class="flex items-center justify-between gap-4 rounded-xl border border-rig-800 bg-rig-900/40 p-4">
				<div>
					<div class="text-sm font-medium">{grow.name}</div>
					<div class="text-xs text-rig-500">
						{titleCase(grow.species) || 'No species'} · Day {grow.totalDays} · {grow.plantCount} plant{grow.plantCount === 1 ? '' : 's'}
					</div>
					<div class="mt-1 text-xs text-rig-500">
						{setupSummary || 'Growing setup not configured yet'}
					</div>
				</div>
				<Button variant="secondary" onclick={onEdit}><Pencil size={15} /> Edit grow</Button>
			</div>
		</section>

		<!-- Care actions -->
		<section class="space-y-3">
			<h2 class="text-sm font-semibold uppercase tracking-wide text-rig-400">Care actions</h2>
			<div class="flex items-center justify-between gap-4 rounded-xl border border-rig-800 bg-rig-900/40 p-4">
				<div>
					<div class="text-sm font-medium">Care schedule &amp; actions</div>
					<div class="text-xs text-rig-500">Configure which care actions apply and how often they're due.</div>
				</div>
				<Button variant="secondary" onclick={onCareSettings}><Settings2 size={15} /> Configure</Button>
			</div>
		</section>

		<!-- Danger zone -->
		<section class="space-y-3">
			<h2 class="text-sm font-semibold uppercase tracking-wide text-danger/80">Danger zone</h2>
			{#if grow.status === 'active'}
				<div class="flex items-center justify-between gap-4 rounded-xl border border-rig-800 bg-rig-900/40 p-4">
					<div>
						<div class="text-sm font-medium">Complete this grow</div>
						<div class="text-xs text-rig-500">Marks the grow as finished. It stays in your history.</div>
					</div>
					<Button variant="secondary" onclick={onComplete}><CircleCheck size={15} /> Complete grow</Button>
				</div>
			{/if}
			<div class="flex items-center justify-between gap-4 rounded-xl border border-danger/30 bg-danger/5 p-4">
				<div>
					<div class="text-sm font-medium">Delete this grow</div>
					<div class="text-xs text-rig-500">Removes the grow and all its plants, care log and photos. This cannot be undone.</div>
				</div>
				<button
					onclick={onDelete}
					class="rounded-md bg-danger/90 px-4 py-1.5 text-sm font-medium text-white transition-colors hover:bg-danger"
				>
					Delete grow
				</button>
			</div>
		</section>
	</div>
{/if}
