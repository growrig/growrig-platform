<script lang="ts">
	import { live } from '$lib/live.svelte';
	import { resolveLocationId } from '$lib/location';
	import { formatDimensions, volumeM3 } from '$lib/format';
	import { Breadcrumb } from '$lib/components/ui';
	import type { EnvironmentView, Location } from '$lib/types';
	import Zap from '@lucide/svelte/icons/zap';
	import CircleCheck from '@lucide/svelte/icons/circle-check';

	interface Props {
		env: EnvironmentView;
		locations: Location[];
		/** Open alerts scoped to this environment; 0 → "everything looks good". */
		alertCount: number;
	}
	let { env, locations, alertCount }: Props = $props();

	const healthDot = (h: string) =>
		h === 'online' ? 'bg-leaf' : h === 'stale' ? 'bg-warn' : 'bg-danger';
	const healthLabel = (h: string) =>
		h === 'online' ? 'Online' : h === 'stale' ? 'Stale' : 'Offline';

	// Breadcrumb: Location / Room (current env is the page title, not repeated).
	// A tent shows its air-source room; a room shows just its location.
	const locationName = $derived.by(() => {
		const locId = resolveLocationId(env, live.snapshot?.environments ?? []);
		return locations.find((l) => l.id === locId)?.name ?? '';
	});
	const crumbs = $derived(
		[
			{ label: 'All environments', href: '/' },
			locationName ? { label: locationName, href: '/' } : undefined,
			env.kind === 'tent' && env.airSource
				? { label: env.airSource.name, href: `/env/${env.airSource.id}` }
				: undefined
		].filter((c): c is { label: string; href: string } => !!c)
	);

	const dims = $derived(formatDimensions(env.widthCm, env.depthCm, env.heightCm));
	const vol = $derived(volumeM3(env.widthCm, env.depthCm, env.heightCm));
	const meta = $derived(
		[env.model, dims, vol ? `${vol.toFixed(2)} m³` : ''].filter(Boolean).join(' · ')
	);
</script>

<header class="space-y-1.5">
	<Breadcrumb items={crumbs} />

	<div class="flex items-start justify-between gap-3">
		<div class="min-w-0">
			<div class="flex flex-wrap items-center gap-2">
				<h1 class="text-2xl font-semibold">{env.name}</h1>
				<span class="rounded-full bg-rig-800 px-2 py-0.5 text-[10px] uppercase tracking-wide text-rig-400">
					{env.kind}
				</span>
				<span class="flex items-center gap-1.5 text-sm text-rig-300">
					<span class="h-2 w-2 rounded-full {healthDot(env.health)}"></span>
					{healthLabel(env.health)}
				</span>
				{#if env.controlGrowId}
					<span
						class="inline-flex items-center gap-1 rounded-full border border-leaf/30 bg-leaf/10 px-2 py-0.5 text-[10px] font-medium uppercase tracking-wide text-leaf"
						title="Automation follows the assigned control grow"
					>
						<Zap size={10} /> Auto
					</span>
				{/if}
			</div>
			{#if meta}<p class="mt-1 text-sm text-rig-400">{meta}</p>{/if}
		</div>

		{#if alertCount === 0}
			<div class="flex shrink-0 items-center gap-3">
				<span class="hidden items-center gap-1.5 text-sm text-rig-400 sm:flex">
					<CircleCheck size={15} class="text-leaf" /> Everything looks good
				</span>
			</div>
		{/if}
	</div>
</header>
