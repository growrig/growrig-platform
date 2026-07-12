// Mirrors the JSON emitted by Grow Core (growcore/internal/domain).

export type Role = 'unassigned' | 'exhaust' | 'intake' | 'circulation';
export type EnvironmentKind = 'tent' | 'room';
export type BindingKind = 'sensor' | 'fan' | 'controller' | 'light' | 'power' | 'camera';
export type Measurement = 'temperature' | 'humidity' | 'co2' | 'power';
export type Health = 'online' | 'stale' | 'offline';
export type Category = 'tent' | 'controller' | 'fan' | 'light' | 'sensor' | 'camera' | 'plug' | 'combo';
export type FanType = 'pc' | 'inline' | 'other';
export type CameraType = 'mjpeg' | 'snapshot' | 'rtsp';

// --- Cultivation layer (grows, plant units, placements) ---

export type GrowStatus = 'active' | 'completed' | 'archived';
export type TrackingMode = 'individual' | 'group';
export type PlantStatus = 'active' | 'harvested' | 'removed' | 'archived';

export interface Grow {
	id: string;
	name: string;
	/** A predefined crop family; drives the stage sequence. */
	species: string;
	/** Current stage name (one of `stages`). */
	stage: string;
	/** Ordered stage sequence, derived from `species`. */
	stages: string[];
	startedAt: string;
	stageStarted: string;
	status: GrowStatus;
	notes: string;
}

export interface PlantUnit {
	id: string;
	growId: string;
	label: string;
	/** Cultivar is per-unit, so one grow can mix cultivars. */
	cultivar: string;
	tracking: TrackingMode;
	quantity: number;
	status: PlantStatus;
	createdAt: string;
}

export interface PlantPlacement {
	id: string;
	plantUnitId: string;
	environmentId: string;
	startedAt: string;
	endedAt?: string; // absent = current
	position?: string;
}

export interface GrowEnvRef {
	id: string;
	name: string;
}

/** Count of active plants of one cultivar within a grow (for card thumbnails). */
export interface GrowCultivarRef {
	cultivar: string;
	count: number;
}

/** Compact live view of an environment's control grow. */
export interface GrowSummary {
	id: string;
	name: string;
	species: string;
	stage: string;
	stageDays: number;
	totalDays: number;
	plantCount: number;
}

/** Dashboard "Active Grows" view: a grow plus derived counts and locations. */
export interface GrowView extends Grow {
	stageDays: number;
	totalDays: number;
	plantCount: number;
	environments: GrowEnvRef[];
	cultivars: GrowCultivarRef[];
}

export interface PlacementView extends PlantPlacement {
	environmentName: string;
}

export type PotUnit = 'L' | 'gal';

/** One pot a plant lived in for a span of time (repot history). */
export interface PlantPot {
	id: string;
	plantUnitId: string;
	size: number;
	unit: PotUnit;
	type?: string;
	startedAt: string;
	endedAt?: string; // absent = current pot
}

export interface PlantDetail extends PlantUnit {
	currentEnvironmentId: string;
	currentEnvironmentName: string;
	placements: PlacementView[];
	currentPot?: PlantPot;
	pots: PlantPot[];
}

export interface PlantView extends PlantUnit {
	growName: string;
	currentEnvironmentId: string;
	currentEnvironmentName: string;
	placements: PlacementView[];
	currentPot?: PlantPot;
	pots: PlantPot[];
}

export interface GrowDetail extends Grow {
	stageDays: number;
	totalDays: number;
	plantCount: number;
	plants: PlantDetail[];
}

/** Current occupants of an environment, grouped by grow. */
export interface EnvPlantsGroup {
	grow: Grow;
	units: PlantUnit[];
}

/** Built-in editable stage sequences per crop family (GET /api/stage-presets). */
export type StagePresets = Record<string, string[]>;

// --- Species catalog & cultivars (YAML-defined; GET /api/species) ---

export type AttrType = 'text' | 'number' | 'percent' | 'enum';

/** One species-specific cultivar field declared in a species' YAML. */
export interface SpeciesAttribute {
	key: string;
	label: string;
	type: AttrType;
	options?: string[];
	unit?: string;
}

export interface SpeciesStage {
	name: string;
	lightHours: number;
}

/** A crop family: its ordered stages and cultivar attribute schema. */
export interface Species {
	id: string;
	label: string;
	stages: SpeciesStage[];
	cultivarAttributes?: SpeciesAttribute[];
}

/** A user-defined strain/variety within a species. */
export interface Cultivar {
	id: string;
	species: string;
	name: string;
	creator: string;
	description: string;
	/** Species-specific values keyed by the species' attribute keys. */
	attributes: Record<string, string>;
	/** MIME type of the stored image, or absent when there is none. */
	imageType?: string;
	createdAt: string;
}

// --- Feeding presets (nutrient schedules; GET /api/feedings) ---

/** One nutrient line in a schedule. `unit` overrides the preset default. */
export interface FeedingProduct {
	key: string;
	label: string;
	unit?: string;
}

/** One week of dosing: product key -> amount in that product's unit. */
export interface FeedingWeek {
	doses: Record<string, number>;
}

/** A named span of the schedule; `stage` optionally links it to a species stage. */
export interface FeedingPhase {
	name: string;
	stage?: string;
	weeks: FeedingWeek[];
}

/**
 * A nutrient feeding schedule: products dosed per week across phases. Built-in
 * presets (`source: 'builtin'`) come from species/<id>/feedings.yaml and are
 * read-only; user presets (`source: 'user'`) are created in-app.
 */
export interface FeedingPreset {
	id: string;
	species: string;
	name: string;
	brand: string;
	description: string;
	source: 'builtin' | 'user';
	/** Default dose unit, e.g. "ml/L". */
	unit: string;
	products: FeedingProduct[];
	phases: FeedingPhase[];
	createdAt: string;
}

export type LightScheduleMode = 'off' | 'phase' | 'custom';

export interface LightSchedule {
	environmentId: string;
	mode: LightScheduleMode;
	/** Local "HH:MM" the light comes on. */
	lightsOnAt: string;
	/** On-duration used in custom mode. */
	onHours: number;
	/** Per-stage on-hour overrides for phase mode; stages absent use defaults. */
	stageOnHours: Record<string, number>;
}

/** Recommended hours of light per known stage (GET /api/lighting/defaults). */
export type StageLightDefaults = Record<string, number>;

export interface BindingTemplate {
	label: string;
	kind: BindingKind;
	measurement?: Measurement;
	role?: Role;
	entityDomain: string;
	deviceClass?: string;
	wattage?: number;
	rpmEntityDomain?: string;
}

/** A concrete product supported by a driver. `specs` is a free-form numeric map
 *  (fans: sizeMm/maxRpm/airflowCfm/…; tents: widthCm/depthCm/heightCm). */
export interface ProductVariant {
	id: string;
	brand?: string;
	vendor?: string;
	group?: string;
	model?: string;
	image?: string;
	images?: { src: string; model?: string }[];
	description?: string;
	specs?: Record<string, number>;
	models?: ProductVariant[];
}

export interface CatalogProduct {
	id: string;
	brand: string;
	vendor?: string;
	model: string;
	image?: string;
	category: Category;
	connection: string;
	description?: string;
	version: string;
	author: string;
	haIntegration?: string;
	documentation?: string;
	provides?: BindingTemplate[];
	maxChannels?: number;
	products?: ProductVariant[];
	fanType?: FanType;
}

export interface Location {
	id: string;
	name: string;
	lat: number;
	lon: number;
	address: string;
}

export interface GeocodeResult {
	displayName: string;
	lat: number;
	lon: number;
}

export interface Weather {
	temp: SeriesPoint[];
	humidity: SeriesPoint[];
	pressure: SeriesPoint[];
}

export interface Environment {
	id: string;
	name: string;
	kind: EnvironmentKind;
	airSourceId: string;
	locationId: string;
	controlGrowId: string;
	model: string;
	widthCm: number;
	depthCm: number;
	heightCm: number;
	targetTempC: number;
	targetHumidity: number;
	targetCO2: number;
	emergencyTempC: number;
	leafTempOffsetC: number;
}

export interface Binding {
	id: string;
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
	fanType?: FanType;
	sizeMm?: number;
	maxRpm?: number;
	airflowCfm?: number;
	staticPressureMmH2O?: number;
	startingVoltage?: number;
	ductSizeInches?: number;
	noiseDba?: number;
	wattage?: number;
	primary?: boolean;
	streamUrl?: string;
	cameraType?: CameraType;
	cameraCaptureInterval?: number;
	cameraRetentionDays?: number;
	cameraStorageMb?: number;
}

export interface DiscoveredEntity {
	entity: string;
	name: string;
	kind: BindingKind;
	measurement?: Measurement;
	haDeviceId?: string;
	deviceName?: string;
	integration?: string;
	entityCategory?: string;
	manufacturer?: string;
	model?: string;
	unit?: string;
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
	maxRpm?: number;
	on: boolean;
	wattage?: number;
	power?: number; // lights: actual measured watts (from plug meter), else rated while on
	primary?: boolean;
}

export interface CameraRef {
	id: string;
	name: string;
	entity?: string;
	/** Generic (non-Home-Assistant) camera stream. */
	streamUrl?: string;
	cameraType?: CameraType;
	cameraCaptureInterval?: number;
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
	grow?: GrowSummary;
	schedule?: LightSchedule;
}

export interface Snapshot {
	time: string;
	environments: EnvironmentView[];
	grows: GrowView[];
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

export interface SeriesPoint {
	time: string;
	value: number;
}

export interface DeviceSeries {
	bindingId: string;
	metric: 'rpm' | 'speed' | 'power';
	points: SeriesPoint[];
}

export interface SensorSeries {
	bindingId: string;
	name: string;
	entity: string;
	measurement: Measurement;
	points: SeriesPoint[];
}

export interface WeatherHistory {
	temp: SeriesPoint[];
	humidity: SeriesPoint[];
	pressure: SeriesPoint[];
}

export interface Activity {
	id: string;
	environmentId?: string;
	growId?: string;
	deviceId?: string;
	time: string;
	level: 'info' | 'warning' | 'error';
	type: 'control' | 'warning' | 'notice' | 'configuration';
	message: string;
}

export interface Info {
	adapter: string;
}

// --- users & auth ---

export type UserRole = 'admin' | 'user';
export type AccessLevel = 'read' | 'write';

export interface EnvAccess {
	environmentId: string;
	access: AccessLevel;
}

export interface User {
	id: string;
	username: string;
	role: UserRole;
	disabled: boolean;
	created: string;
	/** Per-environment grants; empty/omitted for admins (implicit full access). */
	access: EnvAccess[];
}

export interface AuthStatus {
	needsSetup: boolean;
	signupEnabled: boolean;
}

export interface AuthResult {
	token: string;
	user: User;
}

// --- Home Assistant appliance status (admin control panel) ---

export interface HAComponent {
	version: string;
	versionLatest: string;
	updateAvailable: boolean;
}

export interface HAAddon {
	slug: string;
	name: string;
	version: string;
	versionLatest: string;
	updateAvailable: boolean;
}

export interface HASupervisor {
	/** True only when GrowRig runs as a HAOS add-on (Supervisor reachable). */
	available: boolean;
	core: HAComponent;
	os: HAComponent;
	supervisor: HAComponent;
	addons: HAAddon[];
	error?: string;
}

export interface HAStatus {
	adapter: string; // 'simulator' | 'homeassistant'
	health: string; // 'online' | 'stale' | 'offline'
	supervisor: HASupervisor;
}

export type HAUpdateTarget = 'core' | 'os' | 'supervisor' | 'addon';
