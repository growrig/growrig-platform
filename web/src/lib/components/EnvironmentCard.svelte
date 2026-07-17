<script lang="ts">
	import type { Environment, Location } from '$lib/types';
	import { formatDimensions, volumeM3 } from '$lib/format';
	import { Button } from '$lib/components/ui';
	import EnvironmentDetailsDialog from '$lib/components/EnvironmentDetailsDialog.svelte';
	import Pencil from '@lucide/svelte/icons/pencil';

	interface Props {
		env: Environment;
		rooms: Environment[];
		locations: Location[];
		onChanged: () => void;
		flash: (kind: 'ok' | 'err', text: string) => void;
	}
	let { env, rooms, locations, onChanged, flash }: Props = $props();

	let editOpen = $state(false);

	const roomName = $derived(rooms.find((room) => room.id === env.airSourceId)?.name);
	const locationName = $derived(locations.find((l) => l.id === env.locationId)?.name);
	const dimensions = $derived(formatDimensions(env.widthCm, env.depthCm, env.heightCm));

	function onDetailsSaved() {
		flash('ok', 'Environment details saved');
		onChanged();
	}
</script>

<section class="space-y-3">
	<div class="flex items-center justify-between">
		<h2 class="text-sm font-semibold uppercase tracking-wide text-rig-400">Environment</h2>
		<Button variant="secondary" size="sm" onclick={() => (editOpen = true)}><Pencil size={14} /> Edit</Button>
	</div>
	<div class="rounded-xl border border-rig-800 bg-rig-900/40 p-5">
		<div class="grid gap-x-8 gap-y-4 sm:grid-cols-2 lg:grid-cols-4">
			<div><div class="text-xs text-rig-500">Name</div><div class="mt-1 text-sm font-medium">{env.name}</div></div>
			<div><div class="text-xs text-rig-500">Type</div><div class="mt-1 text-sm capitalize">{env.kind}</div></div>
			<div><div class="text-xs text-rig-500">{env.kind === 'tent' ? 'Tent model' : 'Model'}</div><div class="mt-1 text-sm">{env.model || '—'}</div></div>
			<div><div class="text-xs text-rig-500">Location</div><div class="mt-1 text-sm">{locationName || 'None'}</div></div>
			<div><div class="text-xs text-rig-500">ID</div><div class="mt-1 truncate font-mono text-xs text-rig-300">{env.id}</div></div>
			{#if env.kind === 'tent'}
				<div><div class="text-xs text-rig-500">Dimensions</div><div class="mt-1 text-sm">{dimensions || '—'}</div></div>
				<div><div class="text-xs text-rig-500">Volume</div><div class="mt-1 text-sm">{volumeM3(env.widthCm, env.depthCm, env.heightCm) ? `${volumeM3(env.widthCm, env.depthCm, env.heightCm).toFixed(2)} m³` : '—'}</div></div>
				<div><div class="text-xs text-rig-500">Air source</div><div class="mt-1 text-sm">{roomName || 'None'}</div></div>
			{/if}
		</div>
	</div>
</section>

<EnvironmentDetailsDialog {env} {rooms} {locations} bind:open={editOpen} onSaved={onDetailsSaved} onLocationCreated={onChanged} />
