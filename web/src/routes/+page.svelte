<script lang="ts">
	import { live } from '$lib/live.svelte';
	import { climateTone, toneClass, valueNow, vpdZone } from '$lib/format';
	import { getInfo, getLocations, loadDemo, weather } from '$lib/api';
	import { onMount } from 'svelte';
	import type { EnvironmentView, Location, Weather } from '$lib/types';
	import { resolveLocationId } from '$lib/location';
	import Sprout from '@lucide/svelte/icons/sprout';
	import MapPin from '@lucide/svelte/icons/map-pin';
	import Thermometer from '@lucide/svelte/icons/thermometer';
	import Droplets from '@lucide/svelte/icons/droplets';
	import Gauge from '@lucide/svelte/icons/gauge';
	import ChevronRight from '@lucide/svelte/icons/chevron-right';

	const snap = $derived(live.snapshot);

	const healthDot = (h: string) =>
		h === 'online' ? 'bg-leaf' : h === 'stale' ? 'bg-warn' : 'bg-danger';

	let locations = $state<Location[]>([]);
	let weatherByLoc = $state<Record<string, Weather>>({});

	// Effective location per environment, inheriting a tent's air-source room's
	// location when it has none of its own.
	const locOf = (e: EnvironmentView) => resolveLocationId(e, snap?.environments ?? []);

	// Build the Location → Room → Grow box hierarchy. A tent's air source is its
	// room; a room's location is the site. Tents whose air source isn't a room in
	// their location fall back to a "loose" list rendered at the location level.
	type RoomNode = { room: EnvironmentView; boxes: EnvironmentView[] };
	type LocNode = {
		key: string;
		name: string;
		located: boolean;
		loc?: Location;
		rooms: RoomNode[];
		looseBoxes: EnvironmentView[];
	};

	const groups = $derived.by<LocNode[]>(() => {
		const envs = snap?.environments ?? [];
		const byId = new Map(envs.map((e) => [e.id, e]));
		const rooms = envs.filter((e) => e.kind === 'room');
		const tents = envs.filter((e) => e.kind === 'tent');
		const roomOf = (t: EnvironmentView) => {
			const src = t.airSourceId ? byId.get(t.airSourceId) : undefined;
			return src && src.kind === 'room' ? src : undefined;
		};

		const out: LocNode[] = [];
		const placed = new Set<string>();

		for (const loc of [...locations].sort((a, b) => a.name.localeCompare(b.name))) {
			const roomNodes: RoomNode[] = rooms
				.filter((r) => locOf(r) === loc.id)
				.map((room) => ({ room, boxes: tents.filter((t) => roomOf(t)?.id === room.id) }));
			const looseBoxes = tents.filter(
				(t) => locOf(t) === loc.id && !roomNodes.some((n) => n.boxes.includes(t))
			);
			if (!roomNodes.length && !looseBoxes.length) continue;
			for (const n of roomNodes) {
				placed.add(n.room.id);
				n.boxes.forEach((b) => placed.add(b.id));
			}
			looseBoxes.forEach((b) => placed.add(b.id));
			out.push({ key: loc.id, name: loc.name, located: true, loc, rooms: roomNodes, looseBoxes });
		}

		const orphans = envs.filter((e) => !placed.has(e.id));
		if (orphans.length) {
			const roomNodes: RoomNode[] = orphans
				.filter((e) => e.kind === 'room')
				.map((room) => ({
					room,
					boxes: orphans.filter((t) => t.kind === 'tent' && roomOf(t)?.id === room.id)
				}));
			const looseBoxes = orphans.filter(
				(e) => e.kind === 'tent' && !roomNodes.some((n) => n.boxes.includes(e))
			);
			out.push({ key: '__none__', name: 'No location', located: false, rooms: roomNodes, looseBoxes });
		}
		return out;
	});

	const hasEnvs = $derived((snap?.environments ?? []).length > 0);

	let isSimulator = $state(false);
	let loadingDemo = $state(false);
	onMount(async () => {
		try {
			isSimulator = (await getInfo()).adapter === 'simulator';
		} catch {
			/* ignore */
		}
		try {
			locations = await getLocations();
		} catch {
			/* ignore */
		}
		// One weather fetch per sited location for the header strip.
		for (const loc of locations) {
			weather(loc.lat, loc.lon)
				.then((w) => (weatherByLoc = { ...weatherByLoc, [loc.id]: w }))
				.catch(() => {});
		}
	});
	async function seedDemo() {
		loadingDemo = true;
		try {
			await loadDemo();
		} catch {
			/* ignore; live feed will refresh */
		} finally {
			loadingDemo = false;
		}
	}
</script>

<!-- Compact climate readout for a room header (temp · RH, plus VPD). -->
{#snippet climate(env: EnvironmentView)}
	{#if env.hasTemp || env.hasHum || env.hasCO2}
		<div class="flex items-center gap-3 tabular-nums">
			<span class="text-sm text-rig-300">
				{#if env.hasTemp}<span class={climateTone(env.tempC, env.targetTempC, env.emergencyTempC)}
						>{env.tempC.toFixed(1)}°</span
					>{/if}{#if env.hasHum}<span class="text-rig-500"
						>{#if env.hasTemp} · {/if}{env.humidity.toFixed(0)}%</span
					>{/if}
			</span>
			{#if env.hasClimate}
				<span class="text-sm font-medium {toneClass[vpdZone(env.vpd).tone]}">
					{env.vpd.toFixed(2)}<span class="ml-0.5 text-[10px] text-rig-500">VPD</span>
				</span>
			{/if}
		</div>
	{/if}
{/snippet}

<!-- Grow box (tent) card — the leaf of the hierarchy. -->
{#snippet box(env: EnvironmentView)}
	<a
		href="/env/{env.id}"
		class="block rounded-xl border border-rig-800 bg-rig-900/50 p-4 transition-colors hover:border-rig-600"
	>
		<div class="mb-3 flex items-center justify-between gap-2">
			<div class="flex items-center gap-2">
				<span class="h-2 w-2 rounded-full {healthDot(env.health)}"></span>
				<h3 class="font-semibold">{env.name}</h3>
			</div>
			<span
				class="rounded-full bg-rig-800 px-2 py-0.5 text-[10px] uppercase tracking-wide text-rig-400"
			>
				{env.kind}
			</span>
		</div>
		{#if env.hasTemp || env.hasHum || env.hasCO2}
			<div class="flex items-end justify-between">
				<div>
					<div
						class="text-3xl font-semibold tabular-nums {env.hasTemp
							? climateTone(env.tempC, env.targetTempC, env.emergencyTempC)
							: 'text-rig-500'}"
					>
						{env.hasTemp ? `${env.tempC.toFixed(1)}°C` : '—'}
					</div>
					<div class="text-sm text-rig-400">
						{#if env.hasHum}{env.humidity.toFixed(0)}% RH{/if}{#if env.hasCO2}{#if env.hasHum} ·
							{/if}{env.co2.toFixed(0)} ppm{/if}
					</div>
				</div>
				{#if env.hasClimate}
					<div class="text-right">
						<div class="text-lg font-semibold tabular-nums {toneClass[vpdZone(env.vpd).tone]}">
							{env.vpd.toFixed(2)}
						</div>
						<div class="text-xs text-rig-500">VPD kPa</div>
					</div>
				{/if}
			</div>
		{:else}
			<p class="text-sm text-rig-500">no climate sensors yet</p>
		{/if}
	</a>
{/snippet}

<!-- A room and the grow boxes it feeds air to. -->
{#snippet roomBlock(node: RoomNode)}
	<div class="rounded-2xl border border-rig-800 bg-rig-900/20 p-4 sm:p-5">
		<a href="/env/{node.room.id}" class="group flex items-center justify-between gap-3">
			<div class="flex items-center gap-2">
				<span class="h-2 w-2 rounded-full {healthDot(node.room.health)}"></span>
				<h2 class="font-semibold">{node.room.name}</h2>
				<span
					class="rounded-full bg-rig-800 px-2 py-0.5 text-[10px] uppercase tracking-wide text-rig-400"
				>
					{node.room.kind}
				</span>
			</div>
			<div class="flex items-center gap-2 text-rig-500">
				{@render climate(node.room)}
				<ChevronRight size={16} class="transition-transform group-hover:translate-x-0.5" />
			</div>
		</a>

		<div class="mt-4 border-t border-rig-800/70 pt-4">
			{#if node.boxes.length}
				<div class="grid gap-3 sm:grid-cols-2">
					{#each node.boxes as env (env.id)}{@render box(env)}{/each}
				</div>
			{:else}
				<p class="text-sm text-rig-500">No grow boxes in this room yet.</p>
			{/if}
		</div>
	</div>
{/snippet}

<!-- Outdoor weather strip for a located site. -->
{#snippet weatherStrip(w: Weather)}
	{@const t = valueNow(w.temp)}
	{@const h = valueNow(w.humidity)}
	{@const p = valueNow(w.pressure)}
	<div class="flex items-center gap-3 text-sm text-rig-400">
		<span class="text-[11px] uppercase tracking-wide text-rig-600">Outside</span>
		{#if t !== undefined}
			<span class="flex items-center gap-1 tabular-nums"><Thermometer size={14} />{t.toFixed(1)}°C</span>
		{/if}
		{#if h !== undefined}
			<span class="flex items-center gap-1 tabular-nums"><Droplets size={14} />{h.toFixed(0)}%</span>
		{/if}
		{#if p !== undefined}
			<span class="hidden items-center gap-1 tabular-nums sm:flex"
				><Gauge size={14} />{p.toFixed(0)} hPa</span
			>
		{/if}
	</div>
{/snippet}

{#if !snap}
	<div class="grid place-items-center py-24 text-rig-400"><p>Connecting to Grow Core…</p></div>
{:else if !hasEnvs}
	<div class="rounded-xl border border-dashed border-rig-800 p-10 text-center">
		<div class="mb-3 flex justify-center text-rig-500"><Sprout size={40} /></div>
		<h2 class="mb-1 text-lg font-semibold">Welcome to GrowRig</h2>
		<p class="mb-5 text-sm text-rig-400">Set up your first grow box to get started.</p>
		<div class="flex flex-wrap justify-center gap-3">
			<a
				href="/wizard/box"
				class="rounded-md bg-rig-500 px-5 py-2 text-sm font-medium text-rig-950 transition-colors hover:bg-rig-400"
			>
				Set up a Grow Box
			</a>
			{#if isSimulator}
				<button
					onclick={seedDemo}
					disabled={loadingDemo}
					class="rounded-md border border-rig-700 px-5 py-2 text-sm text-rig-200 transition-colors hover:border-rig-500 disabled:opacity-50"
				>
					{loadingDemo ? 'Loading…' : 'Load demo tent'}
				</button>
			{/if}
		</div>
	</div>
{:else}
	<div class="space-y-10">
		{#each groups as group (group.key)}
			<section>
				<div class="mb-4 flex flex-wrap items-center justify-between gap-x-4 gap-y-2">
					<h1
						class="flex items-center gap-1.5 text-sm font-semibold uppercase tracking-wide {group.located
							? 'text-leaf'
							: 'text-rig-500'}"
					>
						<MapPin size={14} />
						{group.name}
					</h1>
					{#if group.loc && weatherByLoc[group.loc.id]}
						{@render weatherStrip(weatherByLoc[group.loc.id])}
					{/if}
				</div>

				<div class="space-y-4">
					{#each group.rooms as node (node.room.id)}{@render roomBlock(node)}{/each}
					{#if group.looseBoxes.length}
						<div class="grid gap-3 sm:grid-cols-2">
							{#each group.looseBoxes as env (env.id)}{@render box(env)}{/each}
						</div>
					{/if}
				</div>
			</section>
		{/each}
	</div>
{/if}
