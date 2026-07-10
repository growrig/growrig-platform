<script lang="ts">
	import type { Environment } from '$lib/types';
	import { updateEnvironment, deleteEnvironment } from '$lib/api';
	import { volumeM3 } from '$lib/format';
	import { Button, Select, Slider, type SelectItem } from '$lib/components/ui';

	interface Props {
		env: Environment;
		rooms: Environment[]; // candidate air sources
		canDelete: boolean;
		onChanged: () => void;
		flash: (kind: 'ok' | 'err', text: string) => void;
	}
	let { env, rooms, canDelete, onChanged, flash }: Props = $props();

	// Editable drafts seeded from the initial prop values (intentional).
	// svelte-ignore state_referenced_locally
	let name = $state(env.name);
	// svelte-ignore state_referenced_locally
	let kind = $state(env.kind);
	// svelte-ignore state_referenced_locally
	let model = $state(env.model);
	// svelte-ignore state_referenced_locally
	let airSourceId = $state(env.airSourceId);
	// svelte-ignore state_referenced_locally
	let temp = $state(env.targetTempC);
	// svelte-ignore state_referenced_locally
	let humidity = $state(env.targetHumidity);
	// svelte-ignore state_referenced_locally
	let co2 = $state(env.targetCO2);
	// svelte-ignore state_referenced_locally
	let emergency = $state(env.emergencyTempC);
	// svelte-ignore state_referenced_locally
	let widthCm = $state(env.widthCm);
	// svelte-ignore state_referenced_locally
	let depthCm = $state(env.depthCm);
	// svelte-ignore state_referenced_locally
	let heightCm = $state(env.heightCm);
	let busy = $state(false);

	const otherRooms = $derived(rooms.filter((r) => r.id !== env.id));
	const volume = $derived(volumeM3(widthCm, depthCm, heightCm));

	const kindItems: SelectItem[] = [
		{ value: 'tent', label: 'Tent' },
		{ value: 'room', label: 'Room' }
	];
	const airItems = $derived<SelectItem[]>([
		{ value: '', label: 'None' },
		...otherRooms.map((r) => ({ value: r.id, label: r.name }))
	]);

	async function save() {
		busy = true;
		try {
			await updateEnvironment(env.id, {
				name,
				kind,
				airSourceId: kind === 'tent' ? airSourceId : '',
				model,
				widthCm,
				depthCm,
				heightCm,
				targetTempC: temp,
				targetHumidity: humidity,
				targetCO2: co2,
				emergencyTempC: emergency
			});
			flash('ok', 'Environment saved');
			onChanged();
		} catch (e) {
			flash('err', e instanceof Error ? e.message : 'Save failed');
		} finally {
			busy = false;
		}
	}

	async function remove() {
		if (!confirm(`Delete "${env.name}"?`)) return;
		try {
			await deleteEnvironment(env.id);
			flash('ok', 'Environment deleted');
			onChanged();
		} catch (e) {
			flash('err', e instanceof Error ? e.message : 'Delete failed');
		}
	}

	const field =
		'rounded-md border border-rig-700 bg-rig-950 px-3 py-1.5 text-sm focus:border-rig-500 focus:outline-none';
</script>

<section class="rounded-xl border border-rig-800 bg-rig-900/40 p-5">
	<div class="mb-4 flex flex-wrap items-center gap-3">
		<input bind:value={name} class="{field} flex-1 text-lg font-semibold" />
		<Select
			items={kindItems}
			value={kind}
			onValueChange={(v) => (kind = v as typeof kind)}
			class="w-28"
		/>
		<span class="text-xs text-rig-500">{env.id}</span>
	</div>

	<label class="mb-4 flex items-center gap-3">
		<span class="w-32 shrink-0 text-sm text-rig-400">{kind === 'tent' ? 'Tent model' : 'Model'}</span>
		<input bind:value={model} placeholder="e.g. MARS HYDRO Grow Tent" class="{field} flex-1" />
	</label>

	{#if kind === 'tent'}
		<div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
			<div class="block">
				<span class="text-sm text-rig-400">Target temp — {temp}°C</span>
				<Slider min={15} max={35} step={0.5} bind:value={temp} class="mt-3" />
			</div>
			<div class="block">
				<span class="text-sm text-rig-400">Target humidity — {humidity}%</span>
				<Slider min={20} max={90} step={1} bind:value={humidity} class="mt-3" />
			</div>
			<div class="block">
				<span class="text-sm text-rig-400">Target CO₂ — {co2 || 'off'}{co2 ? ' ppm' : ''}</span>
				<Slider min={0} max={1500} step={50} bind:value={co2} class="mt-3" />
			</div>
			<div class="block">
				<span class="text-sm text-rig-400">Emergency temp — {emergency}°C</span>
				<Slider min={28} max={45} step={0.5} bind:value={emergency} tone="warn" class="mt-3" />
			</div>
		</div>
		<div class="mt-4 flex flex-wrap items-end gap-4">
			<div>
				<span class="text-sm text-rig-400">Dimensions (cm)</span>
				<div class="mt-2 flex items-center gap-2">
					<input type="number" min="0" step="1" bind:value={widthCm} placeholder="W" class="{field} w-20" />
					<span class="text-rig-600">×</span>
					<input type="number" min="0" step="1" bind:value={depthCm} placeholder="D" class="{field} w-20" />
					<span class="text-rig-600">×</span>
					<input type="number" min="0" step="1" bind:value={heightCm} placeholder="H" class="{field} w-20" />
				</div>
			</div>
			<div class="pb-1.5 text-sm text-rig-400">
				Volume: <span class="font-semibold text-rig-100 tabular-nums">{volume ? `${volume.toFixed(2)} m³` : '—'}</span>
			</div>
		</div>
		<label class="mt-4 flex items-center gap-3">
			<span class="text-sm text-rig-400">Air source (lung room)</span>
			<Select items={airItems} bind:value={airSourceId} placeholder="None" class="w-52" />
		</label>
	{:else}
		<p class="text-sm text-rig-400">Monitored room. Can be linked to a tent as its air source.</p>
	{/if}

	<div class="mt-4 flex gap-2">
		<Button onclick={save} disabled={busy}>Save</Button>
		{#if canDelete}
			<Button
				variant="secondary"
				onclick={remove}
				class="hover:border-danger hover:text-danger"
			>
				Delete
			</Button>
		{/if}
	</div>
</section>
