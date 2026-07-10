<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { getEnvironments, getBindings, deleteBinding, deleteEnvironment } from '$lib/api';
	import type { Binding, Environment } from '$lib/types';
	import EnvironmentCard from '$lib/components/EnvironmentCard.svelte';
	import ArrowLeft from '@lucide/svelte/icons/arrow-left';
	import Cpu from '@lucide/svelte/icons/cpu';
	import ChevronRight from '@lucide/svelte/icons/chevron-right';

	const id = $derived(page.params.id);

	let environments = $state<Environment[]>([]);
	let bindings = $state<Binding[]>([]);
	let loading = $state(true);
	let error = $state<string | null>(null);
	let removing = $state(false);

	let notice = $state<{ kind: 'ok' | 'err'; text: string } | null>(null);
	function flash(kind: 'ok' | 'err', text: string) {
		notice = { kind, text };
		setTimeout(() => (notice = null), 2500);
	}

	async function reload() {
		try {
			[environments, bindings] = await Promise.all([getEnvironments(), getBindings()]);
			error = null;
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to reach Grow Core';
		} finally {
			loading = false;
		}
	}
	onMount(reload);

	const env = $derived(environments.find((e) => e.id === id));
	const rooms = $derived(environments.filter((e) => e.kind === 'room'));
	const myBindings = $derived(bindings.filter((b) => b.environmentId === id));

	async function removeEnvironment() {
		if (!env) return;
		const label = env.kind === 'tent' ? 'grow box' : 'room';
		if (!confirm(`Remove ${label} "${env.name}" and all its devices? This cannot be undone.`)) return;
		removing = true;
		try {
			// Cascade: delete this environment's bindings, then the environment.
			for (const b of myBindings) await deleteBinding(b.id);
			await deleteEnvironment(id!);
			await goto('/');
		} catch (e) {
			error = e instanceof Error ? e.message : 'Remove failed';
			removing = false;
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
	<div class="space-y-8">
		<h1 class="text-2xl font-semibold">{env.name} — Settings</h1>

		{#if notice}
			<div class="rounded-lg px-4 py-2 text-sm {notice.kind === 'ok' ? 'bg-leaf/15 text-leaf' : 'bg-danger/15 text-danger'}">
				{notice.text}
			</div>
		{/if}

		<!-- Environment settings -->
		<EnvironmentCard {env} {rooms} canDelete={false} onChanged={reload} {flash} />

		<!-- Devices -->
		<section class="space-y-3">
			<h2 class="text-sm font-semibold uppercase tracking-wide text-rig-400">Devices</h2>
			<a
				href="/env/{id}/devices"
				class="flex items-center gap-3 rounded-xl border border-rig-800 bg-rig-900/40 p-4 transition-colors hover:border-rig-600"
			>
				<span class="grid h-10 w-10 place-items-center rounded-lg bg-rig-800 text-rig-300">
					<Cpu size={20} />
				</span>
				<div class="min-w-0 flex-1">
					<div class="text-sm font-medium">Manage devices</div>
					<div class="text-xs text-rig-500">
						{myBindings.length} device{myBindings.length === 1 ? '' : 's'} · add, edit or remove sensors, fans, lights and cameras
					</div>
				</div>
				<ChevronRight size={18} class="text-rig-500" />
			</a>
		</section>

		<!-- Danger zone -->
		<section class="space-y-3">
			<h2 class="text-sm font-semibold uppercase tracking-wide text-danger/80">Danger zone</h2>
			<div class="flex items-center justify-between rounded-xl border border-danger/30 bg-danger/5 p-4">
				<div>
					<div class="text-sm font-medium">Remove this {env.kind === 'tent' ? 'grow box' : 'room'}</div>
					<div class="text-xs text-rig-500">Deletes the environment and all {myBindings.length} of its devices.</div>
				</div>
				<button
					onclick={removeEnvironment}
					disabled={removing}
					class="rounded-md bg-danger/90 px-4 py-1.5 text-sm font-medium text-white transition-colors hover:bg-danger disabled:opacity-50"
				>
					{removing ? 'Removing…' : env.kind === 'tent' ? 'Remove tent' : 'Remove room'}
				</button>
			</div>
		</section>
	</div>
{/if}
