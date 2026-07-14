<script lang="ts">
	// System status — the admin landing page. A plain-language, high-level
	// overview of the whole running GrowRig system, written for growers rather
	// than engineers: is everything online, is Home Assistant healthy, and are
	// any grow spaces asking for attention. Deep-dive pages (Home Assistant,
	// Integrations, Debug) live behind the links here.
	import { onMount } from 'svelte';
	import { live } from '$lib/live.svelte';
	import { getHomeAssistant, getIntegrationInstances } from '$lib/api';
	import type { HAStatus, IntegrationInstance, EnvironmentView } from '$lib/types';
	import type { Tone } from '$lib/format';
	import StatTile from '$lib/components/StatTile.svelte';
	import CircleCheck from '@lucide/svelte/icons/circle-check';
	import TriangleAlert from '@lucide/svelte/icons/triangle-alert';
	import CircleX from '@lucide/svelte/icons/circle-x';
	import Activity from '@lucide/svelte/icons/activity';
	import HousePlug from '@lucide/svelte/icons/house-plug';
	import Blocks from '@lucide/svelte/icons/blocks';
	import ChevronRight from '@lucide/svelte/icons/chevron-right';
	import Sprout from '@lucide/svelte/icons/sprout';

	let ha = $state<HAStatus | null>(null);
	let instances = $state<IntegrationInstance[]>([]);
	let loaded = $state(false);
	// A ticking clock so the "updated … ago" line stays live.
	let now = $state(Date.now());

	onMount(() => {
		load();
		const t = setInterval(() => (now = Date.now()), 1000);
		return () => clearInterval(t);
	});

	async function load() {
		// Both are best-effort: the overview still renders (and stays useful)
		// if either the Home Assistant probe or the integrations list fails.
		const [h, i] = await Promise.allSettled([getHomeAssistant(), getIntegrationInstances()]);
		if (h.status === 'fulfilled') ha = h.value;
		if (i.status === 'fulfilled') instances = i.value;
		loaded = true;
	}

	const snap = $derived(live.snapshot);
	const envs = $derived<EnvironmentView[]>(snap?.environments ?? []);
	const activeGrows = $derived((snap?.grows ?? []).filter((g) => g.status === 'active'));

	const counts = $derived({
		spaces: envs.length,
		grows: activeGrows.length,
		plants: activeGrows.reduce((n, g) => n + (g.plantCount ?? 0), 0),
		devices: envs.reduce(
			(n, e) => n + (e.controls?.length ?? 0) + (e.sensors?.length ?? 0) + (e.cameras?.length ?? 0),
			0
		)
	});

	// A grower-facing status for one grow space.
	function envStatus(e: EnvironmentView): { label: string; tone: Tone } {
		if (e.health === 'offline') return { label: 'Sensors offline', tone: 'danger' };
		if (e.hasTemp && e.emergencyTempC > 0 && e.tempC >= e.emergencyTempC)
			return { label: 'Too hot — cooling hard', tone: 'danger' };
		if (e.health === 'stale') return { label: 'Reconnecting', tone: 'warn' };
		if (!e.hasTemp && !e.hasHum) return { label: 'No sensors yet', tone: 'muted' };
		return { label: 'Running normally', tone: 'good' };
	}

	const healthDot = (t: Tone) =>
		t === 'good' ? 'bg-leaf' : t === 'warn' ? 'bg-warn' : t === 'danger' ? 'bg-danger' : 'bg-rig-600';

	// --- The three pieces of the running system, in plain language. ---

	const corePiece = $derived(
		live.status === 'live'
			? { label: 'Online', tone: 'good' as Tone, detail: 'Keeping your climate on target' }
			: live.status === 'connecting'
				? { label: 'Connecting', tone: 'warn' as Tone, detail: 'Reaching your grow system…' }
				: { label: 'Offline', tone: 'danger' as Tone, detail: "Can't reach your grow system" }
	);

	const pendingUpdates = $derived.by(() => {
		const sup = ha?.supervisor;
		if (!sup?.available) return 0;
		const core = [sup.core, sup.os, sup.supervisor].filter((c) => c?.updateAvailable).length;
		const addons = sup.addons?.filter((a) => a.updateAvailable).length ?? 0;
		return core + addons;
	});

	const haPiece = $derived.by<{ label: string; tone: Tone; detail: string }>(() => {
		if (!ha) return { label: 'Checking…', tone: 'muted', detail: '' };
		if (ha.adapter !== 'homeassistant')
			return {
				label: 'Simulator mode',
				tone: 'muted',
				detail: 'Running on the built-in simulator — no hardware connected'
			};
		if (ha.health === 'offline')
			return { label: 'Disconnected', tone: 'danger', detail: 'Home Assistant is not responding' };
		if (ha.health === 'stale')
			return { label: 'Unstable', tone: 'warn', detail: 'Home Assistant connection is dropping in and out' };
		if (pendingUpdates > 0)
			return {
				label: 'Connected',
				tone: 'warn',
				detail: `${pendingUpdates} update${pendingUpdates === 1 ? '' : 's'} available`
			};
		return { label: 'Connected', tone: 'good', detail: 'Connected and up to date' };
	});

	const activeInstances = $derived(instances.filter((i) => i.enabled));
	const serviceErrors = $derived(activeInstances.filter((i) => i.status === 'error').length);
	const servicesPiece = $derived.by<{ label: string; tone: Tone; detail: string }>(() => {
		if (activeInstances.length === 0)
			return { label: 'None', tone: 'muted', detail: 'No external services connected' };
		if (serviceErrors > 0)
			return {
				label: `${serviceErrors} need attention`,
				tone: 'warn',
				detail: `${activeInstances.length} connected · ${serviceErrors} reporting a problem`
			};
		return {
			label: 'All healthy',
			tone: 'good',
			detail: `${activeInstances.length} service${activeInstances.length === 1 ? '' : 's'} connected`
		};
	});

	// --- Anything that wants a look, gathered into one attention list. ---

	interface Issue {
		tone: Exclude<Tone, 'good' | 'muted'>;
		text: string;
		href?: string;
	}
	const issues = $derived.by<Issue[]>(() => {
		const out: Issue[] = [];
		if (live.status === 'offline')
			out.push({ tone: 'danger', text: "Can't reach your grow system — the control engine is offline." });
		for (const e of envs) {
			const s = envStatus(e);
			if (s.tone === 'danger') out.push({ tone: 'danger', text: `${e.name}: ${s.label.toLowerCase()}.`, href: `/env/${e.id}` });
			else if (s.tone === 'warn') out.push({ tone: 'warn', text: `${e.name}: ${s.label.toLowerCase()}.`, href: `/env/${e.id}` });
		}
		if (ha?.adapter === 'homeassistant' && ha.health === 'offline')
			out.push({ tone: 'danger', text: 'Home Assistant is disconnected.', href: '/admin/home-assistant' });
		else if (ha?.adapter === 'homeassistant' && ha.health === 'stale')
			out.push({ tone: 'warn', text: 'Home Assistant connection is unstable.', href: '/admin/home-assistant' });
		if (pendingUpdates > 0)
			out.push({
				tone: 'warn',
				text: `${pendingUpdates} Home Assistant update${pendingUpdates === 1 ? '' : 's'} available.`,
				href: '/admin/home-assistant'
			});
		if (serviceErrors > 0)
			out.push({ tone: 'warn', text: `${serviceErrors} connected service${serviceErrors === 1 ? '' : 's'} reporting a problem.`, href: '/admin/integrations' });
		return out;
	});

	// Overall system health drives the hero banner.
	const overall = $derived.by<'good' | 'warn' | 'danger'>(() => {
		if (issues.some((i) => i.tone === 'danger')) return 'danger';
		if (issues.some((i) => i.tone === 'warn')) return 'warn';
		return 'good';
	});

	const hero = {
		good: {
			icon: CircleCheck,
			title: 'Everything is running smoothly',
			body: 'Your grow system is online and all grow spaces are on track.',
			ring: 'border-leaf/30 bg-leaf/10',
			fg: 'text-leaf'
		},
		warn: {
			icon: TriangleAlert,
			title: 'A few things could use a look',
			body: 'Your grow system is running, but some items below want your attention.',
			ring: 'border-warn/40 bg-warn/10',
			fg: 'text-warn'
		},
		danger: {
			icon: CircleX,
			title: 'Something needs your attention',
			body: 'Part of your grow system needs a hand. See what to check below.',
			ring: 'border-danger/40 bg-danger/10',
			fg: 'text-danger'
		}
	} as const;

	function fmtAge(ms: number): string {
		if (ms < 2000) return 'just now';
		const s = Math.floor(ms / 1000);
		if (s < 60) return `${s}s ago`;
		const m = Math.floor(s / 60);
		return `${m}m ago`;
	}
	const updatedAge = $derived(live.lastMessageAt ? fmtAge(now - live.lastMessageAt) : null);
</script>

{#snippet piece(icon: typeof Activity, title: string, p: { label: string; tone: Tone; detail: string }, href?: string)}
	{@const Icon = icon}
	<svelte:element
		this={href ? 'a' : 'div'}
		{...href ? { href } : {}}
		class="group flex items-center gap-4 px-5 py-4 {href ? 'transition-colors hover:bg-rig-800/30' : ''}"
	>
		<span class="grid h-10 w-10 shrink-0 place-items-center rounded-lg bg-rig-800/70 text-rig-300">
			<Icon size={19} />
		</span>
		<div class="min-w-0 flex-1">
			<div class="flex items-center gap-2">
				<span class="h-2 w-2 shrink-0 rounded-full {healthDot(p.tone)}"></span>
				<span class="font-medium">{title}</span>
			</div>
			<p class="truncate text-sm text-rig-400">{p.detail || '—'}</p>
		</div>
		<div class="flex shrink-0 items-center gap-1.5">
			<span class="text-sm font-medium {p.tone === 'good' ? 'text-leaf' : p.tone === 'warn' ? 'text-warn' : p.tone === 'danger' ? 'text-danger' : 'text-rig-400'}">
				{p.label}
			</span>
			{#if href}
				<ChevronRight size={16} class="text-rig-600 transition-transform group-hover:translate-x-0.5" />
			{/if}
		</div>
	</svelte:element>
{/snippet}

<div class="space-y-6">
	<div class="flex items-center justify-between gap-3">
		<h2 class="text-lg font-semibold">System status</h2>
		{#if updatedAge}
			<span class="text-xs text-rig-500">Updated {updatedAge}</span>
		{/if}
	</div>

	{#if !snap && !loaded}
		<p class="text-sm text-rig-400">Loading your system overview…</p>
	{:else}
		{@const h = hero[overall]}
		{@const HeroIcon = h.icon}
		<!-- Hero: the one-glance answer to "is my grow OK?" -->
		<div class="flex items-start gap-4 rounded-2xl border {h.ring} p-5">
			<span class="mt-0.5 shrink-0 {h.fg}"><HeroIcon size={28} /></span>
			<div>
				<h3 class="text-lg font-semibold {h.fg}">{h.title}</h3>
				<p class="mt-1 text-sm text-rig-300">{h.body}</p>
			</div>
		</div>

		<!-- At-a-glance counts -->
		<section class="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
			<StatTile label="Grow spaces" value={String(counts.spaces)} tone="good" />
			<StatTile label="Active grows" value={String(counts.grows)} />
			<StatTile label="Plants" value={String(counts.plants)} />
			<StatTile label="Devices" value={String(counts.devices)} />
		</section>

		<!-- Attention list — only shown when there's something to say. -->
		{#if issues.length}
			<section class="overflow-hidden rounded-xl border border-rig-800 bg-rig-900/40">
				<h3 class="border-b border-rig-800 px-5 py-3 text-sm font-semibold uppercase tracking-wide text-rig-400">
					Needs attention
				</h3>
				<ul class="divide-y divide-rig-800/70">
					{#each issues as issue (issue.text)}
						<li>
							<svelte:element
								this={issue.href ? 'a' : 'div'}
								{...issue.href ? { href: issue.href } : {}}
								class="group flex items-center gap-3 px-5 py-3 text-sm {issue.href ? 'transition-colors hover:bg-rig-800/30' : ''}"
							>
								<span class="h-2 w-2 shrink-0 rounded-full {issue.tone === 'danger' ? 'bg-danger' : 'bg-warn'}"></span>
								<span class="flex-1 text-rig-200">{issue.text}</span>
								{#if issue.href}
									<ChevronRight size={16} class="text-rig-600 transition-transform group-hover:translate-x-0.5" />
								{/if}
							</svelte:element>
						</li>
					{/each}
				</ul>
			</section>
		{/if}

		<!-- The running system, piece by piece -->
		<section class="overflow-hidden rounded-xl border border-rig-800 bg-rig-900/40">
			<h3 class="border-b border-rig-800 px-5 py-3 text-sm font-semibold uppercase tracking-wide text-rig-400">
				Your system
			</h3>
			<div class="divide-y divide-rig-800/70">
				{@render piece(Activity, 'Control engine', corePiece)}
				{@render piece(HousePlug, 'Home Assistant', haPiece, '/admin/home-assistant')}
				{@render piece(Blocks, 'Connected services', servicesPiece, '/admin/integrations')}
			</div>
		</section>

		<!-- Per grow-space health -->
		<section class="overflow-hidden rounded-xl border border-rig-800 bg-rig-900/40">
			<h3 class="border-b border-rig-800 px-5 py-3 text-sm font-semibold uppercase tracking-wide text-rig-400">
				Grow spaces
			</h3>
			{#if envs.length === 0}
				<div class="flex flex-col items-center gap-2 px-5 py-8 text-center">
					<Sprout size={28} class="text-rig-600" />
					<p class="text-sm text-rig-400">No grow spaces yet.</p>
					<a href="/wizard/box" class="text-sm text-leaf hover:underline">Set up a grow box</a>
				</div>
			{:else}
				<ul class="divide-y divide-rig-800/70">
					{#each envs as e (e.id)}
						{@const s = envStatus(e)}
						<li>
							<a href="/env/{e.id}" class="group flex items-center gap-3 px-5 py-3 transition-colors hover:bg-rig-800/30">
								<span class="h-2 w-2 shrink-0 rounded-full {healthDot(s.tone)}"></span>
								<span class="flex-1 truncate font-medium">{e.name}</span>
								{#if e.hasTemp || e.hasHum}
									<span class="hidden text-sm tabular-nums text-rig-400 sm:block">
										{#if e.hasTemp}{e.tempC.toFixed(1)}°C{/if}{#if e.hasTemp && e.hasHum} · {/if}{#if e.hasHum}{e.humidity.toFixed(0)}%{/if}
									</span>
								{/if}
								<span class="w-36 text-right text-sm {s.tone === 'good' ? 'text-leaf' : s.tone === 'warn' ? 'text-warn' : s.tone === 'danger' ? 'text-danger' : 'text-rig-500'}">
									{s.label}
								</span>
								<ChevronRight size={16} class="text-rig-600 transition-transform group-hover:translate-x-0.5" />
							</a>
						</li>
					{/each}
				</ul>
			{/if}
		</section>
	{/if}
</div>
