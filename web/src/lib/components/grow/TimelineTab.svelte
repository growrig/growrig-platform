<script lang="ts">
	import type { Activity, CareHistory, GrowAnalytics, GrowDetail, GrowPhoto } from '$lib/types';
	import { growPhotoImageURL } from '$lib/api';
	import { careVisual, fmtVolume } from '$lib/care';
	import { titleCase } from '$lib/format';
	import { fmtDate, fmtTime } from '$lib/datetime';
	import type { Component } from 'svelte';
	import Camera from '@lucide/svelte/icons/camera';
	import Milestone from '@lucide/svelte/icons/milestone';
	import Cpu from '@lucide/svelte/icons/cpu';
	import TriangleAlert from '@lucide/svelte/icons/triangle-alert';

	type Kind = 'photo' | 'care' | 'milestone' | 'system';
	// eslint-disable-next-line @typescript-eslint/no-explicit-any
	type AnyIcon = Component<any>;
	interface Item {
		t: number;
		kind: Kind;
		icon: AnyIcon;
		label: string;
		detail?: string;
		photoId?: string;
	}

	interface Props {
		grow: GrowDetail;
		care: CareHistory | null;
		photos: GrowPhoto[];
		activity: Activity[];
		analytics: GrowAnalytics | null;
	}
	let { grow, care, photos, activity, analytics }: Props = $props();

	type Filter = 'all' | 'photo' | 'care' | 'milestone' | 'system';
	let filter = $state<Filter>('all');
	const filters: { id: Filter; label: string }[] = [
		{ id: 'all', label: 'All' },
		{ id: 'photo', label: 'Photos' },
		{ id: 'care', label: 'Care' },
		{ id: 'milestone', label: 'Milestones' },
		{ id: 'system', label: 'System' }
	];

	const items = $derived.by<Item[]>(() => {
		const out: Item[] = [];
		for (const e of care?.events ?? []) {
			const v = careVisual(e.type);
			const bits: string[] = [];
			const ml = (e.applications ?? []).reduce((n, a) => n + (a.amountMl ?? 0), 0);
			if (ml) bits.push(fmtVolume(ml));
			if (e.ph) bits.push(`pH ${e.ph}`);
			if (e.recipeName) bits.push(e.recipeName);
			out.push({ t: new Date(e.occurredAt).getTime(), kind: 'care', icon: v.icon, label: v.label, detail: bits.join(' · ') || undefined });
		}
		for (const p of photos) {
			out.push({ t: new Date(p.takenAt).getTime(), kind: 'photo', icon: Camera, label: 'Photo added', detail: p.caption, photoId: p.id });
		}
		for (const sd of analytics?.stageDurations ?? []) {
			out.push({ t: new Date(sd.from).getTime(), kind: 'milestone', icon: Milestone, label: `Entered ${titleCase(sd.stage)} stage` });
		}
		for (const a of activity) {
			// Photos and stage changes already have first-class entries.
			if (a.type === 'notice' || (a.type === 'configuration' && /advanced to/i.test(a.message))) continue;
			const icon = a.level === 'warning' || a.level === 'error' ? TriangleAlert : Cpu;
			out.push({ t: new Date(a.time).getTime(), kind: 'system', icon, label: a.message });
		}
		return out.sort((x, y) => y.t - x.t);
	});

	const filtered = $derived(filter === 'all' ? items : items.filter((i) => i.kind === filter));

	// Group by calendar day for day headers.
	const groups = $derived.by(() => {
		const map = new Map<string, Item[]>();
		for (const it of filtered) {
			const key = new Date(it.t).toDateString();
			(map.get(key) ?? map.set(key, []).get(key)!).push(it);
		}
		return [...map.entries()];
	});

	const kindTone: Record<Kind, string> = {
		photo: 'text-sky-400',
		care: 'text-leaf',
		milestone: 'text-warn',
		system: 'text-rig-400'
	};
</script>

<div class="mb-4 flex flex-wrap gap-2">
	{#each filters as f (f.id)}
		<button
			onclick={() => (filter = f.id)}
			class="rounded-full border px-3 py-1 text-xs font-medium transition-colors {filter === f.id ? 'border-rig-600 bg-rig-800 text-rig-100' : 'border-rig-800 text-rig-400 hover:border-rig-700'}"
		>
			{f.label}
		</button>
	{/each}
</div>

{#if filtered.length === 0}
	<div class="rounded-xl border border-dashed border-rig-800 p-10 text-center text-sm text-rig-500">
		Nothing here yet — care actions, photos and milestones will appear as the grow progresses.
	</div>
{:else}
	<div class="space-y-6">
		{#each groups as [day, dayItems] (day)}
			<div>
				<div class="mb-2 text-xs font-semibold uppercase tracking-wide text-rig-500">{fmtDate(new Date(day))}</div>
				<div class="space-y-2">
					{#each dayItems as it (it.t + it.label)}
						<div class="flex items-center gap-3 rounded-lg border border-rig-800 bg-rig-900/40 px-3 py-2.5">
							<span class="grid h-8 w-8 shrink-0 place-items-center rounded-md bg-rig-800 {kindTone[it.kind]}"><it.icon size={16} /></span>
							{#if it.photoId}
								<img src={growPhotoImageURL(grow.id, it.photoId)} alt="" class="h-10 w-10 shrink-0 rounded object-cover" />
							{/if}
							<div class="min-w-0 flex-1">
								<p class="truncate text-sm text-rig-100">{it.label}</p>
								{#if it.detail}<p class="truncate text-xs text-rig-500">{it.detail}</p>{/if}
							</div>
							<span class="shrink-0 text-xs tabular-nums text-rig-600">{fmtTime(new Date(it.t), { hour: '2-digit', minute: '2-digit' })}</span>
						</div>
					{/each}
				</div>
			</div>
		{/each}
	</div>
{/if}
