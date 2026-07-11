// REST client for Grow Core. When the web app is served by Grow Core itself
// (embedded, single binary) the base is same-origin. For local development
// against a separately-running Grow Core, set VITE_GROWCORE_URL.
import type {
	Binding,
	Activity,
	BindingKind,
	CatalogProduct,
	Cycle,
	DiscoveredEntity,
	Environment,
	EnvironmentKind,
	Info,
	Measurement,
	Phase,
	Reading,
	Role,
	Snapshot
} from './types';

export const CORE_URL: string = import.meta.env.VITE_GROWCORE_URL?.replace(/\/$/, '') ?? '';

export function wsURL(): string {
	const base = CORE_URL || window.location.origin;
	const u = new URL(base);
	u.protocol = u.protocol === 'https:' ? 'wss:' : 'ws:';
	u.pathname = '/api/ws';
	return u.toString();
}

async function req(path: string, init?: RequestInit): Promise<Response> {
	const res = await fetch(`${CORE_URL}${path}`, {
		headers: { 'Content-Type': 'application/json' },
		...init
	});
	if (!res.ok) {
		let msg = `${res.status} ${res.statusText}`;
		try {
			const body = await res.json();
			if (body?.error) msg = body.error;
		} catch {
			/* non-JSON error body */
		}
		throw new Error(msg);
	}
	return res;
}

async function json<T>(path: string, init?: RequestInit): Promise<T> {
	return (await req(path, init)).json() as Promise<T>;
}

// --- info, catalog & discovery ---

export const getInfo = () => json<Info>('/api/info');
/** Current live snapshot over REST — used for the initial paint before the
 *  WebSocket feed takes over. */
export const getState = () => json<Snapshot>('/api/state');
export const getDiscovery = () => json<DiscoveredEntity[]>('/api/discovery');
export const getCatalog = () => json<CatalogProduct[]>('/api/catalog');
export const getPhases = () => json<Phase[]>('/api/phases');
export const loadDemo = () => req('/api/demo', { method: 'POST' });

// --- environments ---

export const getEnvironments = () => json<Environment[]>('/api/environments');

export interface EnvironmentInput {
	name: string;
	kind: EnvironmentKind;
	airSourceId: string;
	model?: string;
	widthCm?: number;
	depthCm?: number;
	heightCm?: number;
	targetTempC: number;
	targetHumidity: number;
	targetCO2: number;
	emergencyTempC: number;
}

export const createEnvironment = (env: EnvironmentInput) =>
	json<Environment>('/api/environments', { method: 'POST', body: JSON.stringify(env) });

export const updateEnvironment = (id: string, env: EnvironmentInput) =>
	json<Environment>(`/api/environments/${encodeURIComponent(id)}`, {
		method: 'PUT',
		body: JSON.stringify(env)
	});

export const deleteEnvironment = (id: string) =>
	req(`/api/environments/${encodeURIComponent(id)}`, { method: 'DELETE' });

export const getEnvironmentYAML = async (id: string) =>
	(await req(`/api/environments/${encodeURIComponent(id)}/config`)).text();

export const updateEnvironmentYAML = (id: string, yaml: string) =>
	req(`/api/environments/${encodeURIComponent(id)}/config`, {
		method: 'PUT',
		headers: { 'Content-Type': 'application/yaml' },
		body: yaml
	});

// --- bindings ---

export const getBindings = () => json<Binding[]>('/api/bindings');

export interface BindingInput {
	deviceId: string;
	deviceName: string;
	powerControllerId?: string;
	controllerChannelId?: string;
	environmentId: string;
	kind: BindingKind;
	name: string;
	entity: string;
	measurement?: Measurement;
	role?: Role;
	rpmEntity?: string;
	wattage?: number;
	primary?: boolean;
}

export const createBinding = (b: BindingInput) =>
	json<Binding>('/api/bindings', { method: 'POST', body: JSON.stringify(b) });

export const updateBinding = (id: string, b: BindingInput) =>
	json<Binding>(`/api/bindings/${encodeURIComponent(id)}`, {
		method: 'PUT',
		body: JSON.stringify(b)
	});

export const deleteBinding = (id: string) =>
	req(`/api/bindings/${encodeURIComponent(id)}`, { method: 'DELETE' });

export const setSwitch = (bindingId: string, on: boolean) =>
	req(`/api/bindings/${encodeURIComponent(bindingId)}/switch`, {
		method: 'PUT',
		body: JSON.stringify({ on })
	});

// --- cycles ---

export interface CycleInput {
	strain: string;
	startedAt: string; // YYYY-MM-DD or RFC3339
	phase: Phase;
	notes: string;
}

export const setCycle = (envID: string, c: CycleInput) =>
	json<Cycle>(`/api/environments/${encodeURIComponent(envID)}/cycle`, {
		method: 'PUT',
		body: JSON.stringify(c)
	});

export const clearCycle = (envID: string) =>
	req(`/api/environments/${encodeURIComponent(envID)}/cycle`, { method: 'DELETE' });

// --- history ---

export const history = (envID: string, limit = 120) =>
	json<Reading[]>(`/api/environments/${encodeURIComponent(envID)}/history?limit=${limit}`);

export const getActivity = (environmentId?: string, limit = 100) => {
	const params = new URLSearchParams({ limit: String(limit) });
	if (environmentId) params.set('environmentId', environmentId);
	return json<Activity[]>(`/api/activity?${params}`);
};
