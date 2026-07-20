<script lang="ts">
	import type { CareEvent, GrowDetail, StageDuration } from '$lib/types';
	import { Tooltip } from 'bits-ui';
	import { careVisual } from '$lib/care';
	import { titleCase } from '$lib/format';
	import { fmtDate } from '$lib/datetime';
	import { projectStages } from '$lib/growTimeline';
	import { STAGE_PALETTE } from '$lib/stageColor';

	interface Props {
		grow: GrowDetail;
		events: CareEvent[];
		stages?: StageDuration[]; // actual stage history (from analytics)
	}
	let { grow, events, stages = [] }: Props = $props();

	const DAY = 86_400_000;
	const startOfDay = (d: Date | string | number) => {
		const x = new Date(d);
		x.setHours(0, 0, 0, 0);
		return x;
	};
	const addDays = (d: Date, n: number) => new Date(d.getTime() + n * DAY);
	// Monday-based weekday index: 0 = Mon … 6 = Sun.
	const weekday = (d: Date) => (d.getDay() + 6) % 7;

	// Phase palette, indexed by a stage's position in the grow's sequence.
	const palette = STAGE_PALETTE;

	interface Cell {
		date: Date;
		day: number; // 1-based day of the grow (day 1 = start date)
		inRange: boolean;
		future: boolean;
		today: boolean;
		count: number;
		breakdown: { label: string; n: number }[];
		stage?: string; // stage this day falls in (recorded or projected)
		stageColor?: string;
		stagePredicted: boolean; // the stage assignment is a projection
	}

	const model = $derived.by(() => {
		const start = startOfDay(grow.startedAt);
		const today = startOfDay(new Date());
		const daysSinceStart = Math.floor((today.getTime() - start.getTime()) / DAY) + 1;

		// Project the grow's stages (recorded + predicted) onto the timeline, then
		// look up which stage owns any given day.
		const segments = projectStages(grow, stages);
		const segFor = (t: number) => segments.find((s) => t >= s.start.getTime() && t < s.end.getTime());

		// The grid spans to whichever is later: the projected finish of the shown
		// phases, or today — never shorter than a week.
		const projectedEnd = segments.length ? segments[segments.length - 1].end : start;
		const projectedDays = Math.round((projectedEnd.getTime() - start.getTime()) / DAY);
		const span = Math.max(projectedDays, daysSinceStart, 7);
		const projected = span > daysSinceStart;
		const end = addDays(start, span - 1);

		// Snap the grid to whole Monday→Sunday weeks around the range.
		const gridStart = addDays(start, -weekday(start));
		const gridEnd = addDays(end, 6 - weekday(end));
		const weeks = Math.round((gridEnd.getTime() - gridStart.getTime()) / DAY + 1) / 7;

		// Care actions bucketed by local calendar day.
		const byDay = new Map<number, CareEvent[]>();
		for (const e of events) {
			const key = startOfDay(e.occurredAt).getTime();
			(byDay.get(key) ?? byDay.set(key, []).get(key)!).push(e);
		}

		const cells: Cell[] = [];
		const monthLabels: { week: number; label: string }[] = [];
		let lastMonth = -1;
		for (let w = 0; w < weeks; w++) {
			const wkStart = addDays(gridStart, w * 7);
			if (wkStart.getMonth() !== lastMonth && wkStart.getTime() >= start.getTime()) {
				lastMonth = wkStart.getMonth();
				monthLabels.push({ week: w, label: wkStart.toLocaleDateString(undefined, { month: 'short' }) });
			}
			for (let d = 0; d < 7; d++) {
				const date = addDays(gridStart, w * 7 + d);
				const t = date.getTime();
				const evs = byDay.get(t) ?? [];
				const counts = new Map<string, number>();
				for (const e of evs) {
					const label = careVisual(e.type).label;
					counts.set(label, (counts.get(label) ?? 0) + 1);
				}
				const seg = t >= start.getTime() && t <= end.getTime() ? segFor(t) : undefined;
				cells.push({
					date,
					day: Math.round((t - start.getTime()) / DAY) + 1,
					inRange: t >= start.getTime() && t <= end.getTime(),
					future: t > today.getTime(),
					today: t === today.getTime(),
					count: evs.length,
					breakdown: [...counts].map(([label, n]) => ({ label, n })),
					stage: seg ? titleCase(seg.stage) : undefined,
					stageColor: seg ? palette[seg.index % palette.length] : undefined,
					stagePredicted: seg ? seg.status === 'upcoming' : false
				});
			}
		}

		return { cells, weeks, monthLabels, start, end, total: events.length, projected };
	});

	// Two channels: the background is the day's stage colour (so phases read as
	// bands, fainter for future days), and the only border drawn is the care
	// signal — a leaf ring that strengthens with the number of actions. Today
	// keeps a full leaf ring as its marker; no other borders are drawn.
	// Days with no identified stage fall back to silver (rather than the base grey),
	// so an unclassified span reads as deliberate, not empty.
	const UNIDENTIFIED = '#9ca3af';
	const stageBg = (c: Cell, pct: number) =>
		`color-mix(in srgb, ${c.stageColor ?? UNIDENTIFIED} ${pct}%, var(--color-rig-800))`;
	const careBorder = [0, 55, 72, 88, 100]; // leaf strength (%) by action count
	const cellStyle = (c: Cell) => {
		if (!c.inRange) return 'background:transparent;border-color:transparent';
		// Elapsed days show the stage colour at full strength; future days are dimmed.
		if (c.future) return `background:${stageBg(c, 28)};border-color:transparent`;
		const border = c.today
			? 'var(--color-danger)'
			: c.count > 0
				? `color-mix(in srgb, var(--leaf) ${careBorder[Math.min(c.count, 4)]}%, transparent)`
				: 'transparent';
		return `${c.today ? 'border-width:1.5px;' : ''}background:${stageBg(c, 100)};border-color:${border}`;
	};

	const dayLabel = (c: Cell) =>
		`${fmtDate(c.date, { weekday: 'short', day: 'numeric', month: 'numeric' })} (day ${c.day})`;

	const subtitle = $derived(
		`${model.total} action${model.total === 1 ? '' : 's'} since ${fmtDate(model.start, { month: 'short', day: 'numeric' })}` +
			(model.projected ? ` · est. finish ${fmtDate(model.end, { month: 'short', day: 'numeric' })}` : '')
	);

	const legendSteps = [0, 55, 72, 88, 100]; // leaf border strengths (matches careBorder)
</script>

<section>
	<div class="mb-3 flex items-baseline justify-between gap-3">
		<h2 class="text-sm font-semibold uppercase tracking-wide text-rig-400">Grow Timeline</h2>
		<span class="text-xs text-rig-500">{subtitle}</span>
	</div>

	<div class="rounded-xl border border-rig-800 bg-rig-950/40 p-4" style="--weeks:{model.weeks}">
		<Tooltip.Provider delayDuration={80} disableHoverableContent>
			<!-- Month labels -->
			<div class="heat-cols mb-1 h-[0.9rem] text-[10px] leading-none text-rig-500">
				{#each model.monthLabels as m (m.week)}
					<span style="grid-column:{m.week + 1}">{m.label}</span>
				{/each}
			</div>

			<!-- Day grid: 7 rows (Mon→Sun), one column per week. Cells are bordered by stage. -->
			<div class="heat-grid">
				{#each model.cells as c (c.date.getTime())}
					{#if !c.inRange}
						<span class="heat-cell" style={cellStyle(c)}></span>
					{:else}
						<Tooltip.Root>
							<Tooltip.Trigger class="heat-cell heat-trigger" style={cellStyle(c)} aria-label={dayLabel(c)} />
							<Tooltip.Portal>
								<Tooltip.Content sideOffset={6} class="z-50 rounded-md border border-rig-700 bg-rig-900 px-2.5 py-1.5 text-xs shadow-xl">
									<div class="font-medium text-rig-100">
										{dayLabel(c)}{#if c.today}<span class="text-leaf"> · today</span>{/if}
									</div>
									{#if c.stage}
										<div class="mt-0.5 font-medium" style="color:{c.stageColor}">
											{c.stage}{#if c.stagePredicted}<span class="font-normal text-rig-500"> · predicted</span>{/if}
										</div>
									{/if}
									{#if c.count > 0}
										<div class="mt-0.5 text-rig-400">{c.count} care action{c.count === 1 ? '' : 's'}</div>
										<ul class="mt-1 space-y-0.5 text-rig-300">
											{#each c.breakdown as b (b.label)}
												<li class="tabular-nums">{b.n} × {b.label}</li>
											{/each}
										</ul>
									{:else if c.future}
										<div class="mt-0.5 text-rig-500">Upcoming · estimated</div>
									{:else}
										<div class="mt-0.5 text-rig-500">No care logged</div>
									{/if}
								</Tooltip.Content>
							</Tooltip.Portal>
						</Tooltip.Root>
					{/if}
				{/each}
			</div>

			<div class="mt-3 flex items-center justify-end gap-1.5 text-[10px] text-rig-500">
				<span>Care</span>
				{#each legendSteps as pct (pct)}
					<span
						class="h-[11px] w-[11px] rounded-[3px] border"
						style="background:var(--color-rig-800);border-color:{pct === 0
							? 'var(--color-rig-700)'
							: `color-mix(in srgb, var(--leaf) ${pct}%, transparent)`}"
					></span>
				{/each}
				<span>More</span>
			</div>
		</Tooltip.Provider>
	</div>
</section>

<style>
	.heat-grid {
		display: grid;
		grid-template-rows: repeat(7, auto);
		grid-auto-flow: column;
		grid-auto-columns: 1fr;
		gap: 3px;
		width: 100%;
		max-width: calc(var(--weeks) * 1.55rem);
	}
	/* Cells include bits-ui <Tooltip.Trigger> instances, so these must pierce the
	   component boundary (:global) while staying scoped under .heat-grid. */
	.heat-grid :global(.heat-cell) {
		aspect-ratio: 1 / 1;
		border-radius: 3px;
		border-width: 1px;
		border-style: solid;
	}
	.heat-grid :global(.heat-trigger) {
		padding: 0;
		appearance: none;
		cursor: pointer;
		transition: filter 0.1s ease;
	}
	.heat-grid :global(.heat-trigger:hover) {
		filter: brightness(1.25);
	}
	.heat-grid :global(.heat-trigger:focus-visible) {
		outline: 2px solid var(--leaf);
		outline-offset: 1px;
	}
	.heat-cols {
		display: grid;
		grid-template-columns: repeat(var(--weeks), 1fr);
		width: 100%;
		max-width: calc(var(--weeks) * 1.55rem);
	}
	.heat-cols > span {
		grid-row: 1;
	}
</style>
