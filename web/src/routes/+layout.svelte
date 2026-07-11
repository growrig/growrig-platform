<script lang="ts">
	import '../app.css';
	import { onMount } from 'svelte';
	import { page } from '$app/state';
	import { live } from '$lib/live.svelte';
	import { Button, DropdownMenu } from '$lib/components/ui';
	import Sprout from '@lucide/svelte/icons/sprout';
	import Wind from '@lucide/svelte/icons/wind';
	import Plug from '@lucide/svelte/icons/plug';
	import ChevronDown from '@lucide/svelte/icons/chevron-down';
	import TriangleAlert from '@lucide/svelte/icons/triangle-alert';
	import RotateCcw from '@lucide/svelte/icons/rotate-ccw';

	let { children } = $props();

	onMount(() => {
		live.start();
		return () => live.stop();
	});

	const nav = [
		{ href: '/', label: 'Dashboard' },
		{ href: '/activity', label: 'Activity Log' },
		{ href: '/debug', label: 'Debug' }
	];

	const statusMeta = {
		live: { label: 'Live', dot: 'bg-leaf' },
		connecting: { label: 'Connecting', dot: 'bg-warn animate-pulse' },
		offline: { label: 'Offline', dot: 'bg-danger' }
	} as const;

	const addItems = [
		{ href: '/wizard/box', label: 'Grow Box', icon: Sprout },
		{ href: '/wizard/room', label: 'Lung Room', icon: Wind },
		{ href: '/wizard/device', label: 'Device', icon: Plug }
	];
</script>

<div class="min-h-screen">
	<header class="sticky top-0 z-10 border-b border-rig-800 bg-rig-900/60 backdrop-blur">
		<div class="mx-auto flex max-w-5xl items-center gap-6 px-4 py-3">
			<a href="/" class="flex items-center gap-2 font-semibold tracking-tight">
				<span class="grid h-7 w-7 place-items-center rounded-md bg-rig-500 text-rig-950">
					<Sprout size={18} />
				</span>
				<span>GrowRig</span>
			</a>
			<nav class="flex gap-1 text-sm">
				{#each nav as item (item.href)}
					<a
						href={item.href}
						class="rounded-md px-3 py-1.5 transition-colors {page.url.pathname === item.href
							? 'bg-rig-800 text-rig-50'
							: 'text-rig-300 hover:bg-rig-800/50 hover:text-rig-100'}"
					>
						{item.label}
					</a>
				{/each}
			</nav>
			<div class="ml-auto flex items-center gap-4">
				<DropdownMenu
					items={addItems}
					triggerClass="flex items-center gap-1 rounded-md bg-rig-500 px-3 py-1.5 text-sm font-medium text-rig-950 transition-colors hover:bg-rig-400 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-rig-400 focus-visible:ring-offset-2 focus-visible:ring-offset-rig-950"
				>
					{#snippet trigger()}
						Add <ChevronDown size={14} />
					{/snippet}
				</DropdownMenu>
				<div class="flex items-center gap-2 text-sm text-rig-300">
					<span class="h-2 w-2 rounded-full {statusMeta[live.status].dot}"></span>
					{statusMeta[live.status].label}
				</div>
			</div>
		</div>
	</header>

	<main class="mx-auto max-w-5xl px-4 py-6">
		<svelte:boundary onerror={(e) => console.error('[GrowRig] page error:', e)}>
			{@render children()}

			{#snippet failed(error, reset)}
				{@const err = error instanceof Error ? error : new Error(String(error))}
				<div class="mx-auto max-w-2xl rounded-xl border border-danger/40 bg-danger/5 p-6">
					<div class="mb-2 flex items-center gap-2 text-danger">
						<TriangleAlert size={22} />
						<h1 class="text-lg font-semibold">Something broke on this page</h1>
					</div>
					<p class="mb-4 text-sm text-rig-300">
						The rest of GrowRig is still running. You can retry this view, reload, or open the
						Debug page to inspect the live state.
					</p>
					<div class="mb-4 overflow-hidden rounded-lg border border-rig-800 bg-rig-950/60">
						<div class="border-b border-rig-800 px-4 py-2 text-xs font-medium text-rig-400">
							{err.name}: {err.message}
						</div>
						{#if err.stack}
							<pre class="max-h-56 overflow-auto p-4 text-xs leading-relaxed text-rig-400"><code>{err.stack}</code></pre>
						{/if}
					</div>
					<div class="flex flex-wrap gap-2">
						<Button onclick={reset}><RotateCcw size={15} /> Try again</Button>
						<Button variant="secondary" onclick={() => location.reload()}>Reload page</Button>
						<Button variant="ghost" href="/debug">Open Debug</Button>
					</div>
				</div>
			{/snippet}
		</svelte:boundary>
	</main>
</div>
