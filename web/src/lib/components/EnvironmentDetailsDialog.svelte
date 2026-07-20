<script lang="ts">
	import { errMsg } from '$lib/errors';
	import { toast } from '$lib/toast.svelte';
	import type { Environment, EnvironmentKind, Location } from '$lib/types';
	import { updateEnvironment } from '$lib/api';
	import { volumeM3 } from '$lib/format';
	import { Button, Dialog, Select, type SelectItem } from '$lib/components/ui';
	import LocationField from '$lib/components/LocationField.svelte';

	interface Props {
		env: Environment;
		/** Rooms available as air sources (may include env itself; it's filtered out). */
		rooms: Environment[];
		locations: Location[];
		/** Bindable open state. */
		open?: boolean;
		/** Called after a successful save. */
		onSaved?: () => void;
		/** Called when a location is created from the embedded picker. */
		onLocationCreated?: () => void;
	}
	let {
		env,
		rooms,
		locations,
		open = $bindable(false),
		onSaved,
		onLocationCreated
	}: Props = $props();

	let name = $state('');
	let kind = $state<EnvironmentKind>('tent');
	let model = $state('');
	let airSourceId = $state('');
	let locationId = $state('');
	let widthCm = $state(0);
	let depthCm = $state(0);
	let heightCm = $state(0);
	let busy = $state(false);
	let error = $state('');

	// Seed the form from the current env each time the dialog opens, so reopening
	// after a cancel (or for a different env) starts from the saved values.
	let seeded = false;
	$effect(() => {
		if (open && !seeded) {
			name = env.name;
			kind = env.kind;
			model = env.model;
			airSourceId = env.airSourceId;
			locationId = env.locationId;
			widthCm = env.widthCm;
			depthCm = env.depthCm;
			heightCm = env.heightCm;
			error = '';
			seeded = true;
		} else if (!open) {
			seeded = false;
		}
	});

	const otherRooms = $derived(rooms.filter((r) => r.id !== env.id));
	const kindItems: SelectItem[] = [
		{ value: 'tent', label: 'Tent' },
		{ value: 'room', label: 'Room' }
	];
	const airItems = $derived<SelectItem[]>([
		{ value: '__none__', label: 'None' },
		...otherRooms.map((r) => ({ value: r.id, label: r.name }))
	]);
	const volume = $derived(volumeM3(widthCm, depthCm, heightCm));
	const field =
		'w-full rounded-md border border-rig-700 bg-rig-950 px-3 py-2 text-sm focus:border-leaf focus:outline-none';

	async function save() {
		busy = true;
		error = '';
		try {
			await updateEnvironment(env.id, {
				name: name.trim(),
				kind,
				model: model.trim(),
				airSourceId: kind === 'tent' ? airSourceId : '',
				locationId,
				widthCm,
				depthCm,
				heightCm,
				// Climate targets and ranges are managed separately; preserve them.
				targetTempC: env.targetTempC,
				targetHumidity: env.targetHumidity,
				targetCO2: env.targetCO2,
				targetTempMinC: env.targetTempMinC,
				targetTempMaxC: env.targetTempMaxC,
				targetHumidityMin: env.targetHumidityMin,
				targetHumidityMax: env.targetHumidityMax,
				targetVpdMin: env.targetVpdMin,
				targetVpdMax: env.targetVpdMax,
				targetCo2Min: env.targetCo2Min,
				targetCo2Max: env.targetCo2Max,
				emergencyTempC: env.emergencyTempC,
				leafTempOffsetC: env.leafTempOffsetC ?? -2
			});
			open = false;
			toast.success('Environment updated', { description: name.trim() });
			onSaved?.();
		} catch (e) {
			error = errMsg(e, 'Save failed');
		} finally {
			busy = false;
		}
	}
</script>

<Dialog
	bind:open
	title="Edit environment"
	description="Update static environment details. Climate targets are managed separately."
>
	<div class="space-y-4">
		<label class="block"
			><span class="text-sm text-rig-400">Name</span><input bind:value={name} class="{field} mt-1" /></label
		>
		<label class="block"
			><span class="text-sm text-rig-400">Type</span><Select
				items={kindItems}
				value={kind}
				onValueChange={(value) => (kind = value as EnvironmentKind)}
				class="mt-1"
			/></label
		>
		<label class="block"
			><span class="text-sm text-rig-400">{kind === 'tent' ? 'Tent model' : 'Model'}</span><input
				bind:value={model}
				class="{field} mt-1"
			/></label
		>
		<div>
			<span class="text-sm text-rig-400">Location</span>
			<div class="mt-1"><LocationField bind:value={locationId} {locations} onCreated={onLocationCreated} /></div>
			<p class="mt-1 text-xs text-rig-500">Used for local weather and dashboard grouping.</p>
		</div>
		{#if kind === 'tent'}
			<div>
				<span class="text-sm text-rig-400">Dimensions (cm)</span>
				<div class="mt-1 grid grid-cols-[1fr_auto_1fr_auto_1fr] items-center gap-2">
					<input type="number" min="0" bind:value={widthCm} placeholder="W" class={field} />
					<span class="text-rig-600">×</span>
					<input type="number" min="0" bind:value={depthCm} placeholder="D" class={field} />
					<span class="text-rig-600">×</span>
					<input type="number" min="0" bind:value={heightCm} placeholder="H" class={field} />
				</div>
				<div class="mt-1 text-xs text-rig-500">Volume: {volume ? `${volume.toFixed(2)} m³` : '—'}</div>
			</div>
			<label class="block"
				><span class="text-sm text-rig-400">Air source</span><Select
					items={airItems}
					value={airSourceId || '__none__'}
					onValueChange={(value) => (airSourceId = value === '__none__' ? '' : value)}
					class="mt-1"
				/></label
			>
		{/if}
		{#if error}<p class="text-sm text-danger">{error}</p>{/if}
		<div class="flex justify-end gap-2 pt-2">
			<Button variant="ghost" onclick={() => (open = false)}>Cancel</Button>
			<Button onclick={save} disabled={busy || !name.trim()}>{busy ? 'Saving…' : 'Save'}</Button>
		</div>
	</div>
</Dialog>
