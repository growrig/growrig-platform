<script lang="ts">
	import type { Binding, EnvironmentView, Measurement } from '$lib/types';
	import type { DeviceGroup } from '$lib/equipment';
	import type { MetricDescriptor } from '$lib/components/MetricGraph.svelte';
	import { bindingMeta, bindingStatus } from '$lib/equipment';
	import { cameraProxyURL, setSwitch } from '$lib/api';
	import { measurementUnit, measurementLabel } from '$lib/format';
	import { Dialog, Switch, Button } from '$lib/components/ui';
	import KindIcon from '$lib/components/KindIcon.svelte';
	import MetricGraph from '$lib/components/MetricGraph.svelte';
	import CameraPreview from '$lib/components/CameraPreview.svelte';
	import Star from '@lucide/svelte/icons/star';
	import Pencil from '@lucide/svelte/icons/pencil';
	import Trash2 from '@lucide/svelte/icons/trash-2';

	interface Props {
		open?: boolean;
		device: DeviceGroup | null;
		env: EnvironmentView | undefined;
		envId: string;
		canWrite: boolean;
		/** Catalog product image for this device, if it was installed from the catalog. */
		image?: string;
		/** Capability (binding) id to open on. */
		initialCapabilityId?: string | null;
		onEditBinding: (b: Binding) => void;
		onRemoveDevice: (device: DeviceGroup) => void;
		onMakePrimary: (b: Binding) => void;
	}
	let {
		open = $bindable(false),
		device,
		env,
		envId,
		canWrite,
		image,
		initialCapabilityId = null,
		onEditBinding,
		onRemoveDevice,
		onMakePrimary
	}: Props = $props();

	const bindings = $derived(device?.bindings ?? []);

	// Selected capability, seeded from the clicked row each time the modal opens.
	let selectedId = $state<string | null>(null);
	let wasOpen = false;
	$effect(() => {
		if (open && !wasOpen) {
			selectedId = initialCapabilityId ?? bindings[0]?.id ?? null;
		}
		wasOpen = open;
	});
	const selected = $derived(bindings.find((b) => b.id === selectedId) ?? bindings[0]);

	// Bindings that carry an on/off switch (rendered as controls at the top).
	const switchable = $derived(bindings.filter((b) => b.kind === 'light' || b.kind === 'power'));

	function controlOn(b: Binding): boolean {
		return env?.controls.find((c) => c.id === b.id)?.on ?? false;
	}
	async function toggle(b: Binding, on: boolean) {
		try {
			await setSwitch(b.id, on);
		} catch {
			/* reconciles via live feed */
		}
	}

	// Graph descriptor + unit for a capability, or null when it has no history.
	function graphFor(b: Binding): { descriptor: MetricDescriptor; unit: string } | null {
		if (b.kind === 'sensor' && b.measurement)
			return { descriptor: { kind: 'sensor', measurement: b.measurement as Measurement }, unit: measurementUnit[b.measurement] };
		if (b.kind === 'fan') return { descriptor: { kind: 'device', bindingId: b.id, metric: 'rpm' }, unit: 'rpm' };
		if (b.kind === 'light') return { descriptor: { kind: 'device', bindingId: b.id, metric: 'power' }, unit: 'W' };
		return null;
	}

	const camera = $derived(selected?.kind === 'camera' ? env?.cameras.find((c) => c.id === selected.id) : undefined);

	function capLabel(b: Binding): string {
		if (b.kind === 'sensor' && b.measurement) return measurementLabel[b.measurement];
		return bindingMeta(b);
	}
</script>

<Dialog bind:open title={device?.name ?? 'Device'} size="2xl">
	{#if device}
		<div class="space-y-4">
			<!-- Device identity -->
			<div class="flex items-center gap-3 text-sm text-rig-400">
				{#if image}
					<img src={image} alt="" class="h-14 w-14 shrink-0 rounded-lg border border-rig-800 bg-rig-950 object-contain p-1" />
				{:else}
					<KindIcon kind={bindings[0]?.kind ?? 'sensor'} size={18} />
				{/if}
				<span>{bindings.length} {bindings.length === 1 ? 'capability' : 'capabilities'}</span>
			</div>

			<!-- Controls (switches + primary) -->
			{#if switchable.length}
				<div class="space-y-2">
					{#each switchable as b (b.id)}
						<div class="flex items-center justify-between gap-3 rounded-lg border border-rig-800 bg-rig-950/40 px-3 py-2.5">
							<div class="flex min-w-0 items-center gap-2">
								<span class="truncate text-sm font-medium">{b.name}</span>
								{#if b.kind === 'light'}
									{#if b.primary}
										<span class="inline-flex items-center gap-1 rounded-full bg-rig-800 px-2 py-0.5 text-[10px] font-medium uppercase tracking-wide text-rig-300">
											<Star size={10} fill="currentColor" class="text-warn" /> Primary
										</span>
									{:else}
										<button
											onclick={() => onMakePrimary(b)}
											class="rounded-md p-1 text-rig-500 transition-colors hover:bg-rig-800 hover:text-warn"
											title="Make primary grow light"
											aria-label="Make {b.name} the primary light"
										>
											<Star size={14} />
										</button>
									{/if}
								{/if}
							</div>
							<div class="flex items-center gap-2">
								<span class="text-xs font-medium tabular-nums {controlOn(b) ? 'text-leaf' : 'text-rig-400'}">{controlOn(b) ? 'On' : 'Off'}</span>
								{#if canWrite}<Switch checked={controlOn(b)} onCheckedChange={(v) => toggle(b, v)} />{/if}
							</div>
						</div>
					{/each}
				</div>
			{/if}

			<!-- Capability switcher -->
			{#if bindings.length > 1}
				<div class="flex flex-wrap gap-1">
					{#each bindings as b (b.id)}
						<button
							type="button"
							onclick={() => (selectedId = b.id)}
							class="rounded-md px-3 py-1.5 text-xs font-medium capitalize transition-colors {selected?.id === b.id
								? 'bg-rig-700 text-rig-50'
								: 'text-rig-400 hover:bg-rig-800 hover:text-rig-100'}"
						>
							{capLabel(b)}
						</button>
					{/each}
				</div>
			{/if}

			<!-- Selected capability body -->
			{#if selected}
				{@const g = graphFor(selected)}
				{#if selected.kind === 'camera'}
					{#if camera}
						<CameraPreview
							url={camera.cameraType === 'rtsp' || camera.entity || !camera.streamUrl ? cameraProxyURL(camera.id) : camera.streamUrl}
							liveUrl={camera.cameraType === 'rtsp' || (!camera.streamUrl && !camera.entity) ? cameraProxyURL(camera.id, true) : ''}
							type={camera.cameraType === 'rtsp' ? 'snapshot' : camera.streamUrl ? camera.cameraType : 'snapshot'}
							refreshSeconds={camera.cameraType === 'rtsp' ? camera.cameraCaptureInterval ?? 60 : 2}
							emptyLabel="Connecting to camera…"
							errorLabel="Connecting to camera…"
						/>
					{:else}
						<p class="rounded-lg border border-dashed border-rig-800 px-3 py-6 text-center text-sm text-rig-500">Camera offline.</p>
					{/if}
				{:else if g}
					<MetricGraph active={open} {envId} unit={g.unit} descriptor={g.descriptor} sensors={env?.sensors ?? []} controls={env?.controls ?? []} />
				{:else}
					{@const st = bindingStatus(selected, env)}
					<div class="flex items-center justify-between rounded-lg border border-rig-800 bg-rig-950/40 px-3 py-3 text-sm">
						<span class="capitalize text-rig-300">{capLabel(selected)}</span>
						<span class="tabular-nums text-rig-100">{st.value || '—'}</span>
					</div>
				{/if}
			{/if}

			<!-- Footer actions -->
			<div class="flex items-center justify-between border-t border-rig-800 pt-3">
				<Button variant="ghost" onclick={() => onRemoveDevice(device)} class="text-danger hover:text-danger">
					<Trash2 size={15} /> Remove device
				</Button>
				{#if selected}
					<Button variant="secondary" onclick={() => onEditBinding(selected)}>
						<Pencil size={15} /> Edit {bindings.length > 1 ? capLabel(selected) : 'device'}
					</Button>
				{/if}
			</div>
		</div>
	{/if}
</Dialog>
