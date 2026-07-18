<script lang="ts">
	import type { GrowDetail } from '$lib/types';
	import { titleCase } from '$lib/format';
	import { Breadcrumb } from '$lib/components/ui';
	import PhotoUpload from './PhotoUpload.svelte';
	import Droplet from '@lucide/svelte/icons/droplet';
	import CheckCircle from '@lucide/svelte/icons/circle-check';

	interface Props {
		grow: GrowDetail;
		isAdmin: boolean;
		canLogCare: boolean;
		dueCount: number;
		onLogCare: () => void;
		onPhotoUploaded: () => void;
	}
	let { grow, isAdmin, canLogCare, dueCount, onLogCare, onPhotoUploaded }: Props = $props();

	const statusTone = (s: string) =>
		s === 'active' ? 'text-leaf' : s === 'harvested' ? 'text-warn' : 'text-rig-500';

	// Where the grow's active plants currently live (distinct env names).
	const envNames = $derived([
		...new Set(
			grow.plants
				.filter((p) => p.status === 'active' && p.currentEnvironmentName)
				.map((p) => p.currentEnvironmentName)
		)
	]);
	const envLabel = $derived(envNames.length === 1 ? envNames[0] : envNames.length > 1 ? `${envNames.length} environments` : '');
</script>

<header class="space-y-1.5">
	<Breadcrumb items={[{ label: 'All grows', href: '/grows' }]} />

	<div class="flex flex-wrap items-start justify-between gap-3">
		<div class="min-w-0">
			<div class="flex items-center gap-3">
				<h1 class="text-2xl font-semibold">{grow.name}</h1>
				<span class="rounded-full bg-rig-800 px-2 py-0.5 text-xs capitalize {statusTone(grow.status)}">{grow.status}</span>
			</div>
			<p class="mt-1 text-sm text-rig-400">
				{titleCase(grow.species) || 'No species'} · Day {grow.totalDays} · {grow.plantCount} plant{grow.plantCount === 1 ? '' : 's'}{#if envLabel} · {envLabel}{/if}
			</p>
		</div>

		<div class="flex shrink-0 items-center gap-2">
			{#if dueCount === 0}
				<span class="hidden items-center gap-1.5 text-sm text-rig-400 md:flex">
					<CheckCircle size={15} class="text-leaf" /> Everything looks good
				</span>
			{/if}
			{#if isAdmin}
				<PhotoUpload growId={grow.id} onUploaded={onPhotoUploaded} />
				{#if canLogCare}
					<button
						onclick={onLogCare}
						class="inline-flex items-center gap-1.5 rounded-md bg-rig-50 px-4 py-1.5 text-sm font-medium text-rig-950 transition-colors hover:bg-rig-200"
					>
						<Droplet size={15} /> Log care
					</button>
				{/if}
			{/if}
		</div>
	</div>
</header>
