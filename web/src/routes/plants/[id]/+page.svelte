<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/state';
	import { auth } from '$lib/auth.svelte';
	import { getPlant, getEnvironments, movePlant, updatePlant, harvestPlant, removePlant, getCultivars, cultivarImageURL } from '$lib/api';
	import type { Environment, PlantView, Cultivar } from '$lib/types';
	import { titleCase, daysSince } from '$lib/format';
	import { fmtDate } from '$lib/datetime';
	import { Button, Dialog, Select } from '$lib/components/ui';
	import ArrowLeft from '@lucide/svelte/icons/arrow-left';
	import MapPin from '@lucide/svelte/icons/map-pin';
	import Pencil from '@lucide/svelte/icons/pencil';
	import Sprout from '@lucide/svelte/icons/sprout';

	const id = $derived(page.params.id);
	const isAdmin = $derived(auth.isAdmin);

	let plant = $state<PlantView | null>(null);
	let environments = $state<Environment[]>([]);
	let cultivars = $state<Cultivar[]>([]);
	let err = $state('');
	let loading = $state(true);

	// The plant's cultivar record (for its image), resolved by name.
	const cultivar = $derived(plant ? cultivars.find((c) => c.name === plant!.cultivar) : undefined);

	async function reload() {
		if (!id) return;
		try {
			plant = await getPlant(id);
			err = '';
		} catch (e) {
			err = e instanceof Error ? e.message : 'Failed to load plant';
		} finally {
			loading = false;
		}
	}
	onMount(() => {
		reload();
		getEnvironments().then((e) => (environments = e)).catch(() => {});
		getCultivars().then((c) => (cultivars = c)).catch(() => {});
	});

	const envItems = $derived(environments.map((e) => ({ value: e.id, label: e.name })));

	async function move(envId: string) {
		if (!plant || !envId || envId === plant.currentEnvironmentId) return;
		try {
			await movePlant(plant.id, envId);
			await reload();
		} catch (e) {
			err = e instanceof Error ? e.message : 'Failed';
		}
	}
	let editOpen = $state(false);
	let epLabel = $state('');
	let epCultivar = $state('');
	let epQuantity = $state(1);
	let epBusy = $state(false);
	function openEdit() {
		if (!plant) return;
		epLabel = plant.label;
		epCultivar = plant.cultivar;
		epQuantity = plant.quantity;
		editOpen = true;
	}
	async function saveEdit() {
		if (!plant) return;
		epBusy = true;
		try {
			await updatePlant(plant.id, {
				label: epLabel.trim(),
				cultivar: epCultivar.trim(),
				quantity: plant.tracking === 'group' ? epQuantity : undefined
			});
			editOpen = false;
			await reload();
		} catch (e) {
			err = e instanceof Error ? e.message : 'Failed';
		} finally {
			epBusy = false;
		}
	}

	async function harvest() {
		if (!plant) return;
		try {
			await harvestPlant(plant.id);
			await reload();
		} catch (e) {
			err = e instanceof Error ? e.message : 'Failed';
		}
	}
	async function discard() {
		if (!plant || !confirm('Remove this plant?')) return;
		try {
			await removePlant(plant.id);
			await reload();
		} catch (e) {
			err = e instanceof Error ? e.message : 'Failed';
		}
	}

	const statusTone = (s: string) =>
		s === 'active' ? 'text-leaf' : s === 'harvested' ? 'text-warn' : 'text-rig-500';
	const field =
		'w-full rounded-md border border-rig-700 bg-rig-950 px-3 py-1.5 text-sm focus:border-rig-500 focus:outline-none';
</script>

{#if plant}
	<a href="/grows/{plant.growId}" class="mb-4 inline-flex items-center gap-1 text-sm text-rig-400 hover:text-rig-100">
		<ArrowLeft size={15} /> {plant.growName || 'Grow'}
	</a>
{:else}
	<a href="/grows" class="mb-4 inline-flex items-center gap-1 text-sm text-rig-400 hover:text-rig-100">
		<ArrowLeft size={15} /> Grows
	</a>
{/if}

{#if loading}
	<p class="text-rig-400">Loading…</p>
{:else if !plant}
	<p class="text-rig-400">Plant not found.</p>
{:else}
	<div class="space-y-6">
		{#if err}<p class="rounded-md border border-danger/40 bg-danger/10 px-3 py-2 text-sm text-danger">{err}</p>{/if}

		<div class="flex flex-wrap items-start justify-between gap-3">
			<div class="flex items-center gap-4">
				<div class="h-14 w-14 shrink-0 overflow-hidden rounded-full border border-rig-700 bg-rig-950">
					{#if cultivar?.imageType}
						<img src={cultivarImageURL(cultivar.id)} alt={plant.cultivar} class="h-full w-full object-cover" />
					{:else}
						<div class="flex h-full w-full items-center justify-center text-rig-600"><Sprout size={24} /></div>
					{/if}
				</div>
				<div>
				<div class="flex items-center gap-3">
					<h1 class="text-2xl font-semibold">
						{plant.cultivar || plant.label || 'Plant'}{#if plant.tracking === 'group'}<span class="text-rig-500">&nbsp;×{plant.quantity}</span>{/if}
					</h1>
					<span class="rounded-full bg-rig-800 px-2 py-0.5 text-xs capitalize {statusTone(plant.status)}">{plant.status}</span>
				</div>
				<p class="mt-1 text-sm text-rig-400">
					{plant.tracking === 'group' ? `Group of ${plant.quantity}` : 'Individually tracked'}
					· in <a href="/grows/{plant.growId}" class="text-leaf hover:underline">{plant.growName}</a>
					· {daysSince(plant.createdAt)}d old
				</p>
				</div>
			</div>
			{#if isAdmin}
				<div class="flex flex-wrap items-center gap-2">
					<Button variant="ghost" onclick={openEdit}><Pencil size={15} /> Edit</Button>
					{#if plant.status === 'active'}
						<label class="flex items-center gap-2 text-sm">
							<span class="text-rig-400">Move to</span>
							<Select value={plant.currentEnvironmentId} onValueChange={move} items={envItems} />
						</label>
						<Button variant="secondary" onclick={harvest}>Harvest</Button>
						<Button variant="ghost" onclick={discard}>Remove</Button>
					{/if}
				</div>
			{/if}
		</div>

		<section class="rounded-xl border border-rig-800 bg-rig-900/40 p-4">
			<div class="flex items-center gap-2 text-sm">
				<MapPin size={15} class="text-rig-500" />
				<span class="text-rig-400">Currently in</span>
				<span class="font-medium">{plant.currentEnvironmentName || 'nowhere'}</span>
			</div>
		</section>

		<section>
			<h2 class="mb-3 text-sm font-semibold uppercase tracking-wide text-rig-400">Placement history</h2>
			{#if plant.placements.length === 0}
				<p class="text-sm text-rig-500">No placements recorded.</p>
			{:else}
				<ol class="space-y-2">
					{#each plant.placements as p (p.id)}
						<li class="flex items-center justify-between rounded-lg border border-rig-800 bg-rig-950/40 px-4 py-2 text-sm">
							<span class="font-medium">{p.environmentName || p.environmentId}</span>
							<span class="text-rig-400">
								{fmtDate(p.startedAt)} →
								{#if p.endedAt}{fmtDate(p.endedAt)}{:else}<span class="text-leaf">current</span>{/if}
							</span>
						</li>
					{/each}
				</ol>
			{/if}
		</section>
	</div>

	{#if isAdmin}
		<Dialog bind:open={editOpen} title="Edit plant" description="Update this plant's label and cultivar.">
			<div class="space-y-4">
				<div class="grid gap-3 sm:grid-cols-2">
					<label class="block">
						<span class="text-xs text-rig-400">Label</span>
						<input bind:value={epLabel} placeholder="Plant" class="{field} mt-1" />
					</label>
					<label class="block">
						<span class="text-xs text-rig-400">Cultivar</span>
						<input bind:value={epCultivar} placeholder="e.g. Genovese" class="{field} mt-1" />
					</label>
				</div>
				{#if plant.tracking === 'group'}
					<label class="block">
						<span class="text-xs text-rig-400">Plants in group</span>
						<input type="number" min="1" bind:value={epQuantity} class="{field} mt-1" />
					</label>
				{/if}
				<div class="flex justify-end gap-2 border-t border-rig-800 pt-4">
					<Button variant="ghost" onclick={() => (editOpen = false)}>Cancel</Button>
					<Button onclick={saveEdit} disabled={epBusy}>Save</Button>
				</div>
			</div>
		</Dialog>
	{/if}
{/if}
