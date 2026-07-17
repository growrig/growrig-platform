<script lang="ts">
	import type { GrowDetail } from '$lib/types';
	import { titleCase } from '$lib/format';
	import { DropdownMenu, type DropdownItem } from '$lib/components/ui';
	import PhotoUpload from './PhotoUpload.svelte';
	import ArrowLeft from '@lucide/svelte/icons/arrow-left';
	import Droplet from '@lucide/svelte/icons/droplet';
	import MoreHorizontal from '@lucide/svelte/icons/more-horizontal';
	import Pencil from '@lucide/svelte/icons/pencil';
	import CheckCircle from '@lucide/svelte/icons/circle-check';
	import Trash2 from '@lucide/svelte/icons/trash-2';
	import Settings2 from '@lucide/svelte/icons/settings-2';

	interface Props {
		grow: GrowDetail;
		isAdmin: boolean;
		canLogCare: boolean;
		dueCount: number;
		onLogCare: () => void;
		onPhotoUploaded: () => void;
		onEdit: () => void;
		onComplete: () => void;
		onDelete: () => void;
		onCareSettings: () => void;
	}
	let {
		grow,
		isAdmin,
		canLogCare,
		dueCount,
		onLogCare,
		onPhotoUploaded,
		onEdit,
		onComplete,
		onDelete,
		onCareSettings
	}: Props = $props();

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

	const menu = $derived<DropdownItem[]>([
		{ label: 'Edit grow', onSelect: onEdit, icon: Pencil },
		{ label: 'Care actions', onSelect: onCareSettings, icon: Settings2 },
		...(grow.status === 'active' ? [{ label: 'Complete grow', onSelect: onComplete, icon: CheckCircle }] : []),
		{ label: 'Delete grow', onSelect: onDelete, icon: Trash2 }
	]);
</script>

<header class="space-y-1.5">
	<a href="/grows" class="inline-flex items-center gap-1 text-xs text-rig-500 hover:text-rig-300">
		<ArrowLeft size={13} /> All grows
	</a>

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
						class="inline-flex items-center gap-1.5 rounded-md bg-rig-500 px-4 py-1.5 text-sm font-medium text-rig-950 transition-colors hover:bg-rig-400"
					>
						<Droplet size={15} /> Log care
					</button>
				{/if}
				<DropdownMenu
					items={menu}
					align="end"
					triggerClass="grid h-9 w-9 place-items-center rounded-md border border-rig-700 text-rig-300 outline-none transition-colors hover:border-rig-500 hover:text-rig-100"
				>
					{#snippet trigger()}
						<MoreHorizontal size={18} />
					{/snippet}
				</DropdownMenu>
			{/if}
		</div>
	</div>
</header>
