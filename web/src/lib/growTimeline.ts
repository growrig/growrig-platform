// Shared stage-timeline projection for a grow.
//
// A grow moves through an ordered sequence of stages. Past and current stages
// have real recorded dates (from analytics stage history); stages not yet
// reached are projected forward from the species' per-stage typical durations
// (grow.stageEstimates). Both the Overview timeline ribbon and the Plan → Stages
// list render from this, so the predicted milestones stay consistent.

import type { GrowDetail, StageDuration } from './types';

const DAY = 86_400_000;

const startOfDay = (d: Date | string | number): Date => {
	const x = new Date(d);
	x.setHours(0, 0, 0, 0);
	return x;
};
const addDays = (d: Date, n: number): Date => new Date(d.getTime() + n * DAY);

export type StageStatus = 'done' | 'current' | 'upcoming';

export interface StageSegment {
	stage: string;
	index: number; // position in the grow's stage sequence
	start: Date;
	end: Date; // exclusive: the start of the next stage / projected finish
	days: number;
	status: StageStatus;
	/** The start date is a projection (this stage hasn't been entered yet). */
	startPredicted: boolean;
	/** The end date is a projection (stage still running or not yet reached). */
	endPredicted: boolean;
}

/**
 * Project a grow's stages onto a timeline. Recorded stages use their real
 * dates; the current stage's end and all later stages are projected from
 * per-stage typical durations, chained so each phase follows the previous one.
 */
export function projectStages(grow: GrowDetail, history: StageDuration[] = []): StageSegment[] {
	const start = startOfDay(grow.startedAt);
	const today = startOfDay(new Date());
	const estimates = grow.stageEstimates ?? {};

	const actual = new Map<string, { from: Date; to?: Date }>();
	for (const sd of history) {
		actual.set(sd.stage, { from: startOfDay(sd.from), to: sd.to ? startOfDay(sd.to) : undefined });
	}

	const currentIdx = grow.stages.indexOf(grow.stage);
	const segments: StageSegment[] = [];
	let cursor = start;
	grow.stages.forEach((stage, i) => {
		const a = actual.get(stage);
		// Skip past stages we have no record of — we can't claim they happened for
		// this grow (e.g. a grow started straight into veg has no seedling phase).
		if (i < currentIdx && !a) return;
		const segStart = a?.from ?? cursor;
		const est = estimates[stage] ?? 0;
		let segEnd: Date;
		let endPredicted: boolean;
		if (a?.to) {
			segEnd = a.to;
			endPredicted = false;
		} else if (i === currentIdx) {
			// Current stage in progress: project its end, but never before today.
			segEnd = est ? addDays(segStart, est) : today;
			if (segEnd.getTime() < today.getTime()) segEnd = today;
			endPredicted = true;
		} else {
			segEnd = addDays(segStart, est);
			endPredicted = true;
		}
		cursor = segEnd;
		segments.push({
			stage,
			index: i,
			start: segStart,
			end: segEnd,
			days: Math.max(0, Math.round((segEnd.getTime() - segStart.getTime()) / DAY)),
			status: i < currentIdx ? 'done' : i === currentIdx ? 'current' : 'upcoming',
			startPredicted: !a,
			endPredicted
		});
	});
	return segments;
}
