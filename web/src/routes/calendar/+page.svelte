<script lang="ts">
	import { live } from '$lib/live.svelte';
	import { preferences } from '$lib/preferences.svelte';
	import { getCalendar } from '$lib/api';
	import { careVisual, ago, fmtVolume, GROW_COLORS, type GrowColor } from '$lib/care';
	import type { CalendarEvent, GrowView } from '$lib/types';
	import ChevronLeft from '@lucide/svelte/icons/chevron-left';
	import ChevronRight from '@lucide/svelte/icons/chevron-right';
	import CalendarDays from '@lucide/svelte/icons/calendar-days';
	import Sprout from '@lucide/svelte/icons/sprout';
	import Flag from '@lucide/svelte/icons/flag';
	import ArrowRight from '@lucide/svelte/icons/arrow-right';
	import X from '@lucide/svelte/icons/x';

	// A grow lifecycle point derived from the live snapshot (not a care event):
	// when a grow started, and when it entered its current stage.
	interface Marker {
		growId: string;
		kind: 'started' | 'stage';
		label: string;
	}
	interface DayCell {
		date: Date;
		inMonth: boolean;
		today: boolean;
		key: string;
	}

	const snap = $derived(live.snapshot);
	const grows = $derived<GrowView[]>(snap?.grows ?? []);
	const activeGrows = $derived(
		grows
			.filter((g) => g.status === 'active')
			.sort((a, b) => new Date(a.startedAt).getTime() - new Date(b.startedAt).getTime())
	);

	// The month currently on screen, anchored to its first day.
	let viewMonth = $state(startOfMonth(new Date()));
	let events = $state<CalendarEvent[]>([]);
	let loading = $state(false);
	let loadError = $state<string | null>(null);
	let selectedKey = $state<string | null>(null);

	function startOfMonth(d: Date): Date {
		return new Date(d.getFullYear(), d.getMonth(), 1);
	}
	// Local Y-M-D key, so events and markers group by the day the user sees.
	function dateKey(d: Date): string {
		return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`;
	}
	function sameDay(a: Date, b: Date): boolean {
		return dateKey(a) === dateKey(b);
	}

	// Six-week grid (stable height) starting on the Monday on/before the 1st.
	const grid = $derived.by<DayCell[]>(() => {
		const first = startOfMonth(viewMonth);
		const dow = (first.getDay() + 6) % 7; // Mon=0 … Sun=6
		const start = new Date(first);
		start.setDate(first.getDate() - dow);
		const today = new Date();
		return Array.from({ length: 42 }, (_, i) => {
			const date = new Date(start);
			date.setDate(start.getDate() + i);
			return {
				date,
				inMonth: date.getMonth() === viewMonth.getMonth(),
				today: sameDay(date, today),
				key: dateKey(date)
			};
		});
	});

	// Fetch the care feed for exactly the visible window whenever the month moves.
	$effect(() => {
		const cells = grid;
		const from = cells[0].key;
		const to = cells[cells.length - 1].key;
		loading = true;
		loadError = null;
		getCalendar(from, to)
			.then((r) => (events = r.events))
			.catch((e) => (loadError = e instanceof Error ? e.message : 'Failed to load calendar'))
			.finally(() => (loading = false));
	});

	// Stable colour per grow, assigned over all grows (so completed grows keep a
	// colour in history too), keyed by start order.
	const colorFor = $derived.by(() => {
		const ordered = [...grows].sort(
			(a, b) => new Date(a.startedAt).getTime() - new Date(b.startedAt).getTime()
		);
		const map = new Map<string, GrowColor>();
		ordered.forEach((g, i) => map.set(g.id, GROW_COLORS[i % GROW_COLORS.length]));
		return (id: string): GrowColor => map.get(id) ?? GROW_COLORS[0];
	});

	// Group care events and grow markers by day key for O(1) cell lookup.
	const byDay = $derived.by(() => {
		const m = new Map<string, { events: CalendarEvent[]; markers: Marker[] }>();
		const bucket = (key: string) => {
			let b = m.get(key);
			if (!b) m.set(key, (b = { events: [], markers: [] }));
			return b;
		};
		for (const e of events) bucket(dateKey(new Date(e.occurredAt))).events.push(e);
		for (const g of grows) {
			if (g.status !== 'active') continue;
			bucket(dateKey(new Date(g.startedAt))).markers.push({ growId: g.id, kind: 'started', label: 'Started' });
			// Only mark the stage change when it is a different day than the start.
			if (!sameDay(new Date(g.stageStarted), new Date(g.startedAt))) {
				bucket(dateKey(new Date(g.stageStarted))).markers.push({
					growId: g.id,
					kind: 'stage',
					label: g.stage
				});
			}
		}
		return m;
	});

	// Most recent water/feed per active grow within the loaded window — the
	// planning hint. Events arrive newest-first, so the first hit wins.
	const lastCare = $derived.by(() => {
		const m = new Map<string, { water?: string; feed?: string }>();
		for (const e of events) {
			const rec = m.get(e.growId) ?? {};
			if (e.type === 'water' && !rec.water) rec.water = e.occurredAt;
			if (e.type === 'feed' && !rec.feed) rec.feed = e.occurredAt;
			m.set(e.growId, rec);
		}
		return m;
	});

	const monthLabel = $derived(
		new Intl.DateTimeFormat(preferences.locale, { month: 'long', year: 'numeric' }).format(viewMonth)
	);
	const weekdays = $derived.by(() => {
		const base = new Date(2024, 0, 1); // a Monday
		const fmt = new Intl.DateTimeFormat(preferences.locale, { weekday: 'short' });
		return Array.from({ length: 7 }, (_, i) => {
			const d = new Date(base);
			d.setDate(base.getDate() + i);
			return fmt.format(d);
		});
	});

	// Format a calendar cell's date. The grid is built in browser-local time, so
	// the heading must format in that same basis (not preferences.timezone) or it
	// can disagree with the day number the user clicked.
	function fmtCellDate(d: Date): string {
		return new Intl.DateTimeFormat(preferences.locale, {
			weekday: 'long',
			day: 'numeric',
			month: 'long'
		}).format(d);
	}

	const selectedCell = $derived(selectedKey ? (grid.find((c) => c.key === selectedKey) ?? null) : null);
	const selectedDay = $derived(selectedKey ? (byDay.get(selectedKey) ?? { events: [], markers: [] }) : null);

	function shiftMonth(delta: number) {
		viewMonth = new Date(viewMonth.getFullYear(), viewMonth.getMonth() + delta, 1);
		selectedKey = null;
	}
	function goToday() {
		viewMonth = startOfMonth(new Date());
		selectedKey = dateKey(new Date());
	}
	const growName = (id: string) => grows.find((g) => g.id === id)?.name ?? 'Grow';
</script>

<div class="mb-6 flex flex-wrap items-end justify-between gap-4">
	<div>
		<h1 class="flex items-center gap-2 text-2xl font-semibold">
			<CalendarDays size={24} class="text-rig-400" /> Calendar
		</h1>
		<p class="text-sm text-rig-400">Care journal and grow milestones across every grow, day by day.</p>
	</div>
	<div class="flex items-center gap-1">
		<button
			onclick={() => shiftMonth(-1)}
			class="grid h-8 w-8 place-items-center rounded-md border border-rig-800 text-rig-300 transition-colors hover:bg-rig-800/60"
			aria-label="Previous month"
		>
			<ChevronLeft size={16} />
		</button>
		<button
			onclick={goToday}
			class="rounded-md border border-rig-800 px-3 py-1.5 text-sm text-rig-200 transition-colors hover:bg-rig-800/60"
		>
			Today
		</button>
		<button
			onclick={() => shiftMonth(1)}
			class="grid h-8 w-8 place-items-center rounded-md border border-rig-800 text-rig-300 transition-colors hover:bg-rig-800/60"
			aria-label="Next month"
		>
			<ChevronRight size={16} />
		</button>
	</div>
</div>

<div class="grid gap-6 lg:grid-cols-[1fr_20rem]">
	<!-- Calendar grid -->
	<section>
		<div class="mb-3 flex items-center gap-3">
			<h2 class="text-lg font-semibold capitalize">{monthLabel}</h2>
			{#if loading}<span class="text-xs text-rig-500">Loading…</span>{/if}
			{#if loadError}<span class="text-xs text-danger">{loadError}</span>{/if}
		</div>

		<div class="grid grid-cols-7 gap-px overflow-hidden rounded-xl border border-rig-800 bg-rig-800">
			{#each weekdays as wd (wd)}
				<div class="bg-rig-900/60 py-2 text-center text-xs font-medium uppercase tracking-wide text-rig-400">
					{wd}
				</div>
			{/each}

			{#each grid as cell (cell.key)}
				{@const day = byDay.get(cell.key)}
				{@const evs = day?.events ?? []}
				{@const marks = day?.markers ?? []}
				<button
					type="button"
					onclick={() => (selectedKey = selectedKey === cell.key ? null : cell.key)}
					class="flex min-h-[6.5rem] flex-col gap-1 p-1.5 text-left transition-colors
						{cell.inMonth ? 'bg-rig-900/40' : 'bg-rig-950/40 text-rig-600'}
						{selectedKey === cell.key ? 'ring-1 ring-inset ring-rig-500' : 'hover:bg-rig-800/40'}"
				>
					<div class="flex items-center justify-between">
						<span
							class="grid h-6 w-6 place-items-center rounded-full text-xs
								{cell.today ? 'bg-rig-500 font-semibold text-rig-950' : cell.inMonth ? 'text-rig-300' : 'text-rig-600'}"
						>
							{cell.date.getDate()}
						</span>
					</div>

					{#each marks as mk (mk.growId + mk.kind)}
						<div
							class="flex items-center gap-1 truncate rounded border-l-2 bg-rig-800/50 px-1 py-0.5 text-[10px] leading-tight text-rig-300 {colorFor(mk.growId).border}"
							title="{growName(mk.growId)} · {mk.kind === 'started' ? 'Grow started' : `Entered ${mk.label}`}"
						>
							{#if mk.kind === 'started'}<Flag size={9} class="shrink-0" />{:else}<ArrowRight size={9} class="shrink-0" />{/if}
							<span class="truncate">{mk.kind === 'started' ? 'Started' : mk.label}</span>
						</div>
					{/each}

					{#each evs.slice(0, 3) as e (e.id)}
						{@const v = careVisual(e.type)}
						{@const Icon = v.icon}
						<div
							class="flex items-center gap-1 truncate rounded border-l-2 bg-rig-800/60 px-1 py-0.5 text-[10px] leading-tight {colorFor(e.growId).border}"
							title="{e.growName} · {v.label}{e.plantCount ? ` · ${e.plantCount} plant${e.plantCount === 1 ? '' : 's'}` : ''}{e.totalMl ? ` · ${fmtVolume(e.totalMl)}` : ''}"
						>
							<Icon size={10} class="shrink-0 {colorFor(e.growId).text}" />
							<span class="truncate text-rig-200">{v.label}</span>
							{#if e.plantCount > 1}<span class="ml-auto shrink-0 text-rig-500">×{e.plantCount}</span>{/if}
						</div>
					{/each}
					{#if evs.length > 3}
						<span class="px-1 text-[10px] text-rig-500">+{evs.length - 3} more</span>
					{/if}
				</button>
			{/each}
		</div>

		<!-- Legend -->
		{#if activeGrows.length}
			<div class="mt-3 flex flex-wrap gap-x-4 gap-y-1.5 text-xs text-rig-400">
				{#each activeGrows as g (g.id)}
					<span class="flex items-center gap-1.5">
						<span class="h-2.5 w-2.5 rounded-full {colorFor(g.id).dot}"></span>
						{g.name}
					</span>
				{/each}
			</div>
		{/if}
	</section>

	<!-- Sidebar: selected-day detail, else active-grow planning -->
	<aside class="space-y-4">
		{#if selectedCell}
			<section class="rounded-xl border border-rig-800 bg-rig-900/40 p-4">
				<div class="mb-3 flex items-center justify-between">
					<h2 class="text-sm font-semibold text-rig-100">{fmtCellDate(selectedCell.date)}</h2>
					<button onclick={() => (selectedKey = null)} class="text-rig-500 hover:text-rig-200" aria-label="Close">
						<X size={15} />
					</button>
				</div>
				{#if selectedDay && (selectedDay.events.length || selectedDay.markers.length)}
					<div class="space-y-2">
						{#each selectedDay.markers as mk (mk.growId + mk.kind)}
							<div class="flex items-center gap-2 rounded-lg border-l-2 bg-rig-800/40 px-2.5 py-2 text-sm {colorFor(mk.growId).border}">
								{#if mk.kind === 'started'}<Flag size={14} class="shrink-0 text-rig-400" />{:else}<ArrowRight size={14} class="shrink-0 text-rig-400" />{/if}
								<div class="min-w-0">
									<div class="truncate font-medium text-rig-100">{growName(mk.growId)}</div>
									<div class="text-xs text-rig-400">{mk.kind === 'started' ? 'Grow started' : `Entered ${mk.label}`}</div>
								</div>
							</div>
						{/each}
						{#each selectedDay.events as e (e.id)}
							{@const v = careVisual(e.type)}
							{@const Icon = v.icon}
							<div class="flex items-start gap-2 rounded-lg border-l-2 bg-rig-800/40 px-2.5 py-2 text-sm {colorFor(e.growId).border}">
								<Icon size={14} class="mt-0.5 shrink-0 {colorFor(e.growId).text}" />
								<div class="min-w-0 flex-1">
									<div class="flex items-baseline justify-between gap-2">
										<span class="font-medium text-rig-100">{v.label}</span>
										<span class="shrink-0 text-xs text-rig-500">{e.plantCount} plant{e.plantCount === 1 ? '' : 's'}</span>
									</div>
									<div class="truncate text-xs text-rig-400">{e.growName}</div>
									{#if e.recipeName || e.totalMl}
										<div class="mt-0.5 text-xs text-rig-500">
											{#if e.recipeName}{e.recipeName}{/if}{#if e.recipeName && e.totalMl} · {/if}{#if e.totalMl}{fmtVolume(e.totalMl)}{/if}
										</div>
									{/if}
									{#if e.notes}<div class="mt-0.5 text-xs italic text-rig-500">“{e.notes}”</div>{/if}
								</div>
							</div>
						{/each}
					</div>
				{:else}
					<p class="text-sm text-rig-500">Nothing logged on this day.</p>
				{/if}
			</section>
		{:else}
			<section class="rounded-xl border border-rig-800 bg-rig-900/40 p-4">
				<h2 class="mb-3 flex items-center gap-1.5 text-sm font-semibold uppercase tracking-wide text-leaf">
					<Sprout size={14} /> Active grows
				</h2>
				{#if !snap}
					<p class="text-sm text-rig-500">Connecting to Grow Core…</p>
				{:else if activeGrows.length === 0}
					<p class="text-sm text-rig-500">No active grows. <a href="/grows" class="text-rig-300 underline hover:text-rig-100">Start one</a>.</p>
				{:else}
					<ul class="space-y-3">
						{#each activeGrows as g (g.id)}
							{@const lc = lastCare.get(g.id)}
							<li>
								<a href="/grows/{g.id}" class="block rounded-lg border border-rig-800 bg-rig-900/40 p-3 transition-colors hover:border-rig-700 hover:bg-rig-800/40">
									<div class="flex items-center gap-2">
										<span class="h-2.5 w-2.5 shrink-0 rounded-full {colorFor(g.id).dot}"></span>
										<span class="truncate font-medium text-rig-100">{g.name}</span>
										<span class="ml-auto shrink-0 text-xs text-rig-500">Day {g.totalDays}</span>
									</div>
									<div class="mt-1 flex flex-wrap items-center gap-x-2 gap-y-0.5 pl-[18px] text-xs text-rig-400">
										<span class="capitalize">{g.stage}</span>
										<span class="text-rig-600">·</span>
										<span>{g.plantCount} plant{g.plantCount === 1 ? '' : 's'}</span>
									</div>
									<div class="mt-1.5 flex flex-wrap gap-x-3 gap-y-0.5 pl-[18px] text-xs">
										<span class="text-rig-500">Watered <span class="text-rig-300">{lc?.water ? ago(lc.water) : '—'}</span></span>
										<span class="text-rig-500">Fed <span class="text-rig-300">{lc?.feed ? ago(lc.feed) : '—'}</span></span>
									</div>
								</a>
							</li>
						{/each}
					</ul>
					<p class="mt-3 text-[11px] leading-relaxed text-rig-600">Last-care hints reflect the visible month.</p>
				{/if}
			</section>
		{/if}
	</aside>
</div>
