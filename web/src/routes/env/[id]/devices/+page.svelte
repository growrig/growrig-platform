<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/state';
	import { live } from '$lib/live.svelte';
	import {
		getEnvironments,
		getBindings,
		getCatalog,
		getDiscovery,
		deleteBinding,
		updateBinding
	} from '$lib/api';
	import type { Binding, CatalogProduct, DiscoveredEntity, Environment } from '$lib/types';
	import { measurementUnit } from '$lib/format';
	import KindIcon from '$lib/components/KindIcon.svelte';
	import DeviceModal from '$lib/components/DeviceModal.svelte';
	import { Button } from '$lib/components/ui';
	import ArrowLeft from '@lucide/svelte/icons/arrow-left';
	import Plus from '@lucide/svelte/icons/plus';
	import Pencil from '@lucide/svelte/icons/pencil';
	import Trash2 from '@lucide/svelte/icons/trash-2';
	import Star from '@lucide/svelte/icons/star';

	const id = $derived(page.params.id);

	let environments = $state<Environment[]>([]);
	let bindings = $state<Binding[]>([]);
	let catalog = $state<CatalogProduct[]>([]);
	let discovered = $state<DiscoveredEntity[]>([]);
	let loading = $state(true);
	let error = $state<string | null>(null);

	let notice = $state<{ kind: 'ok' | 'err'; text: string } | null>(null);
	function flash(kind: 'ok' | 'err', text: string) {
		notice = { kind, text };
		setTimeout(() => (notice = null), 2500);
	}

	async function reload() {
		try {
			[environments, bindings, catalog, discovered] = await Promise.all([
				getEnvironments(),
				getBindings(),
				getCatalog(),
				getDiscovery()
			]);
			error = null;
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to reach Grow Core';
		} finally {
			loading = false;
		}
	}
	onMount(reload);

	const env = $derived(environments.find((e) => e.id === id));
	const myBindings = $derived(bindings.filter((b) => b.environmentId === id));
	const usedEntities = $derived(new Set(bindings.map((b) => b.entity)));

	// Live view for this environment, keyed by binding id.
	const liveEnv = $derived(live.snapshot?.environments?.find((e) => e.id === id));
	const sensorById = $derived(new Map((liveEnv?.sensors ?? []).map((s) => [s.id, s])));
	const controlById = $derived(new Map((liveEnv?.controls ?? []).map((c) => [c.id, c])));

	// Latest value/state + reachability for one device, from the live snapshot.
	function status(b: Binding): { value: string; online: boolean | null } {
		if (b.kind === 'sensor') {
			const s = sensorById.get(b.id);
			if (!s) return { value: '—', online: null };
			const unit = b.measurement ? measurementUnit[b.measurement] : '';
			return { value: s.ok ? `${s.value}${unit}` : '—', online: s.ok };
		}
		if (b.kind === 'fan') {
			const c = controlById.get(b.id);
			if (!c) return { value: '—', online: liveEnv ? env?.kind === 'tent' : null };
			return { value: `${c.desiredSpeed}%${c.rpm ? ` · ${c.rpm} rpm` : ''}`, online: onlineFromHealth() };
		}
		if (b.kind === 'light') {
			const c = controlById.get(b.id);
			return { value: c ? (c.on ? 'On' : 'Off') : '—', online: onlineFromHealth() };
		}
		return { value: '', online: onlineFromHealth() }; // camera
	}

	function onlineFromHealth(): boolean | null {
		if (!liveEnv) return null;
		return liveEnv.health === 'online';
	}

	function meta(b: Binding): string {
		if (b.kind === 'sensor') return b.measurement ?? 'sensor';
		if (b.kind === 'fan') return b.role ?? 'fan';
		if (b.kind === 'light') return b.wattage ? `${b.wattage} W` : 'light';
		return b.kind;
	}

	// Promote a light to primary; the backend clears the flag on the others.
	async function makePrimary(b: Binding) {
		try {
			await updateBinding(b.id, {
				environmentId: b.environmentId,
				kind: b.kind,
				name: b.name,
				entity: b.entity,
				wattage: b.wattage,
				primary: true
			});
			flash('ok', `${b.name} is now the primary light`);
			reload();
		} catch (e) {
			flash('err', e instanceof Error ? e.message : 'Update failed');
		}
	}

	// --- add / edit modal ---
	let modalOpen = $state(false);
	let editTarget = $state<Binding | null>(null);

	function openAdd() {
		editTarget = null;
		modalOpen = true;
	}
	function openEdit(b: Binding) {
		editTarget = b;
		modalOpen = true;
	}

	async function remove(b: Binding) {
		if (!confirm(`Remove "${b.name}"?`)) return;
		try {
			await deleteBinding(b.id);
			flash('ok', 'Device removed');
			reload();
		} catch (e) {
			flash('err', e instanceof Error ? e.message : 'Delete failed');
		}
	}
</script>

<a href="/env/{id}" class="mb-4 inline-flex items-center gap-1 text-sm text-rig-400 hover:text-rig-100">
	<ArrowLeft size={15} /> Back to {env?.name ?? 'environment'}
</a>

{#if loading}
	<p class="text-rig-400">Loading…</p>
{:else if error}
	<p class="text-danger">{error}</p>
{:else if !env}
	<p class="text-rig-400">Environment not found. <a href="/" class="text-leaf hover:underline">Go back</a></p>
{:else}
	<div class="space-y-5">
		<div class="flex items-center justify-between">
			<div>
				<h1 class="text-2xl font-semibold">Devices</h1>
				<p class="text-sm text-rig-400">{env.name} · {myBindings.length} device{myBindings.length === 1 ? '' : 's'}</p>
			</div>
			<Button onclick={openAdd}><Plus size={16} /> Add device</Button>
		</div>

		{#if notice}
			<div class="rounded-lg px-4 py-2 text-sm {notice.kind === 'ok' ? 'bg-leaf/15 text-leaf' : 'bg-danger/15 text-danger'}">
				{notice.text}
			</div>
		{/if}

		{#if myBindings.length === 0}
			<div class="rounded-xl border border-dashed border-rig-800 p-10 text-center">
				<p class="mb-4 text-sm text-rig-400">No devices yet.</p>
				<Button onclick={openAdd}><Plus size={16} /> Add your first device</Button>
			</div>
		{:else}
			<div class="overflow-hidden rounded-xl border border-rig-800">
				{#each myBindings as b, i (b.id)}
					{@const st = status(b)}
					<div
						class="flex items-center gap-3 bg-rig-900/40 px-4 py-3 {i > 0 ? 'border-t border-rig-800' : ''}"
					>
						<KindIcon kind={b.kind} size={20} class="shrink-0 text-rig-400" />
						<div class="min-w-0 flex-1">
							<div class="truncate text-sm font-medium">{b.name}</div>
							<div class="truncate font-mono text-xs text-rig-500">{b.entity}</div>
						</div>

						<span class="hidden rounded-full bg-rig-800 px-2 py-0.5 text-xs capitalize text-rig-300 sm:inline">
							{meta(b)}
						</span>

						<div class="w-28 text-right text-sm font-semibold tabular-nums {st.online === false ? 'text-rig-600' : 'text-rig-100'}">
							{st.value || '—'}
						</div>

						<span class="flex w-20 items-center justify-end gap-1.5 text-xs">
							{#if st.online === null}
								<span class="h-2 w-2 rounded-full bg-rig-700"></span><span class="text-rig-500">—</span>
							{:else if st.online}
								<span class="h-2 w-2 rounded-full bg-leaf"></span><span class="text-leaf">online</span>
							{:else}
								<span class="h-2 w-2 rounded-full bg-danger"></span><span class="text-danger">offline</span>
							{/if}
						</span>

						<div class="flex items-center gap-1">
							{#if b.kind === 'light'}
								{#if b.primary}
									<span class="flex items-center gap-1 rounded-md px-1.5 py-1 text-xs text-warn" title="Primary grow light">
										<Star size={15} fill="currentColor" /> Primary
									</span>
								{:else}
									<button
										onclick={() => makePrimary(b)}
										class="rounded-md p-1.5 text-rig-500 transition-colors hover:bg-rig-800 hover:text-warn"
										title="Make primary grow light"
										aria-label="Make {b.name} the primary light"
									>
										<Star size={15} />
									</button>
								{/if}
							{/if}
							<button
								onclick={() => openEdit(b)}
								class="rounded-md p-1.5 text-rig-400 transition-colors hover:bg-rig-800 hover:text-rig-100"
								title="Edit"
								aria-label="Edit {b.name}"
							>
								<Pencil size={15} />
							</button>
							<button
								onclick={() => remove(b)}
								class="rounded-md p-1.5 text-rig-400 transition-colors hover:bg-rig-800 hover:text-danger"
								title="Remove"
								aria-label="Remove {b.name}"
							>
								<Trash2 size={15} />
							</button>
						</div>
					</div>
				{/each}
			</div>
		{/if}
	</div>

	<DeviceModal
		bind:open={modalOpen}
		environmentId={id!}
		{catalog}
		{discovered}
		{usedEntities}
		binding={editTarget}
		onSaved={reload}
		{flash}
	/>
{/if}
