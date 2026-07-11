import type { Environment } from '$lib/types';

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
