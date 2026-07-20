// Shared trigger for the universal "New environment" modal, so any surface (the
// header quick-add, the home dashboard) can open it — optionally pre-seeded with
// a kind, a parent air source, or a location.
import type { EnvironmentKind } from './types';

class AddEnvState {
	open = $state(false);
	kind = $state<EnvironmentKind>('tent');
	airSourceId = $state('');
	locationId = $state('');

	start(opts: { kind?: EnvironmentKind; airSourceId?: string; locationId?: string } = {}) {
		this.kind = opts.kind ?? 'tent';
		this.airSourceId = opts.airSourceId ?? '';
		this.locationId = opts.locationId ?? '';
		this.open = true;
	}
}

export const addEnv = new AddEnvState();
