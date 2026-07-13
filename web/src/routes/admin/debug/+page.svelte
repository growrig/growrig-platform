<script lang="ts">
	import { onMount } from 'svelte';
	import { live } from '$lib/live.svelte';
	import { getInfo, getDatabaseTables, CORE_URL, wsURL, clearActivity, restartCore } from '$lib/api';
	import type { DatabaseTable } from '$lib/api';
	import StatTile from '$lib/components/StatTile.svelte';
	import { Button } from '$lib/components/ui';
	import { fmtDateTime } from '$lib/datetime';

	let adapter = $state<string>('…');
	let infoError = $state<string | null>(null);
	let actionError = $state<string | null>(null);
	let clearing = $state(false);
	let restarting = $state(false);
	let databaseTables = $state<DatabaseTable[]>([]);
	let databaseLoading = $state(true);
	let databaseError = $state<string | null>(null);

	async function onClearActivity() {
		if (!confirm('Clear the entire activity log? This cannot be undone.')) return;
		clearing = true;
		actionError = null;
		try {
			await clearActivity();
		} catch (e) {
			actionError = e instanceof Error ? e.message : String(e);
		} finally {
			clearing = false;
		}
	}

	async function onRestart() {
		if (!confirm('Restart Grow Core now? The service will be briefly unavailable.')) return;
		restarting = true;
		actionError = null;
		try {
			await restartCore();
		} catch {
			// The connection often drops as the server shuts down; that's expected.
		}
		// Leave the button in its "restarting" state; the live indicator will show
		// reconnection once the service is back.
	}

	// A ticking clock so the "age" readouts stay live.
	let now = $state(Date.now());

	async function loadInfo() {
		try {
			adapter = (await getInfo()).adapter;
			infoError = null;
		} catch (e) {
			infoError = e instanceof Error ? e.message : String(e);
		}
	}

	async function loadDatabaseTables() {
		databaseLoading = true;
		try {
			databaseTables = await getDatabaseTables();
			databaseError = null;
		} catch (e) {
			databaseError = e instanceof Error ? e.message : String(e);
		} finally {
			databaseLoading = false;
		}
	}

	async function refreshDebug() {
		await Promise.all([loadInfo(), loadDatabaseTables()]);
	}

	onMount(() => {
		refreshDebug();
		const t = setInterval(() => (now = Date.now()), 1000);
		return () => clearInterval(t);
	});

	const snap = $derived(live.snapshot);
	const envs = $derived(snap?.environments ?? []);

	const counts = $derived({
		environments: envs.length,
		controls: envs.reduce((n, e) => n + (e.controls?.length ?? 0), 0),
		sensors: envs.reduce((n, e) => n + (e.sensors?.length ?? 0), 0),
		cameras: envs.reduce((n, e) => n + (e.cameras?.length ?? 0), 0)
	});

	const ageMs = $derived(live.lastMessageAt ? now - live.lastMessageAt : null);
	const databaseRows = $derived(databaseTables.reduce((total, table) => total + table.rows, 0));
	const databaseBytes = $derived(databaseTables.reduce((total, table) => total + table.sizeBytes, 0));

	function fmtBytes(bytes: number): string {
		if (bytes < 1024) return `${bytes} B`;
		const units = ['KB', 'MB', 'GB', 'TB'];
		let value = bytes / 1024;
		let unit = 0;
		while (value >= 1024 && unit < units.length - 1) {
			value /= 1024;
			unit++;
		}
		return `${value < 10 ? value.toFixed(1) : value.toFixed(0)} ${units[unit]}`;
	}

	function fmtAge(ms: number | null): string {
		if (ms === null) return 'never';
		if (ms < 1000) return 'just now';
		const s = Math.floor(ms / 1000);
		if (s < 60) return `${s}s ago`;
		return `${Math.floor(s / 60)}m ${s % 60}s ago`;
	}

	const statusMeta = {
		live: { label: 'Live', dot: 'bg-leaf', tone: 'good' as const },
		connecting: { label: 'Connecting', dot: 'bg-warn animate-pulse', tone: 'warn' as const },
		offline: { label: 'Offline', dot: 'bg-danger', tone: 'danger' as const }
	};

	const snapshotJson = $derived(snap ? JSON.stringify(snap, null, 2) : '// no snapshot yet');

	let copied = $state(false);
	async function copyJson() {
		try {
			await navigator.clipboard.writeText(snapshotJson);
			copied = true;
			setTimeout(() => (copied = false), 1500);
		} catch {
			/* clipboard unavailable */
		}
	}

	const rows = $derived([
		{ k: 'Connection', v: statusMeta[live.status].label },
		{ k: 'Last snapshot', v: `${fmtAge(ageMs)} · via ${live.lastSource ?? '—'}` },
		{ k: 'Snapshot time', v: snap?.time ? fmtDateTime(snap.time) : '—' },
		{ k: 'Adapter', v: adapter },
		{ k: 'Core URL', v: CORE_URL || `${'same-origin'} (${location.origin})` },
		{ k: 'WebSocket URL', v: wsURL() },
		{ k: 'Last error', v: live.lastError ?? infoError ?? 'none' }
	]);
</script>

<div class="space-y-6">
	<div class="flex items-center justify-between">
		<div class="flex items-center gap-3">
			<h2 class="text-lg font-semibold">Debug</h2>
			<span class="flex items-center gap-2 rounded-full bg-rig-800 px-3 py-1 text-xs text-rig-300">
				<span class="h-2 w-2 rounded-full {statusMeta[live.status].dot}"></span>
				{statusMeta[live.status].label}
			</span>
		</div>
		<div class="flex items-center gap-2">
			<Button variant="secondary" size="sm" onclick={onClearActivity} disabled={clearing}>
				{clearing ? 'Clearing…' : 'Clear activity log'}
			</Button>
			<Button variant="danger" size="sm" onclick={onRestart} disabled={restarting}>
				{restarting ? 'Restarting…' : 'Restart'}
			</Button>
			<Button variant="secondary" size="sm" onclick={refreshDebug}>Refresh info</Button>
		</div>
	</div>

	{#if actionError}
		<p class="rounded-md border border-danger/40 bg-danger/10 px-4 py-2 text-sm text-danger">{actionError}</p>
	{/if}

	<section class="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
		<StatTile label="Environments" value={String(counts.environments)} tone="good" />
		<StatTile label="Controls" value={String(counts.controls)} />
		<StatTile label="Sensors" value={String(counts.sensors)} />
		<StatTile label="Cameras" value={String(counts.cameras)} />
	</section>

	<section class="overflow-hidden rounded-xl border border-rig-800 bg-rig-900/40">
		<div class="flex items-center justify-between border-b border-rig-800 px-5 py-3">
			<h3 class="text-sm font-semibold uppercase tracking-wide text-rig-400">Database tables</h3>
			<span class="text-xs text-rig-500">{databaseTables.length} tables · {databaseRows.toLocaleString()} rows · {fmtBytes(databaseBytes)}</span>
		</div>
		{#if databaseLoading}
			<p class="px-5 py-4 text-sm text-rig-400">Reading database…</p>
		{:else if databaseError}
			<p class="px-5 py-4 text-sm text-danger">{databaseError}</p>
		{:else}
			<div class="max-h-[28rem] overflow-auto">
				<table class="w-full text-left text-sm">
					<thead class="sticky top-0 bg-rig-900 text-xs uppercase tracking-wide text-rig-500">
						<tr class="border-b border-rig-800">
							<th class="px-5 py-2 font-medium">Table</th>
							<th class="px-5 py-2 text-right font-medium">Rows</th>
							<th class="px-5 py-2 text-right font-medium">Size</th>
						</tr>
					</thead>
					<tbody class="divide-y divide-rig-800/70">
						{#each databaseTables as table (table.name)}
							<tr class="hover:bg-rig-800/30">
								<td class="px-5 py-2.5 font-mono text-xs text-rig-300">{table.name}</td>
								<td class="px-5 py-2.5 text-right font-mono text-xs tabular-nums text-rig-100">{table.rows.toLocaleString()}</td>
								<td class="px-5 py-2.5 text-right font-mono text-xs tabular-nums text-rig-400">{fmtBytes(table.sizeBytes)}</td>
							</tr>
						{/each}
					</tbody>
				</table>
			</div>
		{/if}
	</section>

	<section class="overflow-hidden rounded-xl border border-rig-800 bg-rig-900/40">
		<h3 class="border-b border-rig-800 px-5 py-3 text-sm font-semibold uppercase tracking-wide text-rig-400">
			Runtime
		</h3>
		<dl class="divide-y divide-rig-800/70">
			{#each rows as row (row.k)}
				<div class="flex items-center justify-between gap-4 px-5 py-2.5 text-sm">
					<dt class="text-rig-400">{row.k}</dt>
					<dd class="truncate text-right font-mono text-xs text-rig-100">{row.v}</dd>
				</div>
			{/each}
		</dl>
	</section>

	<section class="overflow-hidden rounded-xl border border-rig-800 bg-rig-900/40">
		<div class="flex items-center justify-between border-b border-rig-800 px-5 py-3">
			<h3 class="text-sm font-semibold uppercase tracking-wide text-rig-400">Live snapshot</h3>
			<Button variant="ghost" size="sm" onclick={copyJson}>{copied ? 'Copied ✓' : 'Copy JSON'}</Button>
		</div>
		<pre class="max-h-[28rem] overflow-auto p-4 text-xs leading-relaxed text-rig-200"><code>{snapshotJson}</code></pre>
	</section>
</div>
