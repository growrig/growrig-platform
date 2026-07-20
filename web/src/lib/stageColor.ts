// Stage colors — one source of truth so the timeline heatmap, the Plan → Stages
// list and anything else that colors stages stay consistent.
//
// Colors are assigned by a stage's position in the grow's sequence (not its
// name), so every surface that iterates a grow's `stages` in order agrees.

/** Phase palette, indexed by a stage's position in the grow's sequence. */
export const STAGE_PALETTE = ['#4ade80', '#38bdf8', '#a78bfa', '#f97316', '#f472b6', '#facc15'];

/** Color for the stage at the given position in a grow's sequence. */
export const stageColorAt = (index: number): string =>
	STAGE_PALETTE[((index % STAGE_PALETTE.length) + STAGE_PALETTE.length) % STAGE_PALETTE.length];
