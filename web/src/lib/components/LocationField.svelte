<script lang="ts">
	import type { GeocodeResult, Location } from '$lib/types';
	import { geocode, createLocation } from '$lib/api';
	import { Select, type SelectItem } from '$lib/components/ui';
	import MapPin from '@lucide/svelte/icons/map-pin';
	import Search from '@lucide/svelte/icons/search';

	interface Props {
		/** Bindable selected location id ('' = none). */
		value?: string;
		locations: Location[];
		/** Called after a new location is created (so the parent can refresh). */
		onCreated?: (loc: Location) => void;
	}
	let { value = $bindable(''), locations, onCreated }: Props = $props();

	let adding = $state(false);
	let name = $state('');
	let query = $state('');
	let results = $state<GeocodeResult[]>([]);
	let picked = $state<GeocodeResult | null>(null);
	let searching = $state(false);
	let busy = $state(false);
	let err = $state('');

	const items = $derived<SelectItem[]>([
		{ value: '__none__', label: 'No location' },
		...locations.map((l) => ({ value: l.id, label: l.name }))
	]);

	let timer: ReturnType<typeof setTimeout> | undefined;
	function onQuery(e: Event) {
		query = (e.target as HTMLInputElement).value;
		picked = null;
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
		picked = r;
		results = [];
		query = r.displayName;
		if (!name.trim()) name = r.displayName.split(',')[0];
	}

	async function save() {
		if (!picked || !name.trim()) return;
		busy = true;
		err = '';
		try {
			const loc = await createLocation({ name: name.trim(), lat: picked.lat, lon: picked.lon, address: picked.displayName });
			value = loc.id;
			onCreated?.(loc);
			adding = false;
			name = '';
			query = '';
			picked = null;
		} catch (e) {
			err = e instanceof Error ? e.message : 'Could not save location';
		} finally {
			busy = false;
		}
	}

	const field = 'w-full rounded-md border border-rig-700 bg-rig-950 px-3 py-2 text-sm focus:border-rig-500 focus:outline-none';
</script>

<div class="space-y-2">
	<div class="flex items-center gap-2">
		<Select items={items} value={value || '__none__'} onValueChange={(v) => (value = v === '__none__' ? '' : v)} class="flex-1" />
		<button type="button" onclick={() => (adding = !adding)} class="whitespace-nowrap rounded-md border border-rig-700 px-3 py-2 text-sm text-rig-300 hover:border-rig-500">
			{adding ? 'Cancel' : '+ New'}
		</button>
	</div>

	{#if adding}
		<div class="space-y-3 rounded-lg border border-rig-800 bg-rig-950/60 p-3">
			<label class="block">
				<span class="text-xs text-rig-400">Location name</span>
				<input bind:value={name} placeholder="e.g. Home, Greenhouse" class="{field} mt-1" />
			</label>
			<label class="block">
				<span class="text-xs text-rig-400">Search address or place</span>
				<div class="relative mt-1">
					<Search size={14} class="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-rig-500" />
					<input value={query} oninput={onQuery} placeholder="Street, city, or point of interest" class="{field} pl-8" />
				</div>
			</label>

			{#if searching}
				<p class="text-xs text-rig-500">Searching…</p>
			{:else if results.length}
				<ul class="max-h-44 divide-y divide-rig-800 overflow-y-auto rounded-md border border-rig-800">
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

			{#if picked}
				<div class="flex items-center gap-2 rounded-md bg-rig-800/40 px-3 py-2 text-xs text-rig-300">
					<MapPin size={13} class="text-leaf" />
					<span class="tabular-nums">{picked.lat.toFixed(4)}, {picked.lon.toFixed(4)}</span>
				</div>
			{/if}

			{#if err}<p class="text-xs text-danger">{err}</p>{/if}

			<button
				type="button"
				onclick={save}
				disabled={busy || !picked || !name.trim()}
				class="rounded-md bg-rig-500 px-4 py-1.5 text-sm font-medium text-rig-950 hover:bg-rig-400 disabled:opacity-50"
			>
				{busy ? 'Saving…' : 'Add location'}
			</button>
		</div>
	{/if}
</div>
