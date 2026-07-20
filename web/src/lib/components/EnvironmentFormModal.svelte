<script lang="ts">
	import { onMount } from 'svelte';
	import { errMsg } from '$lib/errors';
	import { toast } from '$lib/toast.svelte';
	import type { CatalogProduct, Environment, EnvironmentKind, Location } from '$lib/types';
	import { createEnvironment, getCatalog, getEnvironments, getLocations } from '$lib/api';
	import { Button, Dialog, Select, type SelectGroup } from '$lib/components/ui';

	interface Props {
		open?: boolean;
		/** Pre-selected kind (defaults to grow box). */
		defaultKind?: EnvironmentKind;
		/** Pre-selected parent air source (a room id). */
		defaultAirSourceId?: string;
		/** Pre-selected location. */
		defaultLocationId?: string;
		onSaved?: (env: Environment) => void;
	}
	let {
		open = $bindable(false),
		defaultKind = 'tent',
		defaultAirSourceId = '',
		defaultLocationId = '',
		onSaved
	}: Props = $props();

	// Sensible climate defaults per kind — actual targets, devices and automation
	// are configured later on the environment itself, not here.
	const CLIMATE: Record<EnvironmentKind, { targetTempC: number; targetHumidity: number }> = {
		tent: { targetTempC: 24, targetHumidity: 55 },
		room: { targetTempC: 22, targetHumidity: 50 }
	};

	// Sentinel model selection: a catalog model id picks a predefined tent (with
	// locked dimensions); CUSTOM lets the grower type a model and enter dimensions.
	const CUSTOM = '__custom__';

	let kind = $state<EnvironmentKind>('tent');
	let name = $state('');
	let airSourceId = $state('');
	let locationId = $state('');
	let widthCm = $state<number | null>(null);
	let depthCm = $state<number | null>(null);
	let heightCm = $state<number | null>(null);
	let modelChoice = $state(''); // '' | CUSTOM | catalog model id
	let model = $state(''); // free-text model, used in custom mode
	let busy = $state(false);
	let err = $state('');

	let environments = $state<Environment[]>([]);
	let locations = $state<Location[]>([]);
	let catalog = $state<CatalogProduct[]>([]);
	onMount(() => {
		getEnvironments().then((e) => (environments = e)).catch(() => {});
		getLocations().then((l) => (locations = l)).catch(() => {});
		getCatalog().then((c) => (catalog = c)).catch(() => {});
	});

	// Reseed on open from the caller's defaults.
	$effect(() => {
		if (!open) return;
		kind = defaultKind;
		name = '';
		airSourceId = defaultAirSourceId;
		locationId = defaultLocationId;
		widthCm = depthCm = heightCm = null;
		modelChoice = '';
		model = '';
		err = '';
	});

	const kindItems = [
		{ value: 'tent', label: 'Grow box' },
		{ value: 'room', label: 'Lung room' }
	];
	// A parent air source is a room that supplies intake air.
	const roomItems = $derived([
		{ value: '', label: '— None —' },
		...environments.filter((e) => e.kind === 'room').map((e) => ({ value: e.id, label: e.name }))
	]);
	const locationItems = $derived([
		{ value: '', label: '— None —' },
		...locations.map((l) => ({ value: l.id, label: l.name }))
	]);

	// A tent inherits its location from the room that supplies its air, so when an
	// air source is chosen the location is fixed to the room's and shown read-only.
	const airSource = $derived(environments.find((e) => e.id === airSourceId));
	const locationLocked = $derived(!!airSourceId);
	const effectiveLocationId = $derived(locationLocked ? (airSource?.locationId ?? '') : locationId);

	// Flatten every tent driver into concrete, dimensioned models. Each catalog
	// tent product groups brand "series"; the selectable leaves carry the specs.
	type TentModel = { id: string; brand: string; label: string; w: number; d: number; h: number };
	const tentModels = $derived.by<TentModel[]>(() => {
		const out: TentModel[] = [];
		for (const driver of catalog) {
			if (driver.category !== 'tent') continue;
			for (const series of driver.products ?? []) {
				const brand = series.brand ?? driver.brand;
				const leaves = series.models?.length ? series.models : [series];
				for (const m of leaves) {
					const s = m.specs;
					if (!s?.widthCm || !s?.depthCm || !s?.heightCm) continue;
					out.push({
						id: m.id,
						brand,
						label: `${m.model ?? series.model ?? ''}`.trim() || m.id,
						w: s.widthCm,
						d: s.depthCm,
						h: s.heightCm
					});
				}
			}
		}
		return out;
	});
	// Model picker: "Custom size" first, then predefined models grouped by brand.
	const modelGroups = $derived.by<SelectGroup[]>(() => {
		const byBrand = new Map<string, { value: string; label: string }[]>();
		for (const m of tentModels) {
			const items = byBrand.get(m.brand) ?? [];
			items.push({ value: m.id, label: m.label });
			byBrand.set(m.brand, items);
		}
		return [
			{ label: '', items: [{ value: CUSTOM, label: 'Custom size' }] },
			...[...byBrand].map(([label, items]) => ({ label, items }))
		];
	});

	// A predefined model is selected → dimensions are prefilled and locked.
	const dimsLocked = $derived(kind === 'tent' && !!modelChoice && modelChoice !== CUSTOM);

	function onModelChange(v: string) {
		modelChoice = v;
		if (v === CUSTOM || v === '') return;
		const m = tentModels.find((t) => t.id === v);
		if (!m) return;
		model = `${m.brand} ${m.label}`.trim();
		widthCm = m.w;
		depthCm = m.d;
		heightCm = m.h;
	}

	// The model string persisted: predefined label, or the grower's custom text.
	const savedModel = $derived.by(() => {
		if (kind !== 'tent') return undefined;
		if (dimsLocked) return model;
		return modelChoice === CUSTOM ? model.trim() : '';
	});

	const canSave = $derived(!!name.trim() && !busy);

	async function save() {
		if (!canSave) return;
		busy = true;
		err = '';
		try {
			const saved = await createEnvironment({
				name: name.trim(),
				kind,
				airSourceId: airSourceId || '',
				locationId: effectiveLocationId || undefined,
				model: savedModel,
				widthCm: widthCm ?? 0,
				depthCm: depthCm ?? 0,
				heightCm: heightCm ?? 0,
				targetTempC: CLIMATE[kind].targetTempC,
				targetHumidity: CLIMATE[kind].targetHumidity,
				targetCO2: 0,
				emergencyTempC: 35,
				leafTempOffsetC: -2
			});
			open = false;
			toast.success(kind === 'room' ? 'Lung room created' : 'Grow box created', {
				description: saved.name
			});
			onSaved?.(saved);
		} catch (e) {
			err = errMsg(e, 'Failed to create environment');
		} finally {
			busy = false;
		}
	}

	const field =
		'w-full rounded-md border border-rig-700 bg-rig-950 px-3 py-1.5 text-sm focus:border-leaf focus:outline-none';
	const fieldLocked =
		'w-full rounded-md border border-rig-800 bg-rig-900 px-3 py-1.5 text-sm text-rig-400 cursor-not-allowed';
</script>

<Dialog
	bind:open
	title="New environment"
	description="A grow box or lung room. Devices, climate targets and a control grow are configured on the environment afterwards."
>
	<div class="space-y-4">
		{#if err}<p class="text-xs text-danger">{err}</p>{/if}

		<div class="grid gap-3 sm:grid-cols-2">
			<label class="block">
				<span class="text-xs text-rig-400">Type</span>
				<Select class="mt-1" value={kind} onValueChange={(v) => (kind = v as EnvironmentKind)} items={kindItems} />
			</label>
			<label class="block">
				<span class="text-xs text-rig-400">Name</span>
				<input bind:value={name} placeholder={kind === 'room' ? 'e.g. Lung Room' : 'e.g. Main Grow Tent'} class="{field} mt-1" />
			</label>
		</div>

		<div class="grid gap-3 sm:grid-cols-2">
			<label class="block">
				<span class="text-xs text-rig-400">Air source <span class="text-rig-600">(parent)</span></span>
				<Select class="mt-1" bind:value={airSourceId} placeholder="— None —" items={roomItems} />
				<span class="mt-1 block text-xs text-rig-500">The room that supplies this environment's intake air.</span>
			</label>
			<label class="block">
				<span class="text-xs text-rig-400">Location <span class="text-rig-600">(optional)</span></span>
				<Select
					class="mt-1"
					value={effectiveLocationId}
					onValueChange={(v) => (locationId = v)}
					disabled={locationLocked}
					placeholder="— None —"
					items={locationItems}
				/>
				{#if locationLocked}
					<span class="mt-1 block text-xs text-rig-500">Inherited from the air source.</span>
				{/if}
			</label>
		</div>

		{#if kind === 'tent'}
			<label class="block">
				<span class="text-xs text-rig-400">Model <span class="text-rig-600">(optional)</span></span>
				<Select
					class="mt-1"
					value={modelChoice}
					onValueChange={onModelChange}
					placeholder="Select a model…"
					groups={modelGroups}
				/>
			</label>
			{#if modelChoice === CUSTOM}
				<label class="block">
					<span class="text-xs text-rig-400">Model name <span class="text-rig-600">(optional)</span></span>
					<input bind:value={model} placeholder="e.g. Mars Hydro 4×4" class="{field} mt-1" />
				</label>
			{/if}
		{/if}

		<label class="block">
			<span class="text-xs text-rig-400">Dimensions (cm) <span class="text-rig-600">(optional)</span></span>
			<div class="mt-1 flex items-center gap-2">
				<input type="number" min="0" step="1" bind:value={widthCm} placeholder="W" readonly={dimsLocked} class={dimsLocked ? fieldLocked : field} />
				<span class="text-rig-600">×</span>
				<input type="number" min="0" step="1" bind:value={depthCm} placeholder="D" readonly={dimsLocked} class={dimsLocked ? fieldLocked : field} />
				<span class="text-rig-600">×</span>
				<input type="number" min="0" step="1" bind:value={heightCm} placeholder="H" readonly={dimsLocked} class={dimsLocked ? fieldLocked : field} />
			</div>
			{#if dimsLocked}
				<span class="mt-1 block text-xs text-rig-500">Set by the selected model. Choose “Custom size” to edit.</span>
			{/if}
		</label>

		<div class="flex justify-end gap-2 border-t border-rig-800 pt-4">
			<Button variant="ghost" onclick={() => (open = false)} disabled={busy}>Cancel</Button>
			<Button onclick={save} disabled={!canSave}>Create environment</Button>
		</div>
	</div>
</Dialog>
