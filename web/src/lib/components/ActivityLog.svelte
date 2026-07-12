<script lang="ts">
	import { onMount } from 'svelte';
	import { getActivity, getEnvironments } from '$lib/api';
	import type { Activity, Environment } from '$lib/types';
	import SlidersHorizontal from '@lucide/svelte/icons/sliders-horizontal';
	import TriangleAlert from '@lucide/svelte/icons/triangle-alert';
	import CircleAlert from '@lucide/svelte/icons/circle-alert';
	import Info from '@lucide/svelte/icons/info';
	import { fmtDateTime } from '$lib/datetime';

	// `importantOnly` is the default view: only warnings and errors, hiding
	// routine control actions and notices. Users can flip to "All" per instance.
	interface Props { environmentId?: string; growId?: string; limit?: number; showEnvironment?: boolean; importantOnly?: boolean; title?: string }
	let { environmentId, growId, limit = 20, showEnvironment = false, importantOnly = true, title }: Props = $props();
	let events = $state<Activity[]>([]);
	let environments = $state<Environment[]>([]);
	let loading = $state(true);
	// svelte-ignore state_referenced_locally -- importantOnly is only the initial default; the toggle owns it thereafter
	let showAll = $state(!importantOnly);
	const IMPORTANT_LEVELS = ['warning', 'error'];

	async function reload() {
		try {
			const levels = showAll ? undefined : IMPORTANT_LEVELS;
			[events, environments] = await Promise.all([getActivity({ environmentId, growId, levels, limit }), showEnvironment ? getEnvironments() : Promise.resolve([])]);
		} finally { loading = false; }
	}
	function setShowAll(next: boolean) {
		if (next === showAll) return;
		showAll = next;
		loading = true;
		reload();
	}
	onMount(() => { reload(); const timer = setInterval(reload, 5000); return () => clearInterval(timer); });
	const envName = (id?: string) => environments.find((env) => env.id === id)?.name ?? id ?? 'System';
	const time = (value: string) => fmtDateTime(value);
</script>

<div class="mb-3 flex items-center justify-between gap-3">
	{#if title}<h2 class="text-sm font-semibold uppercase tracking-wide text-rig-400">{title}</h2>{:else}<span></span>{/if}
	<div class="inline-flex overflow-hidden rounded-lg border border-rig-800 text-xs">
		<button
			onclick={() => setShowAll(false)}
			class="px-2.5 py-1 {!showAll ? 'bg-rig-800 text-rig-100' : 'text-rig-500 hover:text-rig-300'}"
		>Important</button>
		<button
			onclick={() => setShowAll(true)}
			class="border-l border-rig-800 px-2.5 py-1 {showAll ? 'bg-rig-800 text-rig-100' : 'text-rig-500 hover:text-rig-300'}"
		>All</button>
	</div>
</div>

{#if loading}
	<p class="text-sm text-rig-500">Loading activity…</p>
{:else if events.length === 0}
	<div class="rounded-xl border border-dashed border-rig-800 p-6 text-center text-sm text-rig-500">{showAll ? 'No activity recorded yet.' : 'No warnings or errors.'}</div>
{:else}
	<div class="overflow-hidden rounded-xl border border-rig-800 bg-rig-900/30">
		{#each events as event, i (event.id)}
			<div class="flex gap-3 px-4 py-3 {i ? 'border-t border-rig-800' : ''}">
				<div class="mt-0.5 {event.level === 'error' ? 'text-danger' : event.level === 'warning' ? 'text-warn' : event.type === 'control' ? 'text-leaf' : 'text-rig-400'}">
					{#if event.level === 'error'}<CircleAlert size={17} />{:else if event.level === 'warning'}<TriangleAlert size={17} />{:else if event.type === 'control'}<SlidersHorizontal size={17} />{:else}<Info size={17} />{/if}
				</div>
				<div class="min-w-0 flex-1">
					<div class="text-sm text-rig-200">{event.message}</div>
					<div class="mt-0.5 flex flex-wrap gap-x-2 text-xs text-rig-500">{#if showEnvironment}<span>{envName(event.environmentId)}</span>{/if}<span>{time(event.time)}</span><span class="capitalize">{event.type}</span></div>
				</div>
			</div>
		{/each}
	</div>
{/if}
