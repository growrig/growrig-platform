<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { getCatalog, getDiscovery, getBindings, getEnvironments, createBinding } from '$lib/api';
	import type { Binding, CatalogProduct, DiscoveredEntity, Environment } from '$lib/types';
	import CatalogDevicePicker, { type BindingDraft } from '$lib/components/CatalogDevicePicker.svelte';
	import KindIcon from '$lib/components/KindIcon.svelte';
	import ArrowLeft from '@lucide/svelte/icons/arrow-left';
	import X from '@lucide/svelte/icons/x';

	let catalog = $state<CatalogProduct[]>([]);
	let discovered = $state<DiscoveredEntity[]>([]);
	let environments = $state<Environment[]>([]);
	let bindings = $state<Binding[]>([]);
	let error = $state<string | null>(null);
	let saving = $state(false);

	// Preselect an environment via ?env=, else the first.
	let envId = $state('');
	let added = $state<BindingDraft[]>([]);

	const boundEntities = $derived(new Set(bindings.map((b) => b.entity)));
	const usedEntities = $derived(new Set([...boundEntities, ...added.map((d) => d.entity)]));

	onMount(async () => {
		try {
			[catalog, discovered, environments, bindings] = await Promise.all([
				getCatalog(),
				getDiscovery(),
				getEnvironments(),
				getBindings()
			]);
			envId = page.url.searchParams.get('env') || environments[0]?.id || '';
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to load';
		}
	});

	async function finish() {
		saving = true;
		error = null;
		try {
			for (const d of added) await createBinding({ environmentId: envId, ...d });
			await goto(`/env/${envId}`);
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to add devices';
			saving = false;
		}
	}

	const field = 'w-full rounded-md border border-rig-700 bg-rig-950 px-3 py-1.5 text-sm focus:border-rig-500 focus:outline-none';
</script>

<div class="mx-auto max-w-2xl">
	<a href="/" class="mb-4 inline-flex items-center gap-1 text-sm text-rig-400 hover:text-rig-100">
	<ArrowLeft size={15} /> Cancel
</a>
	<h1 class="mb-4 text-2xl font-semibold">Add Device</h1>

	{#if error}
		<div class="mb-4 rounded-lg bg-danger/15 px-4 py-2 text-sm text-danger">{error}</div>
	{/if}

	{#if environments.length === 0}
		<p class="text-rig-400">Create a Grow Box or Lung Room first.</p>
	{:else}
		<div class="space-y-4 rounded-xl border border-rig-800 bg-rig-900/40 p-5">
			<label class="block">
				<span class="text-sm text-rig-400">Environment</span>
				<select bind:value={envId} class="{field} mt-1">
					{#each environments as env (env.id)}
						<option value={env.id}>{env.name}</option>
					{/each}
				</select>
			</label>
			<CatalogDevicePicker {catalog} {discovered} {usedEntities} onAdd={(d) => (added = [...added, ...d])} />
			{#if added.length}
				<div class="space-y-1.5">
					{#each added as d, i (i)}
						<div class="flex items-center gap-2 rounded-md bg-rig-950/40 px-3 py-1.5 text-sm">
							<KindIcon kind={d.kind} size={16} class="shrink-0 text-rig-400" />
							<span class="flex-1">{d.name} <span class="text-xs text-rig-500">{d.entity}</span></span>
							<button onclick={() => (added = added.filter((_, j) => j !== i))} class="text-rig-500 hover:text-danger" aria-label="Remove"><X size={15} /></button>
						</div>
					{/each}
				</div>
			{/if}
		</div>

		<div class="mt-4 flex justify-end">
			<button
				onclick={finish}
				disabled={saving || added.length === 0}
				class="rounded-md bg-rig-500 px-5 py-1.5 text-sm font-medium text-rig-950 transition-colors hover:bg-rig-400 disabled:opacity-40"
			>
				{saving ? 'Adding…' : `Add ${added.length} device${added.length === 1 ? '' : 's'}`}
			</button>
		</div>
	{/if}
</div>
