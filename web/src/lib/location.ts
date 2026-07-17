import type { Environment, EnvironmentView, Location } from '$lib/types';

/**
 * Resolve the location an environment sits at, inheriting from its air-source
 * room when the environment itself has none set. A grow box (tent) that is fed
 * by a room with a location automatically adopts that location — for local
 * weather and dashboard grouping — without the user re-entering it.
 *
 * Returns '' when neither the environment nor its air source is sited.
 */
export function resolveLocationId(
	env: Pick<Environment, 'locationId' | 'airSourceId'> | undefined,
	environments: Pick<Environment, 'id' | 'locationId'>[]
): string {
	if (!env) return '';
	if (env.locationId) return env.locationId;
	if (env.airSourceId) {
		const source = environments.find((e) => e.id === env.airSourceId);
		if (source?.locationId) return source.locationId;
	}
	return '';
}

/** One room and the grow boxes (tents) it feeds air to. */
export interface EnvTreeRoom {
	room: EnvironmentView;
	boxes: EnvironmentView[];
}

/** A sited (or "no location") group of rooms and loose tents. */
export interface EnvTreeLocation {
	key: string;
	name: string;
	located: boolean;
	rooms: EnvTreeRoom[];
	looseBoxes: EnvironmentView[];
}

/**
 * Build the Location → Room → Tent hierarchy from the live snapshot's
 * environments and the known locations. A tent's air source is its room; a
 * room's location is its site; tents with no room (or a non-room air source)
 * fall back to a location's "loose" list. Environments that can't be placed
 * under any location land in a trailing "No location" group.
 *
 * Shared by the Home page and the navigation menu so both render the same tree.
 */
export function buildEnvTree(
	environments: EnvironmentView[],
	locations: Location[]
): EnvTreeLocation[] {
	const byId = new Map(environments.map((e) => [e.id, e]));
	const rooms = environments.filter((e) => e.kind === 'room');
	const tents = environments.filter((e) => e.kind === 'tent');
	const roomOf = (t: EnvironmentView) => {
		const src = t.airSourceId ? byId.get(t.airSourceId) : undefined;
		return src && src.kind === 'room' ? src : undefined;
	};
	const locOf = (e: EnvironmentView) => resolveLocationId(e, environments);

	const out: EnvTreeLocation[] = [];
	const placed = new Set<string>();

	for (const loc of [...locations].sort((a, b) => a.name.localeCompare(b.name))) {
		const roomNodes: EnvTreeRoom[] = rooms
			.filter((r) => locOf(r) === loc.id)
			.map((room) => ({ room, boxes: tents.filter((t) => roomOf(t)?.id === room.id) }));
		const looseBoxes = tents.filter(
			(t) => locOf(t) === loc.id && !roomNodes.some((n) => n.boxes.includes(t))
		);
		if (!roomNodes.length && !looseBoxes.length) continue;
		for (const n of roomNodes) {
			placed.add(n.room.id);
			n.boxes.forEach((b) => placed.add(b.id));
		}
		looseBoxes.forEach((b) => placed.add(b.id));
		out.push({ key: loc.id, name: loc.name, located: true, rooms: roomNodes, looseBoxes });
	}

	const orphans = environments.filter((e) => !placed.has(e.id));
	if (orphans.length) {
		const roomNodes: EnvTreeRoom[] = orphans
			.filter((e) => e.kind === 'room')
			.map((room) => ({
				room,
				boxes: orphans.filter((t) => t.kind === 'tent' && roomOf(t)?.id === room.id)
			}));
		const looseBoxes = orphans.filter(
			(e) => e.kind === 'tent' && !roomNodes.some((n) => n.boxes.includes(e))
		);
		out.push({ key: '__none__', name: 'No location', located: false, rooms: roomNodes, looseBoxes });
	}
	return out;
}
