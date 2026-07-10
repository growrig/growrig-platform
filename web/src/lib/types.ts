// Mirrors the JSON emitted by Grow Core (growcore/internal/domain).

export type Role = 'unassigned' | 'exhaust' | 'intake' | 'circulation';
export type EnvironmentKind = 'tent' | 'room';
export type BindingKind = 'sensor' | 'fan' | 'light' | 'camera';
export type Measurement = 'temperature' | 'humidity' | 'co2';
export type Health = 'online' | 'stale' | 'offline';
export type Phase = 'seedling' | 'vegetative' | 'flowering' | 'flush' | 'drying' | 'cure';
export type Category = 'tent' | 'fan' | 'light' | 'sensor' | 'camera' | 'plug' | 'combo';

export interface Cycle {
	environmentId: string;
	strain: string;
	startedAt: string;
	phase: Phase;
	phaseStarted: string;
	notes: string;
}

export interface BindingTemplate {
	label: string;
	kind: BindingKind;
	measurement?: Measurement;
	role?: Role;
	entityDomain: string;
	deviceClass?: string;
	wattage?: number;
}

export interface CatalogProduct {
	id: string;
	brand: string;
	model: string;
	category: Category;
	connection: string;
	description?: string;
	provides?: BindingTemplate[];
}

export interface Environment {
	id: string;
	name: string;
	kind: EnvironmentKind;
	airSourceId: string;
	model: string;
	widthCm: number;
	depthCm: number;
	heightCm: number;
	targetTempC: number;
	targetHumidity: number;
	targetCO2: number;
	emergencyTempC: number;
}

export interface Binding {
	id: string;
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

export interface DiscoveredEntity {
	entity: string;
	name: string;
	kind: BindingKind;
	measurement?: Measurement;
}

export interface SensorReading {
	id: string;
	name: string;
	measurement: Measurement;
	entity: string;
	value: number;
	ok: boolean;
}

export interface ControlState {
	id: string;
	name: string;
	kind: BindingKind;
	role?: Role;
	entity: string;
	desiredSpeed: number;
	rpm: number;
	on: boolean;
	wattage?: number;
	primary?: boolean;
}

export interface CameraRef {
	id: string;
	name: string;
	entity: string;
}

export interface AirSourceView {
	id: string;
	name: string;
	tempC: number;
	humidity: number;
	vpd: number;
	ok: boolean;
}

export interface EnvironmentView extends Environment {
	health: Health;
	hasClimate: boolean;
	hasTemp: boolean;
	hasHum: boolean;
	tempC: number;
	humidity: number;
	co2: number;
	hasCO2: boolean;
	vpd: number;
	sensors: SensorReading[];
	controls: ControlState[];
	cameras: CameraRef[];
	airSource?: AirSourceView;
	cycle?: Cycle;
}

export interface Snapshot {
	time: string;
	environments: EnvironmentView[];
}

export interface Reading {
	environmentId: string;
	time: string;
	tempC: number;
	humidity: number;
	co2: number;
	vpd: number;
	exhaustSpeed: number;
}

export interface Info {
	adapter: string;
}
