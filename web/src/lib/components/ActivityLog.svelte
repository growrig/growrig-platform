<script lang="ts">
	import { onMount } from 'svelte';
	import { getActivity, getEnvironments, getGrows } from '$lib/api';
	import type { Activity, Environment, Grow } from '$lib/types';
	import { Select, Pagination } from '$lib/components/ui';
	import SlidersHorizontal from '@lucide/svelte/icons/sliders-horizontal';
	import TriangleAlert from '@lucide/svelte/icons/triangle-alert';
	import CircleAlert from '@lucide/svelte/icons/circle-alert';
	import Info from '@lucide/svelte/icons/info';
	import Sprout from '@lucide/svelte/icons/sprout';
	import Box from '@lucide/svelte/icons/box';
	import Droplet from '@lucide/svelte/icons/droplet';
	import { fmtDateTime } from '$lib/datetime';

	// `importantOnly` is the default view: only warnings and errors, hiding
	// routine control actions and notices. Users can flip to "All" per instance.
	// `showFilters` adds environment + grow dropdowns (used on the global log).
	interface Props {
		environmentId?: string;
		growId?: string;
		limit?: number;
		showEnvironment?: boolean;
		showFilters?: boolean;
		importantOnly?: boolean;
		title?: string;
	}
	let {
		environmentId,
		growId,
		limit = 20,
		showEnvironment = false,
		showFilters = false,
		importantOnly = true,
		title
	}: Props = $props();

	let events = $state<Activity[]>([]);
	let total = $state(0);
	let page = $state(1);
	let environments = $state<Environment[]>([]);
	let grows = $state<Grow[]>([]);
	let loading = $state(true);
	// View selects which slice of the journal to show: only warnings/errors
	// (important), only care actions, or everything.
	type LogView = 'important' | 'care' | 'all';
	// svelte-ignore state_referenced_locally -- importantOnly is only the initial default; the toggle owns it thereafter
	let view = $state<LogView>(importantOnly ? 'important' : 'all');
	let envFilter = $state('');
	let growFilter = $state('');
	const IMPORTANT_LEVELS = ['warning', 'error'];

	// A source column (which grow/env each event belongs to) is useful whenever
	// the log spans more than one context.
	const showSource = $derived(showEnvironment || showFilters);

	async function reload() {
		try {
			const levels = view === 'important' ? IMPORTANT_LEVELS : undefined;
			const types = view === 'care' ? ['care'] : undefined;
			const env = showFilters ? envFilter : environmentId;
			const grow = showFilters ? growFilter : growId;
			const res = await getActivity({ environmentId: env, growId: grow, levels, types, limit, offset: (page - 1) * limit });
			events = res.items ?? [];
			total = res.total ?? 0;
		} finally {
			loading = false;
		}
	}
	function refetch() {
		loading = true;
		reload();
	}
	// Filter/toggle changes reset to the first page; page changes keep filters.
	function setView(next: LogView) {
		if (next === view) return;
		view = next;
		page = 1;
		refetch();
	}
	function setEnvFilter(v: string) {
		envFilter = v;
		page = 1;
		refetch();
	}
	function setGrowFilter(v: string) {
		growFilter = v;
		page = 1;
		refetch();
	}
	function goPage(p: number) {
		page = p;
		refetch();
	}

	onMount(() => {
		reload();
		if (showSource) {
			getEnvironments().then((e) => (environments = e)).catch(() => {});
			getGrows().then((g) => (grows = g)).catch(() => {});
		}
		const timer = setInterval(reload, 5000);
		return () => clearInterval(timer);
	});

	const envName = (id?: string) => environments.find((e) => e.id === id)?.name ?? id ?? '';
	const growName = (id?: string) => grows.find((g) => g.id === id)?.name ?? id ?? '';
	const time = (value: string) => fmtDateTime(value);

	// Where an event belongs: a grow takes precedence over its environment, since
	// grow events (created, advanced, plant moved) are the more specific source.
	function source(event: Activity): { kind: 'grow' | 'env' | 'system'; name: string } {
		if (event.growId) return { kind: 'grow', name: growName(event.growId) };
		if (event.environmentId) return { kind: 'env', name: envName(event.environmentId) };
		return { kind: 'system', name: 'System' };
	}

	const envItems = $derived([
		{ value: '', label: 'All environments' },
		...environments.map((e) => ({ value: e.id, label: e.name }))
	]);
	const growItems = $derived([
		{ value: '', label: 'All grows' },
		...grows.map((g) => ({ value: g.id, label: g.name }))
	]);
</script>

<div class="mb-3 flex flex-wrap items-center justify-between gap-3">
	<div class="flex flex-wrap items-center gap-2">
		{#if title}<h2 class="text-sm font-semibold uppercase tracking-wide text-rig-400">{title}</h2>{/if}
		{#if showFilters}
			<div class="w-44"><Select value={envFilter} onValueChange={setEnvFilter} items={envItems} /></div>
			<div class="w-40"><Select value={growFilter} onValueChange={setGrowFilter} items={growItems} /></div>
		{/if}
	</div>
	<div class="inline-flex overflow-hidden rounded-lg border border-rig-800 text-xs">
		<button
			onclick={() => setView('important')}
			class="px-2.5 py-1 {view === 'important' ? 'bg-rig-800 text-rig-100' : 'text-rig-500 hover:text-rig-300'}"
		>Important</button>
		<button
			onclick={() => setView('care')}
			class="border-l border-rig-800 px-2.5 py-1 {view === 'care' ? 'bg-rig-800 text-rig-100' : 'text-rig-500 hover:text-rig-300'}"
		>Care</button>
		<button
			onclick={() => setView('all')}
			class="border-l border-rig-800 px-2.5 py-1 {view === 'all' ? 'bg-rig-800 text-rig-100' : 'text-rig-500 hover:text-rig-300'}"
		>All</button>
	</div>
</div>

{#if loading}
	<p class="text-sm text-rig-500">Loading activity…</p>
{:else if events.length === 0}
	<div class="rounded-xl border border-dashed border-rig-800 p-6 text-center text-sm text-rig-500">{view === 'care' ? 'No care logged yet.' : view === 'all' ? 'No activity recorded yet.' : 'No warnings or errors.'}</div>
{:else}
	<div class="overflow-hidden rounded-xl border border-rig-800 bg-rig-900/30">
		{#each events as event, i (event.id)}
			{@const src = source(event)}
			<div class="flex items-start gap-3 px-4 py-3 {i ? 'border-t border-rig-800' : ''}">
				{#if showSource}
					<div class="mt-0.5 flex w-32 shrink-0 items-center gap-1.5 text-xs sm:w-40" title={src.name}>
						{#if src.kind === 'grow'}<Sprout size={13} class="shrink-0 text-leaf" />{:else if src.kind === 'env'}<Box size={13} class="shrink-0 text-rig-400" />{/if}
						<span class="truncate {src.kind === 'system' ? 'text-rig-500' : 'text-rig-300'}">{src.name}</span>
					</div>
				{/if}
				<div class="mt-0.5 {event.level === 'error' ? 'text-danger' : event.level === 'warning' ? 'text-warn' : event.type === 'care' ? 'text-sky-400' : event.type === 'control' ? 'text-leaf' : 'text-rig-400'}">
					{#if event.level === 'error'}<CircleAlert size={17} />{:else if event.level === 'warning'}<TriangleAlert size={17} />{:else if event.type === 'care'}<Droplet size={17} />{:else if event.type === 'control'}<SlidersHorizontal size={17} />{:else}<Info size={17} />{/if}
				</div>
				<div class="min-w-0 flex-1">
					<div class="text-sm text-rig-200">{event.message}</div>
					<div class="mt-0.5 flex flex-wrap gap-x-2 text-xs text-rig-500"><span>{time(event.time)}</span><span class="capitalize">{event.type}</span></div>
				</div>
			</div>
		{/each}
	</div>

	{#if total > limit}
		<div class="mt-3 flex justify-center">
			<Pagination count={total} perPage={limit} {page} onPageChange={goPage} />
		</div>
	{/if}
{/if}
