<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { auth } from '$lib/auth.svelte';
	import {
		getGrow,
		getEnvironments,
		getStagePresets,
		changeStage,
		completeGrow,
		deleteGrow,
		createPlant,
		movePlant,
		updatePlant,
		repotPlant,
		harvestPlant,
		removePlant,
		getCultivars,
		cultivarImageURL,
		getRecipes,
		getCare,
		getCareConfig
	} from '$lib/api';
	import type { Environment, GrowDetail, PlantDetail, StagePresets, TrackingMode, PotUnit, Cultivar, CareActionDef, CareHistory, FeedingRecipe } from '$lib/types';
	import { titleCase, daysSince, defaultPlantLabel, plantDisplayName, plantNumbersById } from '$lib/format';
	import { fmtDate } from '$lib/datetime';
	import GrowFormModal from '$lib/components/GrowFormModal.svelte';
	import ActivityLog from '$lib/components/ActivityLog.svelte';
	import LogCareModal from '$lib/components/LogCareModal.svelte';
	import CareSummary from '$lib/components/CareSummary.svelte';
	import CareSettingsModal from '$lib/components/CareSettingsModal.svelte';
	import GrowAIChat from '$lib/components/GrowAIChat.svelte';
	import { Button, Dialog, Select } from '$lib/components/ui';
	import ArrowLeft from '@lucide/svelte/icons/arrow-left';
	import Sprout from '@lucide/svelte/icons/sprout';
	import Plus from '@lucide/svelte/icons/plus';
	import Pencil from '@lucide/svelte/icons/pencil';
	import Droplet from '@lucide/svelte/icons/droplet';
	import Settings2 from '@lucide/svelte/icons/settings-2';
	import ArrowRightLeft from '@lucide/svelte/icons/arrow-right-left';

	const id = $derived(page.params.id);
	const isAdmin = $derived(auth.isAdmin);

	let grow = $state<GrowDetail | null>(null);
	let environments = $state<Environment[]>([]);
	let presets = $state<StagePresets>({});
	let cultivars = $state<Cultivar[]>([]);
	let err = $state('');
	let loading = $state(true);

	// Cultivars defined for this grow's species, offered as suggestions when
	// binding a cultivar to plants (freeform entry is still allowed).
	const speciesCultivars = $derived(
		grow ? cultivars.filter((c) => c.species === grow!.species) : []
	);
	// Resolve a plant's cultivar name to its record for the row thumbnail.
	const cultivarByName = $derived(new Map(cultivars.map((c) => [c.name, c])));

	// --- care ---
	let care = $state<CareHistory | null>(null);
	// Effective care actions for this grow (species defaults overlaid with the
	// grow's customization). The log dialog uses only the enabled ones; the
	// settings editor works on the full list.
	let careDefs = $state<CareActionDef[]>([]);
	const careActions = $derived(careDefs.filter((d) => d.enabled));
	let recipes = $state<FeedingRecipe[]>([]);
	let careOpen = $state(false);
	let careInitialAction = $state<string | undefined>(undefined);
	let carePreselect = $state<string[]>([]);
	let careSettingsOpen = $state(false);

	async function reloadCare() {
		if (!id) return;
		try {
			care = await getCare(id);
		} catch {
			/* care history is non-critical; leave the last value */
		}
	}
	// Load the grow's effective care actions and the feeding recipes offered when
	// feeding. Runs once the grow (hence species) is known. Feeding uses only the
	// user's own recipes — built-in brand charts stay in the Knowledge library as
	// templates for creating recipes, not as things to log directly.
	async function loadCareActions(_species: string) {
		try {
			if (id) careDefs = (await getCareConfig(id)).actions;
			recipes = await getRecipes();
		} catch {
			/* non-critical */
		}
	}

	function openLogCare(actionKey?: string, plantIds: string[] = []) {
		careInitialAction = actionKey;
		carePreselect = plantIds;
		careOpen = true;
	}
	async function onCareLogged() {
		await Promise.all([reload(), reloadCare()]);
	}

	async function reload() {
		if (!id) return;
		try {
			grow = await getGrow(id);
			err = '';
			if (grow.species && careDefs.length === 0) loadCareActions(grow.species);
		} catch (e) {
			err = e instanceof Error ? e.message : 'Failed to load grow';
		} finally {
			loading = false;
		}
	}
	onMount(() => {
		reload();
		reloadCare();
		getEnvironments().then((e) => (environments = e)).catch(() => {});
		getStagePresets().then((p) => (presets = p)).catch(() => {});
		getCultivars().then((c) => (cultivars = c)).catch(() => {});
	});

	const envItems = $derived(environments.map((e) => ({ value: e.id, label: e.name })));

	// Cultivar dropdown items for this grow's species. `current` is included even
	// if it's a legacy freeform value not in the library, so editing never loses it.
	function cultivarItems(current: string) {
		const items = [{ value: '', label: '— None —' }];
		const names = new Set<string>();
		for (const c of speciesCultivars) {
			items.push({ value: c.name, label: c.name });
			names.add(c.name);
		}
		if (current && !names.has(current)) items.push({ value: current, label: `${current} (custom)` });
		return items;
	}
	const stageItems = $derived((grow?.stages ?? []).map((s) => ({ value: s, label: titleCase(s) })));
	const plantNumbers = $derived(plantNumbersById(grow?.plants ?? []));

	let editing = $state(false);
	let addingPlants = $state(false);

	async function advanceStage(stage: string) {
		if (!grow || stage === grow.stage) return;
		try {
			await changeStage(grow.id, stage);
			await reload();
		} catch (e) {
			err = e instanceof Error ? e.message : 'Failed';
		}
	}

	// --- move plant to another environment ---
	let moveOpen = $state(false);
	let movingPlant = $state<PlantDetail | null>(null);
	let mpEnv = $state('');
	let mpBusy = $state(false);
	function openMove(plant: PlantDetail) {
		movingPlant = plant;
		mpEnv = plant.currentEnvironmentId;
		moveOpen = true;
	}
	async function saveMove() {
		if (!movingPlant || !mpEnv || mpEnv === movingPlant.currentEnvironmentId) {
			moveOpen = false;
			return;
		}
		mpBusy = true;
		try {
			await movePlant(movingPlant.id, mpEnv);
			moveOpen = false;
			await reload();
		} catch (e) {
			err = e instanceof Error ? e.message : 'Failed';
		} finally {
			mpBusy = false;
		}
	}

	// --- edit a single plant unit (label / cultivar / group size) ---
	let editOpen = $state(false);
	let editingPlant = $state<PlantDetail | null>(null);
	let epLabel = $state('');
	let epCultivar = $state('');
	let epTracking = $state<TrackingMode>('individual');
	let epQuantity = $state(1);
	let epPotSize = $state<number | null>(null);
	let epPotUnit = $state<PotUnit>('L');
	let epPotType = $state('');
	let epBusy = $state(false);
	function openEdit(plant: PlantDetail) {
		editingPlant = plant;
		epLabel = plant.label === defaultPlantLabel(plant.tracking) ? '' : plant.label;
		epCultivar = plant.cultivar;
		epTracking = plant.tracking;
		epQuantity = plant.quantity;
		epPotSize = plant.currentPot?.size ?? null;
		epPotUnit = plant.currentPot?.unit ?? 'L';
		epPotType = plant.currentPot?.type ?? '';
		editOpen = true;
	}
	function potChanged(
		current: PlantDetail['currentPot'],
		size: number | null,
		unit: PotUnit,
		type: string
	): boolean {
		if (!size || size <= 0) return false;
		if (!current) return true;
		return current.size !== size || current.unit !== unit || (current.type ?? '') !== type;
	}
	async function saveEdit() {
		if (!editingPlant) return;
		epBusy = true;
		try {
			await updatePlant(editingPlant.id, {
				label: epLabel.trim(),
				cultivar: epCultivar.trim(),
				tracking: epTracking,
				quantity: epTracking === 'group' ? epQuantity : 1
			});
			if (potChanged(editingPlant.currentPot, epPotSize, epPotUnit, epPotType)) {
				await repotPlant(editingPlant.id, {
					size: epPotSize!,
					unit: epPotUnit,
					type: epPotType
				});
			}
			editOpen = false;
			await reload();
		} catch (e) {
			err = e instanceof Error ? e.message : 'Failed';
		} finally {
			epBusy = false;
		}
	}

	async function harvest(plant: PlantDetail) {
		if (!confirm(`Harvest ${plantDisplayName(plant, plantNumbers.get(plant.id))}?`)) return;
		try {
			await harvestPlant(plant.id);
			await reload();
		} catch (e) {
			err = e instanceof Error ? e.message : 'Failed';
		}
	}
	async function discard(plant: PlantDetail) {
		if (!confirm(`Remove ${plantDisplayName(plant, plantNumbers.get(plant.id))}?`)) return;
		try {
			await removePlant(plant.id);
			await reload();
		} catch (e) {
			err = e instanceof Error ? e.message : 'Failed';
		}
	}

	async function complete() {
		if (!grow || !confirm('Mark this grow as completed?')) return;
		try {
			await completeGrow(grow.id);
			await reload();
		} catch (e) {
			err = e instanceof Error ? e.message : 'Failed';
		}
	}
	async function destroy() {
		if (!grow || !confirm('Delete this grow and all its plants? This cannot be undone.')) return;
		try {
			await deleteGrow(grow.id);
			goto('/grows');
		} catch (e) {
			err = e instanceof Error ? e.message : 'Failed';
		}
	}

	// --- add plant form (one plant per submit: an individual, or a group) ---
	let apTracking = $state<TrackingMode>('individual');
	let apQuantity = $state(1);
	let apLabel = $state('');
	let apCultivar = $state('');
	let apEnv = $state('');
	let apPotSize = $state<number | null>(null);
	let apPotUnit = $state<PotUnit>('L');
	let apPotType = $state('');
	let apBusy = $state(false);
	const trackingItems = [
		{ value: 'individual', label: 'Individual plant' },
		{ value: 'group', label: 'Group (tray / bed / batch)' }
	];
	const potUnitItems = [
		{ value: 'L', label: 'liters (L)' },
		{ value: 'gal', label: 'gallons' }
	];
	const potTypeItems = [
		{ value: '', label: '—' },
		{ value: 'fabric', label: 'Fabric' },
		{ value: 'plastic', label: 'Plastic' },
		{ value: 'terracotta', label: 'Terracotta' },
		{ value: 'air-pot', label: 'Air pot' },
		{ value: 'other', label: 'Other' }
	];
	$effect(() => {
		if (addingPlants && !apEnv && environments.length) apEnv = environments[0].id;
	});
	async function addPlant() {
		if (!grow) return;
		apBusy = true;
		try {
			await createPlant(grow.id, {
				tracking: apTracking,
				quantity: apTracking === 'group' ? apQuantity : 1,
				label: apLabel.trim() || undefined,
				cultivar: apCultivar,
				environmentId: apEnv,
				...(apPotSize && apPotSize > 0
					? { potSize: apPotSize, potUnit: apPotUnit, potType: apPotType }
					: {})
			});
			addingPlants = false;
			apLabel = '';
			apCultivar = '';
			apQuantity = 1;
			apPotSize = null;
			apPotType = '';
			await reload();
		} catch (e) {
			err = e instanceof Error ? e.message : 'Failed';
		} finally {
			apBusy = false;
		}
	}

	const statusTone = (s: string) =>
		s === 'active' ? 'text-leaf' : s === 'harvested' ? 'text-warn' : 'text-rig-500';
	const field =
		'w-full rounded-md border border-rig-700 bg-rig-950 px-3 py-1.5 text-sm focus:border-rig-500 focus:outline-none';
</script>

<a href="/grows" class="mb-4 inline-flex items-center gap-1 text-sm text-rig-400 hover:text-rig-100">
	<ArrowLeft size={15} /> All grows
</a>

{#if loading}
	<p class="text-rig-400">Loading…</p>
{:else if !grow}
	<p class="text-rig-400">Grow not found. <a href="/grows" class="text-leaf hover:underline">Go back</a></p>
{:else}
	<div class="space-y-6">
		{#if err}<p class="rounded-md border border-danger/40 bg-danger/10 px-3 py-2 text-sm text-danger">{err}</p>{/if}

		<div class="flex flex-wrap items-start justify-between gap-3">
			<div>
				<div class="flex items-center gap-3">
					<h1 class="text-2xl font-semibold">{grow.name}</h1>
					<span class="rounded-full bg-rig-800 px-2 py-0.5 text-xs capitalize {statusTone(grow.status)}">{grow.status}</span>
				</div>
				<p class="mt-1 text-sm text-rig-400">
					{titleCase(grow.species) || 'No species set'}
					· started {fmtDate(grow.startedAt)} (day {grow.totalDays})
				</p>
			</div>
			{#if isAdmin}
				<div class="flex flex-wrap items-center gap-2">
					<Button variant="ghost" onclick={() => (editing = true)}>Edit</Button>
					{#if grow.status === 'active'}
						<Button variant="secondary" onclick={complete}>Complete</Button>
					{/if}
					<Button variant="ghost" onclick={destroy}>Delete</Button>
				</div>
			{/if}
		</div>

		<!-- Stage & timeline -->
		<section class="rounded-xl border border-rig-800 bg-rig-900/40 p-4">
			<div class="flex flex-wrap items-center justify-between gap-3">
				<div>
					<div class="text-xs uppercase tracking-wide text-rig-500">Current stage</div>
					<div class="mt-0.5 text-lg font-semibold capitalize">{grow.stage || '—'} <span class="text-sm font-normal text-rig-400">· {grow.stageDays}d</span></div>
				</div>
				{#if isAdmin && grow.status === 'active'}
					<label class="flex items-center gap-2 text-sm">
						<span class="text-rig-400">Advance to</span>
						<Select value={grow.stage} onValueChange={advanceStage} items={stageItems} />
					</label>
				{/if}
			</div>
			<!-- Stage sequence -->
			<div class="mt-4 flex flex-wrap gap-1.5">
				{#each grow.stages as st, i (st)}
					{@const current = st === grow.stage}
					<span
						class="rounded-full px-2.5 py-0.5 text-xs capitalize {current
							? 'bg-leaf/20 text-leaf'
							: 'bg-rig-800 text-rig-400'}"
					>
						{i + 1}. {st}
					</span>
				{/each}
			</div>
		</section>

		<!-- Plants -->
		<section>
			<div class="mb-3 flex items-center justify-between">
				<h2 class="text-sm font-semibold uppercase tracking-wide text-rig-400">Plants · {grow.plantCount} active</h2>
				{#if isAdmin}
					<div class="flex items-center gap-2">
						{#if careActions.length > 0 && grow.plantCount > 0}
							<button
								onclick={() => openLogCare()}
								class="inline-flex items-center gap-1.5 rounded-md border border-rig-700 px-3 py-1.5 text-sm text-rig-300 transition-colors hover:border-rig-500"
							>
								<Droplet size={14} /> Log care
							</button>
						{/if}
						{#if careDefs.length > 0}
							<button
								onclick={() => (careSettingsOpen = true)}
								title="Care actions"
								aria-label="Care actions"
								class="inline-flex items-center rounded-md border border-rig-700 p-1.5 text-rig-400 transition-colors hover:border-rig-500 hover:text-rig-200"
							>
								<Settings2 size={15} />
							</button>
						{/if}
						<button
							onclick={() => (addingPlants = true)}
							class="inline-flex items-center gap-1.5 rounded-md border border-rig-700 px-3 py-1.5 text-sm text-rig-300 transition-colors hover:border-rig-500"
						>
							<Plus size={14} /> Add plant
						</button>
					</div>
				{/if}
			</div>
			{#if grow.plants.length === 0}
				<div class="rounded-xl border border-dashed border-rig-800 p-6 text-center text-sm text-rig-500">
					No plants yet.{#if isAdmin} Add some to start tracking placements.{/if}
				</div>
			{:else}
				<div class="overflow-x-auto rounded-xl border border-rig-800">
					<table class="w-full min-w-[36rem] text-sm">
						<thead class="border-b border-rig-800 text-left text-xs uppercase tracking-wide text-rig-500">
							<tr>
								<th class="px-4 py-2 font-medium">Plant</th>
								<th class="px-4 py-2 font-medium">Status</th>
								<th class="px-4 py-2 font-medium">Location</th>
								<th class="px-4 py-2 font-medium">Pot</th>
								<th class="px-4 py-2 font-medium">Age</th>
								{#if isAdmin}<th class="px-4 py-2 font-medium">Actions</th>{/if}
							</tr>
						</thead>
						<tbody>
							{#each grow.plants as p (p.id)}
								{@const cv = cultivarByName.get(p.cultivar)}
								<tr class="border-b border-rig-800/60 last:border-0">
									<td class="px-4 py-2">
										<div class="flex items-center gap-3">
											<div class="h-9 w-9 shrink-0 overflow-hidden rounded-full border border-rig-700 bg-rig-950">
												{#if cv?.imageType}
													<img src={cultivarImageURL(cv.id)} alt={p.cultivar} class="h-full w-full object-cover" />
												{:else}
													<div class="flex h-full w-full items-center justify-center text-rig-600"><Sprout size={15} /></div>
												{/if}
											</div>
											<div class="min-w-0 truncate">
												<a href="/plants/{p.id}" class="font-medium hover:text-leaf">{plantDisplayName(p, plantNumbers.get(p.id))}</a>
												{#if p.tracking === 'group' && p.quantity > 1}<span class="ml-1 text-xs text-rig-500">×{p.quantity}</span>{/if}
											</div>
										</div>
									</td>
									<td class="px-4 py-2 capitalize {statusTone(p.status)}">{p.status}</td>
									<td class="px-4 py-2 text-rig-300">
										<div class="flex items-center gap-1.5">
											{#if p.currentEnvironmentId}
												<a href="/env/{p.currentEnvironmentId}" class="truncate hover:text-leaf hover:underline">
													{p.currentEnvironmentName || p.currentEnvironmentId}
												</a>
											{:else}
												<span>—</span>
											{/if}
											{#if isAdmin && p.status === 'active'}
												<button
													onclick={() => openMove(p)}
													title="Change location"
													aria-label="Change location"
													class="shrink-0 rounded-md border border-rig-700 p-1 text-rig-400 hover:border-rig-500 hover:text-rig-200"
												>
													<ArrowRightLeft size={13} />
												</button>
											{/if}
										</div>
									</td>
									<td class="px-4 py-2 tabular-nums text-rig-300">{p.currentPot ? `${p.currentPot.size} ${p.currentPot.unit}` : '—'}</td>
									<td class="px-4 py-2 tabular-nums text-rig-400">{daysSince(p.createdAt)}d</td>
									{#if isAdmin}
										<td class="px-4 py-2">
											<div class="flex items-center gap-1.5">
												{#if p.status === 'active' && careActions.length > 0}
													<button onclick={() => openLogCare(undefined, [p.id])} title="Log care" aria-label="Log care" class="rounded-md border border-rig-700 p-1.5 text-rig-400 hover:border-sky-500/60 hover:text-sky-300"><Droplet size={14} /></button>
												{/if}
												<button onclick={() => openEdit(p)} title="Edit plant" aria-label="Edit plant" class="rounded-md border border-rig-700 p-1.5 text-rig-400 hover:border-rig-500 hover:text-rig-200"><Pencil size={14} /></button>
												{#if p.status === 'active'}
													<button onclick={() => harvest(p)} class="rounded-md border border-rig-700 px-2 py-1 text-xs text-rig-300 hover:border-rig-500">Harvest</button>
													<button onclick={() => discard(p)} class="rounded-md border border-rig-700 px-2 py-1 text-xs text-rig-400 hover:border-danger/60 hover:text-danger">Remove</button>
												{/if}
											</div>
										</td>
									{/if}
								</tr>
							{/each}
						</tbody>
					</table>
				</div>
			{/if}
		</section>

		{#if grow.notes}
			<section>
				<h2 class="mb-2 text-sm font-semibold uppercase tracking-wide text-rig-400">Notes</h2>
				<p class="whitespace-pre-wrap rounded-xl border border-rig-800 bg-rig-950/40 p-4 text-sm text-rig-300">{grow.notes}</p>
			</section>
		{/if}

		<!-- Care summary -->
		{#if care && careActions.length > 0 && grow.plantCount > 0}
			<CareSummary
				{care}
				actions={careActions}
				canWrite={isAdmin}
				onQuick={(key) => openLogCare(key)}
				onLog={() => openLogCare()}
			/>
		{/if}

		<GrowAIChat growId={grow.id} growName={grow.name} />

		<!-- Activity log -->
		<section>
			<ActivityLog growId={grow.id} limit={30} title="Activity Log" />
		</section>
	</div>

	{#if isAdmin}
		<GrowFormModal bind:open={editing} grow={grow} {presets} onSaved={reload} />

		<LogCareModal
			bind:open={careOpen}
			{grow}
			actions={careActions}
			{recipes}
			preselectedPlantIds={carePreselect}
			initialActionKey={careInitialAction}
			onLogged={onCareLogged}
		/>

		<CareSettingsModal
			bind:open={careSettingsOpen}
			growId={grow.id}
			actions={careDefs}
			onSaved={(a) => (careDefs = a)}
		/>

		<Dialog bind:open={moveOpen} title="Change location" description="Move this plant to another environment. Its placement history is kept.">
			<div class="space-y-4">
				<label class="block">
					<span class="text-xs text-rig-400">Environment</span>
					<Select value={mpEnv} onValueChange={(v) => (mpEnv = v)} items={envItems} class="mt-1" />
				</label>
				<div class="flex justify-end gap-2 border-t border-rig-800 pt-4">
					<Button variant="ghost" onclick={() => (moveOpen = false)}>Cancel</Button>
					<Button onclick={saveMove} disabled={mpBusy || !mpEnv || mpEnv === movingPlant?.currentEnvironmentId}>
						<ArrowRightLeft size={15} /> Move
					</Button>
				</div>
			</div>
		</Dialog>

		<Dialog bind:open={editOpen} title="Edit plant" description="Change this plant's type, label, cultivar and pot. Each plant keeps its own id and history.">
			<div class="space-y-4">
				<div class="grid gap-3 sm:grid-cols-2">
					<label class="block">
						<span class="text-xs text-rig-400">Type</span>
						<Select value={epTracking} onValueChange={(v) => (epTracking = v as TrackingMode)} items={trackingItems} class="mt-1" />
					</label>
					{#if epTracking === 'group'}
						<label class="block">
							<span class="text-xs text-rig-400">Plants in group</span>
							<input type="number" min="1" bind:value={epQuantity} class="{field} mt-1" />
						</label>
					{/if}
				</div>
				<div class="grid gap-3 sm:grid-cols-2">
					<label class="block">
						<span class="text-xs text-rig-400">Label <span class="text-rig-600">(optional)</span></span>
						<input bind:value={epLabel} placeholder="Plant" class="{field} mt-1" />
					</label>
					<label class="block">
						<span class="text-xs text-rig-400">Cultivar <span class="text-rig-600">(optional)</span></span>
						<Select value={epCultivar} onValueChange={(v) => (epCultivar = v)} items={cultivarItems(epCultivar)} class="mt-1" />
					</label>
				</div>
				<div class="grid gap-3 sm:grid-cols-3">
					<label class="block">
						<span class="text-xs text-rig-400">Pot size <span class="text-rig-600">(optional)</span></span>
						<input type="number" min="0" step="any" bind:value={epPotSize} placeholder="e.g. 11" class="{field} mt-1" />
					</label>
					<label class="block">
						<span class="text-xs text-rig-400">Unit</span>
						<Select value={epPotUnit} onValueChange={(v) => (epPotUnit = v as PotUnit)} items={potUnitItems} class="mt-1" />
					</label>
					<label class="block">
						<span class="text-xs text-rig-400">Pot type</span>
						<Select value={epPotType} onValueChange={(v) => (epPotType = v)} items={potTypeItems} class="mt-1" />
					</label>
				</div>
				<div class="flex justify-end gap-2 border-t border-rig-800 pt-4">
					<Button variant="ghost" onclick={() => (editOpen = false)}>Cancel</Button>
					<Button onclick={saveEdit} disabled={epBusy || (epTracking === 'group' && epQuantity < 1)}>Save</Button>
				</div>
			</div>
		</Dialog>

		<Dialog bind:open={addingPlants} title="Add a plant" description="Add one plant — an individual, or a group (tray / bed / batch). Each gets its own id and history.">
			<div class="space-y-4">
				<div class="grid gap-3 sm:grid-cols-2">
					<label class="block">
						<span class="text-xs text-rig-400">Type</span>
						<Select value={apTracking} onValueChange={(v) => (apTracking = v as TrackingMode)} items={trackingItems} class="mt-1" />
					</label>
					{#if apTracking === 'group'}
						<label class="block">
							<span class="text-xs text-rig-400">Plants in group</span>
							<input type="number" min="1" bind:value={apQuantity} class="{field} mt-1" />
						</label>
					{/if}
				</div>
				<div class="grid gap-3 sm:grid-cols-2">
					<label class="block">
						<span class="text-xs text-rig-400">Label <span class="text-rig-600">(optional)</span></span>
						<input bind:value={apLabel} placeholder={apTracking === 'group' ? 'Group' : 'Plant'} class="{field} mt-1" />
					</label>
					<label class="block">
						<span class="text-xs text-rig-400">Cultivar <span class="text-rig-600">(optional)</span></span>
						<Select value={apCultivar} onValueChange={(v) => (apCultivar = v)} items={cultivarItems(apCultivar)} class="mt-1" />
					</label>
				</div>
				<label class="block">
					<span class="text-xs text-rig-400">Place in</span>
					<Select value={apEnv} onValueChange={(v) => (apEnv = v)} items={envItems} class="mt-1" />
				</label>
				<div class="grid gap-3 sm:grid-cols-3">
					<label class="block">
						<span class="text-xs text-rig-400">Pot size <span class="text-rig-600">(optional)</span></span>
						<input type="number" min="0" step="any" bind:value={apPotSize} placeholder="e.g. 11" class="{field} mt-1" />
					</label>
					<label class="block">
						<span class="text-xs text-rig-400">Unit</span>
						<Select value={apPotUnit} onValueChange={(v) => (apPotUnit = v as PotUnit)} items={potUnitItems} class="mt-1" />
					</label>
					<label class="block">
						<span class="text-xs text-rig-400">Pot type</span>
						<Select value={apPotType} onValueChange={(v) => (apPotType = v)} items={potTypeItems} class="mt-1" />
					</label>
				</div>
				<div class="flex justify-end gap-2 border-t border-rig-800 pt-4">
					<Button variant="ghost" onclick={() => (addingPlants = false)}>Cancel</Button>
					<Button onclick={addPlant} disabled={apBusy || (apTracking === 'group' && apQuantity < 1)}><Sprout size={15} /> Add plant</Button>
				</div>
			</div>
		</Dialog>
	{/if}
{/if}
