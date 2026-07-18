<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { errMsg } from '$lib/errors';
	import type { Binding, BindingKind, CatalogProduct, DiscoveredEntity, EnvironmentView } from '$lib/types';
	import { groupDevices, bindingMeta, bindingStatus, type DeviceGroup } from '$lib/equipment';
	import {
		getBindings,
		getCatalog,
		getDiscovery,
		deleteBinding,
		updateBinding
	} from '$lib/api';
	import { Button } from '$lib/components/ui';
	import KindIcon from '$lib/components/KindIcon.svelte';
	import DeviceModal from '$lib/components/DeviceModal.svelte';
	import EquipmentDetailModal from '$lib/components/env/EquipmentDetailModal.svelte';
	import CameraStreamStats from '$lib/components/CameraStreamStats.svelte';
	import Plus from '@lucide/svelte/icons/plus';
	import Star from '@lucide/svelte/icons/star';

	interface Props {
		env: EnvironmentView;
		canWrite: boolean;
		isAdmin: boolean;
	}
	let { env, canWrite, isAdmin }: Props = $props();

	let bindings = $state<Binding[]>([]);
	let catalog = $state<CatalogProduct[]>([]);
	let discovered = $state<DiscoveredEntity[]>([]);

	let notice = $state<{ kind: 'ok' | 'err'; text: string } | null>(null);
	function flash(kind: 'ok' | 'err', text: string) {
		notice = { kind, text };
		setTimeout(() => (notice = null), 2500);
	}

	async function reload() {
		try {
			[bindings, catalog, discovered] = await Promise.all([getBindings(), getCatalog(), getDiscovery()]);
		} catch (e) {
			flash('err', errMsg(e, 'Failed to reach Grow Core'));
		}
	}
	onMount(reload);

	const myBindings = $derived(bindings.filter((b) => b.environmentId === env.id));
	const devices = $derived(groupDevices(myBindings));
	const sections = $derived([
		{ label: 'Controllers', kind: 'controller' as BindingKind, devices: devices.filter((d) => d.bindings.some((b) => b.kind === 'controller')) },
		{ label: 'Lights', kind: 'light' as BindingKind, devices: devices.filter((d) => d.bindings.some((b) => b.kind === 'light')) },
		{ label: 'Fans & Airflow', kind: 'fan' as BindingKind, devices: devices.filter((d) => d.bindings.some((b) => b.kind === 'fan')) },
		{ label: 'Sensors', kind: 'sensor' as BindingKind, devices: devices.filter((d) => d.bindings.every((b) => b.kind === 'sensor')) },
		{ label: 'Cameras', kind: 'camera' as BindingKind, devices: devices.filter((d) => d.bindings.some((b) => b.kind === 'camera')) },
		{ label: 'Irrigation', kind: 'irrigation' as BindingKind, devices: devices.filter((d) => d.bindings.some((b) => b.kind === 'irrigation')) },
		{ label: 'Power & Switching', kind: 'power' as BindingKind, devices: devices.filter((d) => d.bindings.some((b) => b.kind === 'power')) }
	]);
	const usedEntities = $derived(new Set(bindings.map((b) => b.entity)));
	// Catalog product per id, so a device box can show its real product image.
	const productById = $derived(new Map(catalog.map((p) => [p.id, p])));
	const deviceProduct = (device: DeviceGroup) => productById.get(device.bindings[0]?.productId ?? '');

	// Only categories that actually have devices — drives the section list and the
	// jump-link nav at the top.
	const visibleSections = $derived(sections.filter((s) => s.devices.length));
	const sectionId = (label: string) => 'eq-' + label.toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/(^-|-$)/g, '');
	function jump(label: string) {
		document.getElementById(sectionId(label))?.scrollIntoView({ behavior: 'smooth', block: 'start' });
	}

	// --- detail modal ---
	let detailOpen = $state(false);
	let detailDeviceId = $state<string | null>(null);
	let detailCapabilityId = $state<string | null>(null);
	const detailDevice = $derived(devices.find((d) => d.id === detailDeviceId) ?? null);
	function openDetail(device: DeviceGroup, capabilityId?: string) {
		detailDeviceId = device.id;
		detailCapabilityId = capabilityId ?? device.bindings[0]?.id ?? null;
		detailOpen = true;
	}

	// --- add / edit modal ---
	let modalOpen = $state(false);
	let editTarget = $state<Binding | null>(null);
	let addCategory = $state<BindingKind>('sensor');
	function openAdd(category: BindingKind = 'sensor') {
		addCategory = category;
		editTarget = null;
		modalOpen = true;
	}
	function openEdit(b: Binding) {
		detailOpen = false;
		editTarget = b;
		modalOpen = true;
	}
	function installProduct(product: CatalogProduct) {
		modalOpen = false;
		goto(`/env/${env.id}/devices/install/${product.id}`);
	}

	async function removeDevice(device: DeviceGroup) {
		if (!confirm(`Remove "${device.name}" and all its capabilities?`)) return;
		try {
			for (const b of device.bindings) await deleteBinding(b.id);
			detailOpen = false;
			flash('ok', 'Device removed');
			reload();
		} catch (e) {
			flash('err', errMsg(e, 'Delete failed'));
		}
	}

	// Promote a light to primary; the backend clears the flag on the others.
	async function makePrimary(b: Binding) {
		try {
			await updateBinding(b.id, {
				deviceId: b.deviceId,
				deviceName: b.deviceName,
				productId: b.productId,
				powerControllerId: b.powerControllerId,
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
			flash('err', errMsg(e, 'Update failed'));
		}
	}
</script>

<div class="space-y-6">
	<div class="flex items-center justify-between">
		<div>
			<h2 class="text-sm font-semibold uppercase tracking-wide text-rig-400">Equipment</h2>
			<p class="text-xs text-rig-500">{devices.length} physical device{devices.length === 1 ? '' : 's'} · {myBindings.length} capabilities</p>
		</div>
		{#if isAdmin}
			<Button onclick={() => openAdd()}><Plus size={16} /> Add device</Button>
		{/if}
	</div>

	{#if notice}
		<div class="rounded-lg px-4 py-2 text-sm {notice.kind === 'ok' ? 'bg-leaf/15 text-leaf' : 'bg-danger/15 text-danger'}">
			{notice.text}
		</div>
	{/if}

	{#if myBindings.length === 0}
		<div class="rounded-xl border border-dashed border-rig-800 p-10 text-center">
			<p class="mb-4 text-sm text-rig-400">No equipment yet.</p>
			{#if isAdmin}<Button onclick={() => openAdd()}><Plus size={16} /> Add your first device</Button>{/if}
		</div>
	{:else}
		<!-- Fast links: jump to a category -->
		{#if visibleSections.length > 1}
			<nav class="-mx-1 flex flex-wrap gap-2 border-y border-rig-800/60 px-1 py-3">
				{#each visibleSections as section (section.label)}
					<button
						type="button"
						onclick={() => jump(section.label)}
						class="inline-flex items-center gap-1.5 rounded-full border border-rig-800 bg-rig-950/40 px-3 py-1.5 text-xs font-medium text-rig-300 transition-colors hover:border-rig-600 hover:text-rig-100"
					>
						<KindIcon kind={section.kind} size={14} />
						{section.label}
						<span class="text-rig-500">{section.devices.length}</span>
					</button>
				{/each}
			</nav>
		{/if}

		<div class="space-y-12">
			{#each visibleSections as section (section.label)}
				<section id={sectionId(section.label)} class="scroll-mt-24">
					<div class="mb-4 flex items-center gap-3 border-b border-rig-800/60 pb-3">
						<div class="flex h-11 w-11 shrink-0 items-center justify-center rounded-xl border border-rig-800 bg-rig-950/60">
							<KindIcon kind={section.kind} size={26} class="text-rig-300" />
						</div>
						<div>
							<h3 class="text-lg font-semibold text-rig-100">{section.label}</h3>
							<p class="text-xs text-rig-500">{section.devices.length} device{section.devices.length === 1 ? '' : 's'}</p>
						</div>
					</div>
					<div class="grid gap-3 sm:grid-cols-2">
						{#each section.devices as device (device.id)}
							{@const product = deviceProduct(device)}
							<div class="overflow-hidden rounded-xl border border-rig-800 bg-rig-900/40 transition-colors hover:border-rig-600">
								<button
									type="button"
									onclick={() => openDetail(device)}
									class="flex w-full items-center gap-3 border-b border-rig-800/60 px-3 py-3 text-left focus-visible:outline-none"
								>
									{#if product?.image}
										<img src={product.image} alt="" class="h-12 w-12 shrink-0 rounded-lg border border-rig-800 bg-rig-950 object-contain p-0.5" />
									{/if}
									<div class="min-w-0 flex-1">
										<div class="truncate text-sm font-semibold">{device.name}</div>
										<div class="text-xs text-rig-500">{device.bindings.length} {device.bindings.length === 1 ? 'capability' : 'capabilities'}</div>
									</div>
								</button>
								{#each device.bindings as b (b.id)}
									{@const st = bindingStatus(b, env)}
									<button
										type="button"
										onclick={() => openDetail(device, b.id)}
										class="flex w-full items-center gap-3 border-b border-rig-800/40 px-3 py-2.5 text-left transition-colors last:border-0 hover:bg-rig-800/40 focus-visible:outline-none"
									>
										<div class="min-w-0 flex-1">
											<div class="text-xs font-medium capitalize text-rig-300">{bindingMeta(b)}</div>
										</div>
										{#if b.kind === 'camera' && b.cameraType === 'rtsp'}
											<CameraStreamStats cameraId={b.id} class="text-xs font-semibold text-rig-300" />
										{:else}
											<span class="text-sm font-semibold tabular-nums text-rig-100">{st.value || '—'}</span>
										{/if}
										{#if b.kind === 'light' && b.primary}
											<Star size={13} fill="currentColor" class="text-warn" />
										{/if}
										{#if b.kind !== 'camera' || b.cameraType !== 'rtsp'}
											<span class="h-2 w-2 shrink-0 rounded-full {st.online ? 'bg-leaf' : st.online === false ? 'bg-danger' : 'bg-rig-700'}" title={st.online ? 'Online' : st.online === false ? 'Offline' : 'Unknown'}></span>
										{/if}
									</button>
								{/each}
							</div>
						{/each}
					</div>
				</section>
			{/each}
		</div>
	{/if}
</div>

<EquipmentDetailModal
	bind:open={detailOpen}
	device={detailDevice}
	{env}
	envId={env.id}
	{canWrite}
	image={detailDevice ? deviceProduct(detailDevice)?.image : undefined}
	initialCapabilityId={detailCapabilityId}
	onEditBinding={openEdit}
	onRemoveDevice={removeDevice}
	onMakePrimary={makePrimary}
/>

<DeviceModal
	bind:open={modalOpen}
	environmentId={env.id}
	{catalog}
	{discovered}
	{usedEntities}
	bindings={myBindings}
	binding={editTarget}
	onSaved={reload}
	{flash}
	onInstall={installProduct}
	initialCategory={addCategory}
/>
