// Package domain defines the grow-domain model for Grow Core.
//
// The model is entity-oriented and semantic: an Environment (a grow tent or a
// monitored room such as a lung room) owns a set of typed Bindings that attach
// Home Assistant entities as sensors, fans, lights or cameras. This mirrors how
// Home Assistant exposes things while keeping Grow Core's own roles and targets
// independent. See ../../../growrig/docs/architecture.md.
package domain

import "time"

// EnvironmentKind distinguishes a controlled grow space from a monitored room.
type EnvironmentKind string

const (
	// KindTent is a controlled grow space with targets and control.
	KindTent EnvironmentKind = "tent"
	// KindRoom is a monitored space (e.g. a lung room feeding a tent's intake).
	KindRoom EnvironmentKind = "room"
)

// Role is the grow purpose assigned to a fan channel.
type Role string

const (
	RoleUnassigned  Role = "unassigned"
	RoleExhaust     Role = "exhaust"
	RoleIntake      Role = "intake"
	RoleCirculation Role = "circulation"
)

// AllFanRoles lists the roles a fan may be assigned.
var AllFanRoles = []Role{RoleUnassigned, RoleExhaust, RoleIntake, RoleCirculation}

// BindingKind is the category of thing an entity binding represents.
type BindingKind string

const (
	KindSensor BindingKind = "sensor"
	KindFan    BindingKind = "fan"
	KindLight  BindingKind = "light"
	KindCamera BindingKind = "camera"
)

// Measurement is what a sensor binding measures.
type Measurement string

const (
	MeasureTemperature Measurement = "temperature"
	MeasureHumidity    Measurement = "humidity"
	MeasureCO2         Measurement = "co2"
)

// ControllerHealth describes connection/adapter liveness.
type ControllerHealth string

const (
	HealthOnline  ControllerHealth = "online"
	HealthStale   ControllerHealth = "stale"
	HealthOffline ControllerHealth = "offline"
)

// Environment is a controlled tent or a monitored room.
type Environment struct {
	ID   string          `json:"id"`
	Name string          `json:"name"`
	Kind EnvironmentKind `json:"kind"`
	// AirSourceID optionally references the room (lung room) that supplies this
	// tent's intake air. Empty for rooms or tents without a linked source.
	AirSourceID string `json:"airSourceId"`

	// Model is an optional descriptive field (e.g. the grow-tent product)
	// captured by the setup wizard.
	Model string `json:"model"`

	// Tent dimensions in centimetres; 0 = unset. VolumeM3 derives from these.
	WidthCm  float64 `json:"widthCm"`
	DepthCm  float64 `json:"depthCm"`
	HeightCm float64 `json:"heightCm"`

	TargetTempC    float64 `json:"targetTempC"`
	TargetHumidity float64 `json:"targetHumidity"`
	TargetCO2      float64 `json:"targetCO2"` // ppm; 0 = unset
	EmergencyTempC float64 `json:"emergencyTempC"`
}

// VolumeM3 returns the tent's air volume in cubic metres, or 0 if any
// dimension is unset.
func (e Environment) VolumeM3() float64 {
	if e.WidthCm <= 0 || e.DepthCm <= 0 || e.HeightCm <= 0 {
		return 0
	}
	return e.WidthCm * e.DepthCm * e.HeightCm / 1_000_000
}

// Phase is a stage of a grow cycle.
type Phase string

const (
	PhaseSeedling   Phase = "seedling"
	PhaseVegetative Phase = "vegetative"
	PhaseFlowering  Phase = "flowering"
	PhaseFlush      Phase = "flush"
	PhaseDrying     Phase = "drying"
	PhaseCure       Phase = "cure"
)

// AllPhases lists grow phases in chronological order.
var AllPhases = []Phase{PhaseSeedling, PhaseVegetative, PhaseFlowering, PhaseFlush, PhaseDrying, PhaseCure}

// Cycle is a running grow in a tent: a strain, a start date, and the current
// phase. One active cycle per environment in this MVP.
type Cycle struct {
	EnvironmentID string    `json:"environmentId"`
	Strain        string    `json:"strain"`
	StartedAt     time.Time `json:"startedAt"`
	Phase         Phase     `json:"phase"`
	PhaseStarted  time.Time `json:"phaseStarted"`
	Notes         string    `json:"notes"`
}

// Binding attaches a Home Assistant entity (or simulator entity id) to an
// environment with a semantic category.
type Binding struct {
	ID            string      `json:"id"`
	EnvironmentID string      `json:"environmentId"`
	Kind          BindingKind `json:"kind"`
	Name          string      `json:"name"`
	Entity        string      `json:"entity"`

	// Sensor only:
	Measurement Measurement `json:"measurement,omitempty"`
	// Fan only:
	Role      Role   `json:"role,omitempty"`
	RPMEntity string `json:"rpmEntity,omitempty"`
	// Light only:
	Wattage float64 `json:"wattage,omitempty"` // rated power in watts; 0 = unknown
	Primary bool    `json:"primary,omitempty"` // the box's main grow light (one per env)
}

// Reading is a single historical sample persisted for an environment.
type Reading struct {
	EnvironmentID string    `json:"environmentId"`
	Time          time.Time `json:"time"`
	TempC         float64   `json:"tempC"`
	Humidity      float64   `json:"humidity"`
	CO2           float64   `json:"co2"`
	VPD           float64   `json:"vpd"`
	ExhaustSpeed  int       `json:"exhaustSpeed"`
}

// --- Live view types (built each control tick, sent to clients) ---

// SensorReading is a sensor binding with its current value.
type SensorReading struct {
	ID          string      `json:"id"`
	Name        string      `json:"name"`
	Measurement Measurement `json:"measurement"`
	Entity      string      `json:"entity"`
	Value       float64     `json:"value"`
	OK          bool        `json:"ok"`
}

// ControlState is a fan or light binding with its current state.
type ControlState struct {
	ID           string      `json:"id"`
	Name         string      `json:"name"`
	Kind         BindingKind `json:"kind"`
	Role         Role        `json:"role,omitempty"`
	Entity       string      `json:"entity"`
	DesiredSpeed int         `json:"desiredSpeed"`      // fans
	RPM          int         `json:"rpm"`               // fans
	On           bool        `json:"on"`                // lights
	Wattage      float64     `json:"wattage,omitempty"` // lights: rated power (W)
	Primary      bool        `json:"primary,omitempty"` // lights: the box's main grow light
}

// CameraRef is a camera binding (no stream in this MVP).
type CameraRef struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Entity string `json:"entity"`
}

// AirSourceView summarises a linked lung room on a tent's dashboard.
type AirSourceView struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	TempC    float64 `json:"tempC"`
	Humidity float64 `json:"humidity"`
	VPD      float64 `json:"vpd"`
	OK       bool    `json:"ok"`
}

// EnvironmentView is the full live state of one environment.
type EnvironmentView struct {
	Environment
	Health     ControllerHealth `json:"health"`
	HasClimate bool             `json:"hasClimate"`
	HasTemp    bool             `json:"hasTemp"`
	HasHum     bool             `json:"hasHum"`
	TempC      float64          `json:"tempC"`
	Humidity   float64          `json:"humidity"`
	CO2        float64          `json:"co2"`
	HasCO2     bool             `json:"hasCO2"`
	VPD        float64          `json:"vpd"`
	Sensors    []SensorReading  `json:"sensors"`
	Controls   []ControlState   `json:"controls"`
	Cameras    []CameraRef      `json:"cameras"`
	AirSource  *AirSourceView   `json:"airSource,omitempty"`
	Cycle      *Cycle           `json:"cycle,omitempty"`
}

// Snapshot is the full live system state broadcast to clients.
type Snapshot struct {
	Time         time.Time         `json:"time"`
	Environments []EnvironmentView `json:"environments"`
}
