<script lang="ts">
	import { goto } from '$app/navigation';
	import { errMsg } from '$lib/errors';
	import { live } from '$lib/live.svelte';
	import { resolveLocationId } from '$lib/location';
	import { formatDimensions, volumeM3 } from '$lib/format';
	import {
		deleteBinding,
		deleteEnvironment,
		getBindings,
		getEnvironmentYAML,
		updateEnvironmentYAML
	} from '$lib/api';
	import { Button, Dialog, DropdownMenu, type DropdownItem } from '$lib/components/ui';
	import type { EnvironmentView, Location } from '$lib/types';
	import Settings from '@lucide/svelte/icons/settings';
	import Code2 from '@lucide/svelte/icons/code-2';
	import Trash2 from '@lucide/svelte/icons/trash-2';
	import MoreHorizontal from '@lucide/svelte/icons/more-horizontal';
	import Zap from '@lucide/svelte/icons/zap';
	import CircleCheck from '@lucide/svelte/icons/circle-check';

	interface Props {
		env: EnvironmentView;
		locations: Location[];
		/** Open alerts scoped to this environment; 0 → "everything looks good". */
		alertCount: number;
		isAdmin: boolean;
	}
	let { env, locations, alertCount, isAdmin }: Props = $props();

	let removing = $state(false);
	let yamlOpen = $state(false);
	let yamlText = $state('');
	let yamlBusy = $state(false);
	let yamlError = $state('');

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

	async function openYAML() {
		yamlBusy = true;
		yamlError = '';
		try {
			yamlText = await getEnvironmentYAML(env.id);
			yamlOpen = true;
		} catch (e) {
			alert(errMsg(e, 'Could not load YAML'));
		} finally {
			yamlBusy = false;
		}
	}

	async function saveYAML() {
		yamlBusy = true;
		yamlError = '';
		try {
			await updateEnvironmentYAML(env.id, yamlText);
			yamlOpen = false;
			window.location.reload();
		} catch (e) {
			yamlError = errMsg(e, 'Could not save YAML');
		} finally {
			yamlBusy = false;
		}
	}

	async function removeEnvironment() {
		const label = env.kind === 'tent' ? 'grow box' : 'room';
		if (!confirm(`Delete ${label} "${env.name}" and all its devices? This cannot be undone.`)) return;
		removing = true;
		try {
			const bindings = (await getBindings()).filter((b) => b.environmentId === env.id);
			for (const b of bindings) await deleteBinding(b.id);
			await deleteEnvironment(env.id);
			await goto('/');
		} catch (e) {
			alert(errMsg(e, 'Could not delete environment'));
			removing = false;
		}
	}

	const menu = $derived<DropdownItem[]>([
		{ label: 'Settings', href: `/env/${env.id}/settings`, icon: Settings },
		...(isAdmin
			? [
					{ label: 'Edit YAML', onSelect: openYAML, icon: Code2, disabled: yamlBusy },
					{
						label: 'Delete environment',
						onSelect: removeEnvironment,
						icon: Trash2,
						danger: true,
						disabled: removing
					}
				]
			: [])
	]);
</script>

<header class="space-y-1.5">
	<!-- Breadcrumb -->
	<nav class="flex items-center gap-1 text-xs text-rig-500">
		<a href="/" class="hover:text-rig-300">All environments</a>
		{#each crumbs as crumb (crumb.href + crumb.label)}
			<span class="text-rig-700">/</span>
			<a href={crumb.href} class="hover:text-rig-300">{crumb.label}</a>
		{/each}
	</nav>

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

		<div class="flex shrink-0 items-center gap-3">
			{#if alertCount === 0}
				<span class="hidden items-center gap-1.5 text-sm text-rig-400 sm:flex">
					<CircleCheck size={15} class="text-leaf" /> Everything looks good
				</span>
			{/if}
			<DropdownMenu
				items={menu}
				align="end"
				triggerClass="grid h-9 w-9 place-items-center rounded-md border border-rig-700 text-rig-300 outline-none transition-colors hover:border-leaf hover:text-rig-100"
			>
				{#snippet trigger()}
					<MoreHorizontal size={18} />
				{/snippet}
			</DropdownMenu>
		</div>
	</div>
</header>

{#if isAdmin}
	<Dialog
		bind:open={yamlOpen}
		title="Edit environment YAML"
		description="Changes are validated and applied immediately. Keep the environment id unchanged."
	>
		<div class="space-y-3">
			<textarea
				bind:value={yamlText}
				rows="24"
				spellcheck="false"
				class="w-full resize-y rounded-md border border-rig-700 bg-rig-950 p-3 font-mono text-xs leading-5 text-rig-200 focus:border-leaf focus:outline-none"
			></textarea>
			{#if yamlError}<p class="text-sm text-danger">{yamlError}</p>{/if}
			<div class="flex justify-end gap-2">
				<Button variant="ghost" onclick={() => (yamlOpen = false)} disabled={yamlBusy}>Cancel</Button>
				<Button onclick={saveYAML} disabled={yamlBusy || !yamlText.trim()}>
					{yamlBusy ? 'Saving…' : 'Save YAML'}
				</Button>
			</div>
		</div>
	</Dialog>
{/if}
