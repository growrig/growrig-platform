<script lang="ts">
	import type { Location } from '$lib/types';
	import { Select, type SelectItem } from '$lib/components/ui';
	import NewLocationForm from '$lib/components/NewLocationForm.svelte';

	interface Props {
		/** Bindable selected location id ('' = none). */
		value?: string;
		locations: Location[];
		/** Called after a new location is created (so the parent can refresh). */
		onCreated?: (loc: Location) => void;
	}
	let { value = $bindable(''), locations, onCreated }: Props = $props();

	let adding = $state(false);

	const items = $derived<SelectItem[]>([
		{ value: '__none__', label: 'No location' },
		...locations.map((l) => ({ value: l.id, label: l.name }))
	]);

	function handleCreated(loc: Location) {
		value = loc.id;
		adding = false;
		onCreated?.(loc);
	}
</script>

<div class="space-y-2">
	<div class="flex items-center gap-2">
		<Select items={items} value={value || '__none__'} onValueChange={(v) => (value = v === '__none__' ? '' : v)} class="flex-1" />
		<button type="button" onclick={() => (adding = !adding)} class="whitespace-nowrap rounded-md border border-rig-700 px-3 py-2 text-sm text-rig-300 hover:border-leaf">
			{adding ? 'Cancel' : '+ New'}
		</button>
	</div>

	{#if adding}
		<div class="rounded-lg border border-rig-800 bg-rig-950/60 p-3">
			<NewLocationForm onSaved={handleCreated} />
		</div>
	{/if}
</div>
