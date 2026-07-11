// Client-side mirror of Grow Core's photoperiod logic (growcore/internal/domain
// LightSchedule). Used to draw light-ON bands on the timeline — including the
// projected future, since the schedule is deterministic from wall-clock time.
import type { LightSchedule, Phase, PhotoperiodDefaults } from './types';

export interface Interval {
	start: number; // epoch ms
	end: number; // epoch ms
}

const DAY = 86_400_000;

function clampHours(h: number): number {
	return Math.max(0, Math.min(24, h));
}

function parseHHMM(s: string | undefined): number | null {
	const m = /^(\d{1,2}):(\d{2})$/.exec((s ?? '').trim());
	if (!m) return null;
	const h = +m[1];
	const mm = +m[2];
	if (h < 0 || h > 23 || mm < 0 || mm > 59) return null;
	return h * 60 + mm;
}

/** Hours of light for the given phase under this schedule. */
export function effectiveOnHours(
	sched: LightSchedule,
	phase: Phase,
	defaults: PhotoperiodDefaults
): number {
	if (sched.mode === 'custom') return clampHours(sched.onHours);
	const override = sched.phaseOnHours?.[phase];
	if (override != null) return clampHours(override);
	return defaults[phase] ?? 18;
}

export interface Transition {
	at: number; // epoch ms of the next boundary
	on: boolean; // true if the light turns ON at that boundary, false if OFF
}

/** The next light on/off boundary after `now`, or null when the schedule never
 *  flips (mode off, or an always-on / always-off duration). */
export function nextTransition(
	sched: LightSchedule | undefined,
	phase: Phase,
	defaults: PhotoperiodDefaults,
	now: number
): Transition | null {
	if (!sched || sched.mode === 'off') return null;
	const hours = effectiveOnHours(sched, phase, defaults);
	if (hours <= 0 || hours >= 24) return null;
	const onAt = parseHHMM(sched.lightsOnAt);
	if (onAt == null) return null;
	const offAt = (onAt + hours * 60) % 1440;

	const midnight = new Date(now);
	midnight.setHours(0, 0, 0, 0);
	const base = midnight.getTime();
	const occur = (minute: number) => {
		const cand = base + minute * 60_000;
		return cand <= now ? cand + DAY : cand;
	};
	const onNext = occur(onAt);
	const offNext = occur(offAt);
	return onNext < offNext ? { at: onNext, on: true } : { at: offNext, on: false };
}

/** Light-ON intervals intersecting [start, end], per the current schedule. */
export function lightIntervals(
	sched: LightSchedule | undefined,
	phase: Phase,
	defaults: PhotoperiodDefaults,
	start: number,
	end: number
): Interval[] {
	if (!sched || sched.mode === 'off') return [];
	const hours = effectiveOnHours(sched, phase, defaults);
	if (hours <= 0) return [];
	if (hours >= 24) return [{ start, end }];
	const onAt = parseHHMM(sched.lightsOnAt);
	if (onAt == null) return [];
	const durMs = hours * 3_600_000;

	const out: Interval[] = [];
	// Walk local-midnight days across the window (padded so a wrapped window at
	// the edges is still captured).
	const first = new Date(start);
	first.setHours(0, 0, 0, 0);
	for (let day = first.getTime() - DAY; day <= end + DAY; day += DAY) {
		const onStart = day + onAt * 60_000;
		const s = Math.max(onStart, start);
		const e = Math.min(onStart + durMs, end);
		if (e > s) out.push({ start: s, end: e });
	}
	return out;
}
