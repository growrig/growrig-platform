<script lang="ts">
	import { live } from '$lib/live.svelte';
	import { climateTone, toneClass, vpdZone } from '$lib/format';
	import { getInfo, getLocations, loadDemo } from '$lib/api';
	import { onMount } from 'svelte';
	import type { EnvironmentView, Location } from '$lib/types';
	import { resolveLocationId } from '$lib/location';
	import Sprout from '@lucide/svelte/icons/sprout';
	import MapPin from '@lucide/svelte/icons/map-pin';

	const snap = $derived(live.snapshot);
	const tents = $derived((snap?.environments ?? []).filter((e) => e.kind === 'tent'));
	const rooms = $derived((snap?.environments ?? []).filter((e) => e.kind === 'room'));

	const healthDot = (h: string) =>
		h === 'online' ? 'bg-leaf' : h === 'stale' ? 'bg-warn' : 'bg-danger';

	let locations = $state<Location[]>([]);
	// Effective location per environment, inheriting a tent's air-source room's
	// location when it has none of its own.
	const locOf = (e: EnvironmentView) => resolveLocationId(e, snap?.environments ?? []);
	// Group environments by location once any environment is sited; otherwise
	// fall back to the flat tents/rooms layout.
	const useGroups = $derived(
		locations.length > 0 && (snap?.environments ?? []).some((e) => { const l = locOf(e); return l && locations.some((x) => x.id === l); })
	);
	const groups = $derived.by(() => {
		const envs = snap?.environments ?? [];
		const out: { key: string; name: string; located: boolean; envs: EnvironmentView[] }[] = [];
		for (const loc of [...locations].sort((a, b) => a.name.localeCompare(b.name))) {
			const inLoc = envs.filter((e) => locOf(e) === loc.id);
			if (inLoc.length) out.push({ key: loc.id, name: loc.name, located: true, envs: inLoc });
		}
		const orphan = envs.filter((e) => { const l = locOf(e); return !l || !locations.some((x) => x.id === l); });
		if (orphan.length) out.push({ key: '__none__', name: 'No location', located: false, envs: orphan });
		return out;
	});

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

{#snippet card(env: EnvironmentView)}
	<a
		href="/env/{env.id}"
		class="block rounded-xl border border-rig-800 bg-rig-900/40 p-5 transition-colors hover:border-rig-600"
	>
		<div class="mb-3 flex items-center justify-between">
			<div class="flex items-center gap-2">
				<span class="h-2 w-2 rounded-full {healthDot(env.health)}"></span>
				<h2 class="font-semibold">{env.name}</h2>
			</div>
			<span class="rounded-full bg-rig-800 px-2 py-0.5 text-[10px] uppercase tracking-wide text-rig-400">
				{env.kind}
			</span>
		</div>
		{#if env.hasTemp || env.hasHum || env.hasCO2}
			<div class="flex items-end justify-between">
				<div>
					<div class="text-3xl font-semibold tabular-nums {env.hasTemp ? climateTone(env.tempC, env.targetTempC, env.emergencyTempC) : 'text-rig-500'}">
						{env.hasTemp ? `${env.tempC.toFixed(1)}°C` : '—'}
					</div>
					<div class="text-sm text-rig-400">
						{#if env.hasHum}{env.humidity.toFixed(0)}% RH{/if}{#if env.hasCO2}{#if env.hasHum} · {/if}{env.co2.toFixed(0)} ppm{/if}
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

{#if !snap}
	<div class="grid place-items-center py-24 text-rig-400"><p>Connecting to Grow Core…</p></div>
{:else if tents.length === 0 && rooms.length === 0}
	<div class="rounded-xl border border-dashed border-rig-800 p-10 text-center">
		<div class="mb-3 flex justify-center text-rig-500"><Sprout size={40} /></div>
		<h2 class="mb-1 text-lg font-semibold">Welcome to GrowRig</h2>
		<p class="mb-5 text-sm text-rig-400">Set up your first grow box to get started.</p>
		<div class="flex flex-wrap justify-center gap-3">
			<a href="/wizard/box" class="rounded-md bg-rig-500 px-5 py-2 text-sm font-medium text-rig-950 transition-colors hover:bg-rig-400">
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
{:else if useGroups}
	<div class="space-y-10">
		{#each groups as group (group.key)}
			<section>
				<h1 class="mb-3 flex items-center gap-1.5 text-sm font-semibold uppercase tracking-wide {group.located ? 'text-leaf' : 'text-rig-500'}">
					<MapPin size={14} />
					{group.name}
				</h1>
				<div class="grid gap-4 sm:grid-cols-2">
					{#each group.envs as env (env.id)}{@render card(env)}{/each}
				</div>
			</section>
		{/each}
	</div>
{:else}
	<div class="space-y-8">
		{#if tents.length}
			<section>
				<h1 class="mb-3 text-sm font-semibold uppercase tracking-wide text-rig-400">Grow tents</h1>
				<div class="grid gap-4 sm:grid-cols-2">
					{#each tents as env (env.id)}{@render card(env)}{/each}
				</div>
			</section>
		{/if}
		{#if rooms.length}
			<section>
				<h1 class="mb-3 text-sm font-semibold uppercase tracking-wide text-rig-400">Rooms</h1>
				<div class="grid gap-4 sm:grid-cols-2">
					{#each rooms as env (env.id)}{@render card(env)}{/each}
				</div>
			</section>
		{/if}
	</div>
{/if}
