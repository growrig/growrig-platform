<script lang="ts">
	import { attention } from '$lib/attention.svelte';
	import { completeTask, skipTask, resolveAlert } from '$lib/api';
	import { careVisual } from '$lib/care';
	import type { Alert, Task } from '$lib/types';
	import TriangleAlert from '@lucide/svelte/icons/triangle-alert';
	import Info from '@lucide/svelte/icons/info';
	import Check from '@lucide/svelte/icons/check';
	import X from '@lucide/svelte/icons/x';
	import Package from '@lucide/svelte/icons/package';
	import Plug from '@lucide/svelte/icons/plug';
	import CircleCheck from '@lucide/svelte/icons/circle-check';

	const data = $derived(attention.data);
	const hasAnything = $derived(attention.count > 0);

	// Split due tasks into overdue vs due-today for the two sections.
	const now = $derived(Date.now());
	const overdue = $derived(data.tasks.filter((t) => t.dueAt && new Date(t.dueAt).getTime() < now));
	const dueToday = $derived(data.tasks.filter((t) => !overdue.includes(t)));

	let busy = $state<Record<string, boolean>>({});

	async function onComplete(t: Task) {
		busy = { ...busy, [t.id]: true };
		try {
			await completeTask(t.id);
			await attention.load();
		} finally {
			busy = { ...busy, [t.id]: false };
		}
	}
	async function onSkip(t: Task) {
		busy = { ...busy, [t.id]: true };
		try {
			await skipTask(t.id);
			await attention.load();
		} finally {
			busy = { ...busy, [t.id]: false };
		}
	}
	async function onDismiss(a: Alert) {
		busy = { ...busy, [a.id]: true };
		try {
			await resolveAlert(a.id);
			await attention.load();
		} finally {
			busy = { ...busy, [a.id]: false };
		}
	}

	const severityTone: Record<Alert['severity'], string> = {
		critical: 'border-danger/40 bg-danger/5 text-danger',
		warning: 'border-warn/40 bg-warn/5 text-warn',
		info: 'border-rig-700 bg-rig-900/40 text-rig-300'
	};
</script>

{#snippet taskRow(t: Task)}
	{@const v = careVisual(t.actionType || 'inspect')}
	<div class="flex items-center gap-3 rounded-lg border border-rig-800 bg-rig-900/40 px-3 py-2.5">
		<span class="grid h-8 w-8 shrink-0 place-items-center rounded-md bg-rig-800 text-rig-300">
			<v.icon size={16} />
		</span>
		<div class="min-w-0 flex-1">
			<p class="truncate text-sm text-rig-100">{t.title}</p>
			{#if t.dueAt}
				<p class="text-xs text-rig-500">
					Due {new Date(t.dueAt).toLocaleDateString(undefined, { month: 'short', day: 'numeric' })}
				</p>
			{/if}
		</div>
		<div class="flex shrink-0 items-center gap-1.5">
			<button
				onclick={() => onComplete(t)}
				disabled={busy[t.id]}
				title="Mark done"
				aria-label="Mark done"
				class="grid h-8 w-8 place-items-center rounded-md bg-rig-500 text-rig-950 transition-colors hover:bg-rig-400 disabled:opacity-40"
			>
				<Check size={16} />
			</button>
			<button
				onclick={() => onSkip(t)}
				disabled={busy[t.id]}
				title="Skip"
				aria-label="Skip"
				class="grid h-8 w-8 place-items-center rounded-md border border-rig-700 text-rig-400 transition-colors hover:border-rig-500 hover:text-rig-100 disabled:opacity-40"
			>
				<X size={15} />
			</button>
		</div>
	</div>
{/snippet}

<section>
	<div class="mb-4 flex items-center justify-between gap-4">
		<h1 class="flex items-center gap-1.5 text-sm font-semibold uppercase tracking-wide text-leaf">
			<TriangleAlert size={14} /> Needs attention
		</h1>
	</div>

	{#if !hasAnything}
		<div
			class="flex items-center gap-3 rounded-xl border border-rig-800 bg-rig-900/30 p-5 text-sm text-rig-400"
		>
			<CircleCheck size={20} class="text-leaf" />
			<span>All good — nothing needs your attention right now.</span>
		</div>
	{:else}
		<div class="space-y-3">
			<!-- Alerts: something is wrong. -->
			{#each data.alerts as a (a.id)}
				<div class="flex items-center gap-3 rounded-lg border px-3 py-2.5 {severityTone[a.severity]}">
					<span class="shrink-0">
						{#if a.severity === 'info'}<Info size={16} />{:else}<TriangleAlert size={16} />{/if}
					</span>
					<div class="min-w-0 flex-1">
						<p class="truncate text-sm font-medium">{a.title}</p>
						{#if a.message && a.message !== a.title}
							<p class="truncate text-xs opacity-80">{a.message}</p>
						{/if}
					</div>
					<button
						onclick={() => onDismiss(a)}
						disabled={busy[a.id]}
						title="Dismiss"
						aria-label="Dismiss alert"
						class="grid h-8 w-8 shrink-0 place-items-center rounded-md border border-current/30 opacity-70 transition-opacity hover:opacity-100 disabled:opacity-40"
					>
						<X size={15} />
					</button>
				</div>
			{/each}

			<!-- Overdue tasks. -->
			{#each overdue as t (t.id)}{@render taskRow(t)}{/each}

			<!-- Low inventory. -->
			{#each data.lowStock as item (item.id)}
				<a
					href="/inventory"
					class="flex items-center gap-3 rounded-lg border border-warn/30 bg-warn/5 px-3 py-2.5 transition-colors hover:border-warn/60"
				>
					<Package size={16} class="shrink-0 text-warn" />
					<div class="min-w-0 flex-1">
						<p class="truncate text-sm text-rig-100">Low stock: {item.name}</p>
						<p class="text-xs text-rig-500">{item.quantity} left · {item.category}</p>
					</div>
				</a>
			{/each}

			<!-- Unhealthy integrations. -->
			{#each data.integrations as ig (ig.id)}
				<a
					href="/admin"
					class="flex items-center gap-3 rounded-lg border border-danger/30 bg-danger/5 px-3 py-2.5 transition-colors hover:border-danger/60"
				>
					<Plug size={16} class="shrink-0 text-danger" />
					<div class="min-w-0 flex-1">
						<p class="truncate text-sm text-rig-100">{ig.name} · {ig.status}</p>
						{#if ig.message}<p class="truncate text-xs text-rig-500">{ig.message}</p>{/if}
					</div>
				</a>
			{/each}
		</div>
	{/if}
</section>

{#if dueToday.length}
	<section>
		<div class="mb-4 flex items-center gap-1.5 text-sm font-semibold uppercase tracking-wide text-leaf">
			Today
		</div>
		<div class="space-y-3">
			{#each dueToday as t (t.id)}{@render taskRow(t)}{/each}
		</div>
	</section>
{/if}
