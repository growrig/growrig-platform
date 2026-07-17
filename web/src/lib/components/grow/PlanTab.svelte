<script lang="ts">
	import { live } from '$lib/live.svelte';
	import type { CareActionDef, GrowAnalytics, GrowDetail } from '$lib/types';
	import { titleCase } from '$lib/format';
	import { fmtDate } from '$lib/datetime';
	import { Select } from '$lib/components/ui';
	import Settings2 from '@lucide/svelte/icons/settings-2';
	import Zap from '@lucide/svelte/icons/zap';
	import BookOpen from '@lucide/svelte/icons/book-open';

	interface Props {
		grow: GrowDetail;
		isAdmin: boolean;
		analytics: GrowAnalytics | null;
		careDefs: CareActionDef[];
		onAdvance: (stage: string) => void;
		onCareSettings: () => void;
	}
	let { grow, isAdmin, analytics, careDefs, onAdvance, onCareSettings }: Props = $props();

	const stageItems = $derived(grow.stages.map((s) => ({ value: s, label: titleCase(s) })));
	const currentIndex = $derived(grow.stages.indexOf(grow.stage));

	// Actual entered dates per stage, keyed by stage name (from history).
	const enteredByStage = $derived(new Map((analytics?.stageDurations ?? []).map((sd) => [sd.stage, sd.from])));

	// Environments whose automation follows this grow.
	const drivenEnvs = $derived(
		(live.snapshot?.environments ?? []).filter((e) => e.controlGrowId === grow.id)
	);
	const enabledActions = $derived(careDefs.filter((d) => d.enabled));
</script>

<div class="space-y-6">
	<!-- Stage plan -->
	<section>
		<div class="mb-3 flex items-center justify-between gap-3">
			<h2 class="text-sm font-semibold uppercase tracking-wide text-rig-400">Stages</h2>
			{#if isAdmin && grow.status === 'active'}
				<label class="flex items-center gap-2 text-sm">
					<span class="text-rig-400">Advance to</span>
					<Select value={grow.stage} onValueChange={onAdvance} items={stageItems} />
				</label>
			{/if}
		</div>
		<div class="overflow-hidden rounded-xl border border-rig-800">
			{#each grow.stages as st, i (st)}
				<div class="flex items-center justify-between gap-3 border-b border-rig-800/60 px-4 py-2.5 last:border-0 {st === grow.stage ? 'bg-leaf/5' : ''}">
					<div class="flex items-center gap-2">
						<span class="grid h-6 w-6 place-items-center rounded-full text-xs {st === grow.stage ? 'bg-leaf/20 text-leaf' : i < currentIndex ? 'bg-rig-800 text-rig-400' : 'bg-rig-800/50 text-rig-500'}">{i + 1}</span>
						<span class="text-sm capitalize {st === grow.stage ? 'font-medium text-rig-100' : 'text-rig-300'}">{st}</span>
						{#if st === grow.stage}<span class="rounded-full bg-leaf/20 px-2 py-0.5 text-[10px] uppercase tracking-wide text-leaf">current</span>{/if}
					</div>
					<span class="text-xs text-rig-500">{enteredByStage.has(st) ? fmtDate(enteredByStage.get(st)!) : i > currentIndex ? 'upcoming' : '—'}</span>
				</div>
			{/each}
		</div>
	</section>

	<!-- Care schedule / actions -->
	<section>
		<div class="mb-3 flex items-center justify-between gap-3">
			<h2 class="text-sm font-semibold uppercase tracking-wide text-rig-400">Care actions</h2>
			{#if isAdmin}
				<button onclick={onCareSettings} class="inline-flex items-center gap-1.5 rounded-md border border-rig-700 px-3 py-1.5 text-sm text-rig-300 transition-colors hover:border-rig-500"><Settings2 size={14} /> Configure</button>
			{/if}
		</div>
		{#if enabledActions.length}
			<div class="flex flex-wrap gap-2">
				{#each enabledActions as a (a.key)}
					<span class="rounded-full border border-rig-700 px-3 py-1 text-xs text-rig-300">{a.label}</span>
				{/each}
			</div>
		{:else}
			<p class="text-sm text-rig-500">No care actions enabled.</p>
		{/if}
	</section>

	<!-- Feeding recipes (managed in the Library) -->
	<section>
		<h2 class="mb-3 text-sm font-semibold uppercase tracking-wide text-rig-400">Feeding</h2>
		<a href="/library" class="inline-flex items-center gap-2 rounded-lg border border-rig-800 bg-rig-950/40 px-4 py-3 text-sm text-rig-300 transition-colors hover:border-rig-600">
			<BookOpen size={16} class="text-rig-400" /> Manage feeding recipes in the Library
		</a>
	</section>

	<!-- Lighting / automation relationship -->
	<section>
		<h2 class="mb-3 text-sm font-semibold uppercase tracking-wide text-rig-400">Automation</h2>
		{#if drivenEnvs.length}
			<div class="space-y-2">
				{#each drivenEnvs as e (e.id)}
					<a href="/env/{e.id}" class="flex items-center gap-2 rounded-lg border border-leaf/30 bg-leaf/5 px-4 py-3 text-sm transition-colors hover:border-leaf/60">
						<Zap size={15} class="text-leaf" />
						<span class="text-rig-200">{e.name}</span>
						<span class="text-xs text-rig-500">follows this grow's stage for its photoperiod</span>
					</a>
				{/each}
			</div>
		{:else}
			<p class="text-sm text-rig-500">No environment currently follows this grow for automation. Set a control grow on an environment to link its photoperiod to these stages.</p>
		{/if}
	</section>
</div>
