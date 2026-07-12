<script lang="ts">
	import { untrack } from 'svelte';
	import type { GeocodeResult, Location } from '$lib/types';
	import { geocode, createLocation, updateLocation } from '$lib/api';
	import MapPin from '@lucide/svelte/icons/map-pin';
	import Search from '@lucide/svelte/icons/search';

	interface Props {
		/** When set, the form edits this location instead of creating a new one. */
		location?: Location;
		/** Called with the created or updated location. */
		onSaved?: (loc: Location) => void;
	}
	let { location, onSaved }: Props = $props();

	// Seed the form once from the prop; the parent remounts (via {#key}) when it
	// switches to a different location, so a one-time snapshot is what we want.
	const seed = untrack(() => location);
	const editing = !!seed;
	// 0,0 is our sentinel for "no coordinates set".
	const seedHasCoords = !!seed && !(seed.lat === 0 && seed.lon === 0);

	let name = $state(seed?.name ?? '');
	let query = $state(seed?.address ?? '');
	let results = $state<GeocodeResult[]>([]);
	let lat = $state<number | null>(seedHasCoords ? seed!.lat : null);
	let lon = $state<number | null>(seedHasCoords ? seed!.lon : null);
	let address = $state(seed?.address ?? '');
	let searching = $state(false);
	let busy = $state(false);
	let err = $state('');

	let timer: ReturnType<typeof setTimeout> | undefined;
	function onQuery(e: Event) {
		query = (e.target as HTMLInputElement).value;
		clearTimeout(timer);
		const q = query.trim();
		if (q.length < 3) {
			results = [];
			return;
		}
		timer = setTimeout(async () => {
			searching = true;
			try {
				results = await geocode(q);
			} catch {
				results = [];
			} finally {
				searching = false;
			}
		}, 400);
	}

	function pick(r: GeocodeResult) {
		lat = r.lat;
		lon = r.lon;
		address = r.displayName;
		results = [];
		query = r.displayName;
		if (!name.trim()) name = r.displayName.split(',')[0];
	}

	async function save() {
		if (!name.trim()) return;
		busy = true;
		err = '';
		try {
			const input = {
				name: name.trim(),
				lat: lat ?? 0,
				lon: lon ?? 0,
				address: address.trim()
			};
			const loc = editing ? await updateLocation(seed!.id, input) : await createLocation(input);
			onSaved?.(loc);
			if (!editing) {
				name = '';
				query = '';
				address = '';
				lat = null;
				lon = null;
			}
		} catch (e) {
			err = e instanceof Error ? e.message : 'Could not save location';
		} finally {
			busy = false;
		}
	}

	const field =
		'w-full rounded-md border border-rig-700 bg-rig-950 px-3 py-2 text-sm focus:border-rig-500 focus:outline-none';
</script>

<div class="space-y-3">
	<label class="block">
		<span class="text-xs text-rig-400">Location name</span>
		<input bind:value={name} placeholder="e.g. Home, Greenhouse" class="{field} mt-1" />
	</label>

	<div>
		<span class="text-xs text-rig-400">Coordinates <span class="text-rig-600">(optional)</span></span>
		<p class="mt-0.5 text-xs text-rig-600">Enables local weather. Search below, or enter manually.</p>
		<div class="relative mt-1.5">
			<Search size={14} class="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-rig-500" />
			<input value={query} oninput={onQuery} placeholder="Search street, city, or point of interest" class="{field} pl-8" />
		</div>

		{#if searching}
			<p class="mt-1.5 text-xs text-rig-500">Searching…</p>
		{:else if results.length}
			<ul class="mt-1.5 max-h-44 divide-y divide-rig-800 overflow-y-auto rounded-md border border-rig-800">
				{#each results as r (r.displayName)}
					<li>
						<button type="button" onclick={() => pick(r)} class="flex w-full items-start gap-2 px-3 py-2 text-left text-xs hover:bg-rig-800/60">
							<MapPin size={13} class="mt-0.5 shrink-0 text-rig-500" />
							<span class="text-rig-200">{r.displayName}</span>
						</button>
					</li>
				{/each}
			</ul>
		{/if}

		<div class="mt-1.5 grid grid-cols-2 gap-2">
			<label class="block">
				<span class="text-[11px] text-rig-500">Latitude</span>
				<input type="number" step="any" min="-90" max="90" bind:value={lat} placeholder="—" class="{field} mt-0.5 tabular-nums" />
			</label>
			<label class="block">
				<span class="text-[11px] text-rig-500">Longitude</span>
				<input type="number" step="any" min="-180" max="180" bind:value={lon} placeholder="—" class="{field} mt-0.5 tabular-nums" />
			</label>
		</div>
	</div>

	{#if err}<p class="text-xs text-danger">{err}</p>{/if}

	<button
		type="button"
		onclick={save}
		disabled={busy || !name.trim()}
		class="rounded-md bg-rig-500 px-4 py-1.5 text-sm font-medium text-rig-950 hover:bg-rig-400 disabled:opacity-50"
	>
		{busy ? 'Saving…' : editing ? 'Save changes' : 'Add location'}
	</button>
</div>
