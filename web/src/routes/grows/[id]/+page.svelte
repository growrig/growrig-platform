<script lang="ts">
	import { errMsg } from '$lib/errors';
	import { onMount } from 'svelte';
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { auth } from '$lib/auth.svelte';
	import { live } from '$lib/live.svelte';
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
		getRecipes,
		getCare,
		getCareConfig,
		getLocations,
		getLightingDefaults,
		getGrowPhotos,
		getGrowAnalytics,
		getActivity,
		getAlerts,
		getTasks,
		historyRange,
		deviceHistory,
		weather
	} from '$lib/api';
	import type {
		Activity,
		Alert,
		Cultivar,
		CareActionDef,
		CareHistory,
		DeviceSeries,
		Environment,
		FeedingRecipe,
		GrowAnalytics,
		GrowDetail,
		GrowPhoto,
		Location,
		PlantDetail,
		PotUnit,
		Reading,
		StagePresets,
		Task,
		TrackingMode,
		Weather
	} from '$lib/types';
	import { defaultPlantLabel, plantDisplayName, plantNumbersById } from '$lib/format';
	import { resolveLocationId } from '$lib/location';
	import GrowFormModal from '$lib/components/GrowFormModal.svelte';
	import LogCareModal from '$lib/components/LogCareModal.svelte';
	import CareSettingsModal from '$lib/components/CareSettingsModal.svelte';
	import { Button, Dialog, Select } from '$lib/components/ui';
	import GrowHeader from '$lib/components/grow/GrowHeader.svelte';
	import OverviewTab from '$lib/components/grow/OverviewTab.svelte';
	import PlantsTab from '$lib/components/grow/PlantsTab.svelte';
	import PlanTab from '$lib/components/grow/PlanTab.svelte';
	import AnalyticsTab from '$lib/components/grow/AnalyticsTab.svelte';
	import TimelineTab from '$lib/components/grow/TimelineTab.svelte';
	import Sprout from '@lucide/svelte/icons/sprout';
	import ArrowRightLeft from '@lucide/svelte/icons/arrow-right-left';

	const id = $derived(page.params.id);
	const isAdmin = $derived(auth.isAdmin);

	let grow = $state<GrowDetail | null>(null);
	let environments = $state<Environment[]>([]);
	let presets = $state<StagePresets>({});
	let cultivars = $state<Cultivar[]>([]);
	let err = $state('');
	let loading = $state(true);

	// care
	let care = $state<CareHistory | null>(null);
	let careDefs = $state<CareActionDef[]>([]);
	const careActions = $derived(careDefs.filter((d) => d.enabled));
	let recipes = $state<FeedingRecipe[]>([]);
	let careOpen = $state(false);
	let careInitialAction = $state<string | undefined>(undefined);
	let carePreselect = $state<string[]>([]);
	let careSettingsOpen = $state(false);

	// profile data
	let photos = $state<GrowPhoto[]>([]);
	let analytics = $state<GrowAnalytics | null>(null);
	let activity = $state<Activity[]>([]);
	let alerts = $state<Alert[]>([]);
	let tasks = $state<Task[]>([]);

	// timeline (grow's primary environment)
	let rangeReadings = $state<Reading[]>([]);
	let deviceSeries = $state<DeviceSeries[]>([]);
	let weatherData = $state<Weather | undefined>();
	let locations = $state<Location[]>([]);
	let lightingDefaults = $state<Record<string, number>>({});
	let timelineHours = $state(168);

	let editing = $state(false);

	const canLogCare = $derived(careActions.length > 0 && (grow?.plantCount ?? 0) > 0);
	const growTasks = $derived(tasks.filter((t) => t.growId === id));
	const growAlerts = $derived(alerts.filter((a) => a.growId === id));

	// --- tabs ---
	type Tab = 'overview' | 'plants' | 'plan' | 'analytics' | 'timeline';
	const tabs: { id: Tab; label: string }[] = [
		{ id: 'overview', label: 'Overview' },
		{ id: 'plants', label: 'Plants' },
		{ id: 'plan', label: 'Plan' },
		{ id: 'analytics', label: 'Analytics' },
		{ id: 'timeline', label: 'Timeline' }
	];
	const activeTab = $derived.by<Tab>(() => {
		const t = page.url.searchParams.get('tab') as Tab | null;
		return t && tabs.some((x) => x.id === t) ? t : 'overview';
	});
	function setTab(t: Tab) {
		const url = new URL(page.url);
		if (t === 'overview') url.searchParams.delete('tab');
		else url.searchParams.set('tab', t);
		goto(url, { replaceState: true, keepFocus: true, noScroll: true });
	}

	const primaryEnvId = $derived(grow?.plants.find((p) => p.status === 'active')?.currentEnvironmentId ?? '');

	async function reloadCare() {
		if (!id) return;
		try {
			care = await getCare(id);
		} catch {
			/* non-critical */
		}
	}
	async function loadCareActions() {
		try {
			if (id) careDefs = (await getCareConfig(id)).actions;
			recipes = await getRecipes();
		} catch {
			/* non-critical */
		}
	}
	function reloadPhotos() {
		if (id) getGrowPhotos(id).then((p) => (photos = p)).catch(() => {});
	}
	function reloadAnalytics() {
		if (id) getGrowAnalytics(id).then((a) => (analytics = a)).catch(() => {});
	}
	function reloadActivity() {
		if (id) getActivity({ growId: id, limit: 100 }).then((p) => (activity = p.items)).catch(() => {});
	}
	function reloadAttention() {
		getAlerts().then((a) => (alerts = a)).catch(() => {});
		getTasks('open').then((t) => (tasks = t)).catch(() => {});
	}

	async function reload() {
		if (!id) return;
		try {
			grow = await getGrow(id);
			err = '';
			if (grow.species && careDefs.length === 0) loadCareActions();
		} catch (e) {
			err = errMsg(e, 'Failed to load grow');
		} finally {
			loading = false;
		}
	}

	async function refreshHistory() {
		if (!primaryEnvId) {
			rangeReadings = [];
			deviceSeries = [];
			return;
		}
		try {
			[rangeReadings, deviceSeries] = await Promise.all([
				historyRange(primaryEnvId, timelineHours, 500),
				deviceHistory(primaryEnvId, timelineHours, 500)
			]);
		} catch {
			/* keep last */
		}
	}
	function onRangeChange(h: number) {
		timelineHours = h;
		refreshHistory();
	}
	// Refetch history when the primary environment resolves/changes.
	$effect(() => {
		primaryEnvId;
		refreshHistory();
	});
	// Weather for the primary environment's location.
	$effect(() => {
		const env = live.snapshot?.environments?.find((e) => e.id === primaryEnvId);
		const locId = resolveLocationId(env, live.snapshot?.environments ?? []);
		const loc = locations.find((l) => l.id === locId);
		if (!loc) {
			weatherData = undefined;
			return;
		}
		weather(loc.lat, loc.lon).then((w) => (weatherData = w)).catch(() => {});
	});

	onMount(() => {
		reload();
		reloadCare();
		reloadPhotos();
		reloadAnalytics();
		reloadActivity();
		reloadAttention();
		getEnvironments().then((e) => (environments = e)).catch(() => {});
		getStagePresets().then((p) => (presets = p)).catch(() => {});
		getCultivars().then((c) => (cultivars = c)).catch(() => {});
		getLocations().then((l) => (locations = l)).catch(() => {});
		getLightingDefaults().then((d) => (lightingDefaults = d)).catch(() => {});
	});

	// --- care logging ---
	function openLogCare(actionKey?: string, plantIds: string[] = []) {
		careInitialAction = actionKey;
		carePreselect = plantIds;
		careOpen = true;
	}
	async function onCareLogged() {
		await Promise.all([reload(), reloadCare(), reloadActivity(), reloadAnalytics()]);
	}
	function onPhotoUploaded() {
		reloadPhotos();
		reloadActivity();
	}

	// --- stage / lifecycle ---
	async function advanceStage(stage: string) {
		if (!grow || stage === grow.stage) return;
		try {
			await changeStage(grow.id, stage);
			await Promise.all([reload(), reloadAnalytics(), reloadActivity()]);
		} catch (e) {
			err = errMsg(e, 'Failed');
		}
	}
	async function complete() {
		if (!grow || !confirm('Mark this grow as completed?')) return;
		try {
			await completeGrow(grow.id);
			await reload();
		} catch (e) {
			err = errMsg(e, 'Failed');
		}
	}
	async function destroy() {
		if (!grow || !confirm('Delete this grow and all its plants? This cannot be undone.')) return;
		try {
			await deleteGrow(grow.id);
			goto('/grows');
		} catch (e) {
			err = errMsg(e, 'Failed');
		}
	}

	const envItems = $derived(environments.map((e) => ({ value: e.id, label: e.name })));
	const speciesCultivars = $derived(grow ? cultivars.filter((c) => c.species === grow!.species) : []);
	const plantNumbers = $derived(plantNumbersById(grow?.plants ?? []));
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

	// --- move plant ---
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
			err = errMsg(e, 'Failed');
		} finally {
			mpBusy = false;
		}
	}

	// --- edit plant ---
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
	function potChanged(current: PlantDetail['currentPot'], size: number | null, unit: PotUnit, type: string): boolean {
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
				await repotPlant(editingPlant.id, { size: epPotSize!, unit: epPotUnit, type: epPotType });
			}
			editOpen = false;
			await reload();
		} catch (e) {
			err = errMsg(e, 'Failed');
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
			err = errMsg(e, 'Failed');
		}
	}
	async function discard(plant: PlantDetail) {
		if (!confirm(`Remove ${plantDisplayName(plant, plantNumbers.get(plant.id))}?`)) return;
		try {
			await removePlant(plant.id);
			await reload();
		} catch (e) {
			err = errMsg(e, 'Failed');
		}
	}

	// --- add plant ---
	let addingPlants = $state(false);
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
				...(apPotSize && apPotSize > 0 ? { potSize: apPotSize, potUnit: apPotUnit, potType: apPotType } : {})
			});
			addingPlants = false;
			apLabel = '';
			apCultivar = '';
			apQuantity = 1;
			apPotSize = null;
			apPotType = '';
			await reload();
		} catch (e) {
			err = errMsg(e, 'Failed');
		} finally {
			apBusy = false;
		}
	}

	const field = 'w-full rounded-md border border-rig-700 bg-rig-950 px-3 py-1.5 text-sm focus:border-rig-500 focus:outline-none';
</script>

{#if loading}
	<p class="text-rig-400">Loading…</p>
{:else if !grow}
	<p class="text-rig-400">Grow not found. <a href="/grows" class="text-leaf hover:underline">Go back</a></p>
{:else}
	<div class="space-y-6">
		{#if err}<p class="rounded-md border border-danger/40 bg-danger/10 px-3 py-2 text-sm text-danger">{err}</p>{/if}

		<GrowHeader
			{grow}
			{isAdmin}
			{canLogCare}
			dueCount={growTasks.length + growAlerts.length}
			onLogCare={() => openLogCare()}
			{onPhotoUploaded}
			onEdit={() => (editing = true)}
			onComplete={complete}
			onDelete={destroy}
			onCareSettings={() => (careSettingsOpen = true)}
		/>

		<div class="flex gap-1 overflow-x-auto border-b border-rig-800">
			{#each tabs as t (t.id)}
				<button
					onclick={() => setTab(t.id)}
					class="-mb-px shrink-0 border-b-2 px-4 py-2 text-sm font-medium transition-colors {activeTab === t.id ? 'border-leaf text-rig-50' : 'border-transparent text-rig-400 hover:text-rig-100'}"
				>
					{t.label}
				</button>
			{/each}
		</div>

		{#if activeTab === 'overview'}
			<OverviewTab
				{grow}
				{isAdmin}
				{photos}
				{care}
				{careActions}
				{analytics}
				{rangeReadings}
				{deviceSeries}
				{weatherData}
				defaults={lightingDefaults}
				{timelineHours}
				alerts={growAlerts}
				tasks={growTasks}
				{onRangeChange}
				onMoveStage={advanceStage}
				{onPhotoUploaded}
				onLogCare={() => openLogCare()}
				onQuickCare={(key) => openLogCare(key)}
			/>
		{:else if activeTab === 'plants'}
			<PlantsTab
				{grow}
				{isAdmin}
				{cultivars}
				{canLogCare}
				onAddPlant={() => (addingPlants = true)}
				onEdit={openEdit}
				onMove={openMove}
				onHarvest={harvest}
				onDiscard={discard}
				onLogCare={(pid) => openLogCare(undefined, [pid])}
			/>
		{:else if activeTab === 'plan'}
			<PlanTab {grow} {isAdmin} {analytics} {careDefs} onAdvance={advanceStage} onCareSettings={() => (careSettingsOpen = true)} />
		{:else if activeTab === 'analytics'}
			<AnalyticsTab {grow} {analytics} {photos} />
		{:else if activeTab === 'timeline'}
			<TimelineTab {grow} {care} {photos} {activity} {analytics} />
		{/if}
	</div>

	{#if isAdmin}
		<GrowFormModal bind:open={editing} {grow} {presets} onSaved={reload} />
		<LogCareModal
			bind:open={careOpen}
			{grow}
			actions={careActions}
			{recipes}
			preselectedPlantIds={carePreselect}
			initialActionKey={careInitialAction}
			onLogged={onCareLogged}
		/>
		<CareSettingsModal bind:open={careSettingsOpen} growId={grow.id} actions={careDefs} onSaved={(a) => (careDefs = a)} />

		<Dialog bind:open={moveOpen} title="Change location" description="Move this plant to another environment. Its placement history is kept.">
			<div class="space-y-4">
				<label class="block">
					<span class="text-xs text-rig-400">Environment</span>
					<Select value={mpEnv} onValueChange={(v) => (mpEnv = v)} items={envItems} class="mt-1" />
				</label>
				<div class="flex justify-end gap-2 border-t border-rig-800 pt-4">
					<Button variant="ghost" onclick={() => (moveOpen = false)}>Cancel</Button>
					<Button onclick={saveMove} disabled={mpBusy || !mpEnv || mpEnv === movingPlant?.currentEnvironmentId}><ArrowRightLeft size={15} /> Move</Button>
				</div>
			</div>
		</Dialog>

		<Dialog bind:open={editOpen} title="Edit plant" description="Change this plant's type, label, cultivar and pot. Each plant keeps its own id and history.">
			<div class="space-y-4">
				<div class="grid gap-3 sm:grid-cols-2">
					<label class="block"><span class="text-xs text-rig-400">Type</span><Select value={epTracking} onValueChange={(v) => (epTracking = v as TrackingMode)} items={trackingItems} class="mt-1" /></label>
					{#if epTracking === 'group'}<label class="block"><span class="text-xs text-rig-400">Plants in group</span><input type="number" min="1" bind:value={epQuantity} class="{field} mt-1" /></label>{/if}
				</div>
				<div class="grid gap-3 sm:grid-cols-2">
					<label class="block"><span class="text-xs text-rig-400">Label <span class="text-rig-600">(optional)</span></span><input bind:value={epLabel} placeholder="Plant" class="{field} mt-1" /></label>
					<label class="block"><span class="text-xs text-rig-400">Cultivar <span class="text-rig-600">(optional)</span></span><Select value={epCultivar} onValueChange={(v) => (epCultivar = v)} items={cultivarItems(epCultivar)} class="mt-1" /></label>
				</div>
				<div class="grid gap-3 sm:grid-cols-3">
					<label class="block"><span class="text-xs text-rig-400">Pot size <span class="text-rig-600">(optional)</span></span><input type="number" min="0" step="any" bind:value={epPotSize} placeholder="e.g. 11" class="{field} mt-1" /></label>
					<label class="block"><span class="text-xs text-rig-400">Unit</span><Select value={epPotUnit} onValueChange={(v) => (epPotUnit = v as PotUnit)} items={potUnitItems} class="mt-1" /></label>
					<label class="block"><span class="text-xs text-rig-400">Pot type</span><Select value={epPotType} onValueChange={(v) => (epPotType = v)} items={potTypeItems} class="mt-1" /></label>
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
					<label class="block"><span class="text-xs text-rig-400">Type</span><Select value={apTracking} onValueChange={(v) => (apTracking = v as TrackingMode)} items={trackingItems} class="mt-1" /></label>
					{#if apTracking === 'group'}<label class="block"><span class="text-xs text-rig-400">Plants in group</span><input type="number" min="1" bind:value={apQuantity} class="{field} mt-1" /></label>{/if}
				</div>
				<div class="grid gap-3 sm:grid-cols-2">
					<label class="block"><span class="text-xs text-rig-400">Label <span class="text-rig-600">(optional)</span></span><input bind:value={apLabel} placeholder={apTracking === 'group' ? 'Group' : 'Plant'} class="{field} mt-1" /></label>
					<label class="block"><span class="text-xs text-rig-400">Cultivar <span class="text-rig-600">(optional)</span></span><Select value={apCultivar} onValueChange={(v) => (apCultivar = v)} items={cultivarItems(apCultivar)} class="mt-1" /></label>
				</div>
				<label class="block"><span class="text-xs text-rig-400">Place in</span><Select value={apEnv} onValueChange={(v) => (apEnv = v)} items={envItems} class="mt-1" /></label>
				<div class="grid gap-3 sm:grid-cols-3">
					<label class="block"><span class="text-xs text-rig-400">Pot size <span class="text-rig-600">(optional)</span></span><input type="number" min="0" step="any" bind:value={apPotSize} placeholder="e.g. 11" class="{field} mt-1" /></label>
					<label class="block"><span class="text-xs text-rig-400">Unit</span><Select value={apPotUnit} onValueChange={(v) => (apPotUnit = v as PotUnit)} items={potUnitItems} class="mt-1" /></label>
					<label class="block"><span class="text-xs text-rig-400">Pot type</span><Select value={apPotType} onValueChange={(v) => (apPotType = v)} items={potTypeItems} class="mt-1" /></label>
				</div>
				<div class="flex justify-end gap-2 border-t border-rig-800 pt-4">
					<Button variant="ghost" onclick={() => (addingPlants = false)}>Cancel</Button>
					<Button onclick={addPlant} disabled={apBusy || (apTracking === 'group' && apQuantity < 1)}><Sprout size={15} /> Add plant</Button>
				</div>
			</div>
		</Dialog>
	{/if}
{/if}
