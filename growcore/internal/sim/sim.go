// Package sim is a simulator adapter for Grow Core.
//
// It emulates Home Assistant entities for a grow tent (temperature, humidity,
// CO2, two PWM fans, a light, a camera) plus a lung room (temperature,
// humidity), running a small physical model so the whole platform can be
// exercised end-to-end without hardware. It implements control.Adapter.
package sim

import (
	"context"
	"math"
	"math/rand"
	"sync"
	"time"

	"github.com/growrig/growrig-platform/growcore/internal/control"
	"github.com/growrig/growrig-platform/growcore/internal/domain"
)

// Simulated entity ids and the demo environment ids they belong to.
const (
	TentID = "env-main"
	RoomID = "env-lung"

	tentTemp   = "sim.tent_temperature"
	tentHumid  = "sim.tent_humidity"
	tentCO2    = "sim.tent_co2"
	tentFan1   = "sim.tent_fan1"
	tentFan1R  = "sim.tent_fan1_rpm"
	tentFan2   = "sim.tent_fan2"
	tentFan2R  = "sim.tent_fan2_rpm"
	tentLight  = "sim.tent_light"
	tentLightW = "sim.tent_light_power" // plug's actual power draw (W)
	tentCamera = "sim.tent_camera"
	roomTemp   = "sim.room_temperature"
	roomHumid  = "sim.room_humidity"
)

const (
	ambientTempC   = 22.0
	ambientHumid   = 48.0
	ambientCO2     = 450.0
	heatInputC     = 1.6
	lightHeatC     = 0.6
	moistureInput  = 3.0
	co2Input       = 55.0
	baseCoolRate   = 0.04
	exhaustCooling = 0.9
)

type fan struct {
	speed int
	rpm   int
}

type Simulator struct {
	mu sync.Mutex

	tentTempC    float64
	tentHumidity float64
	tentCO2      float64
	lightOn      bool
	fans         map[string]*fan // by fan entity id
	rpmOf        map[string]string

	roomTempC    float64
	roomHumidity float64

	rng *rand.Rand
}

func New() *Simulator {
	s := &Simulator{
		tentTempC:    26.5,
		tentHumidity: 60,
		tentCO2:      800,
		fans:         map[string]*fan{tentFan1: {speed: 40}, tentFan2: {speed: 30}},
		rpmOf:        map[string]string{tentFan1R: tentFan1, tentFan2R: tentFan2},
		roomTempC:    21,
		roomHumidity: 45,
		rng:          rand.New(rand.NewSource(time.Now().UnixNano())),
	}
	return s
}

func (s *Simulator) Start(context.Context) error     { return nil }
func (s *Simulator) Close() error                    { return nil }
func (s *Simulator) Health() domain.ControllerHealth { return domain.HealthOnline }

// Tick advances the physical model by dt using the currently commanded speeds.
func (s *Simulator) Tick(dt time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()

	m := dt.Minutes()
	ex := float64(s.dominantSpeed()) / 100.0

	heat := heatInputC
	if s.lightOn {
		heat += lightHeatC
	}
	coolRate := baseCoolRate + exhaustCooling*ex
	s.tentTempC += heat*m - coolRate*(s.tentTempC-ambientTempC)*m + s.noise(0.05)
	s.tentTempC = clampF(s.tentTempC, ambientTempC-2, 45)

	s.tentHumidity += moistureInput*m - (0.05+ex)*(s.tentHumidity-ambientHumid)*m + s.noise(0.2)
	s.tentHumidity = clampF(s.tentHumidity, 20, 95)

	s.tentCO2 += co2Input*m - (0.1+ex)*(s.tentCO2-ambientCO2)*m + s.noise(4)
	s.tentCO2 = clampF(s.tentCO2, ambientCO2-30, 1600)

	// Lung room drifts slowly and independently.
	s.roomTempC = clampF(s.roomTempC+s.noise(0.03), 18, 26)
	s.roomHumidity = clampF(s.roomHumidity+s.noise(0.15), 35, 65)

	for _, f := range s.fans {
		if f.speed < 8 {
			f.rpm = 0
			continue
		}
		f.rpm = clampInt(int(float64(f.speed)*38+s.noise(60)), 0, 4200)
	}
}

func (s *Simulator) Value(entity string) (float64, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	switch entity {
	case tentTemp:
		return s.tentTempC, true
	case tentHumid:
		return s.tentHumidity, true
	case tentCO2:
		return s.tentCO2, true
	case tentLightW:
		// A real LED grow light draws a bit under its rated wattage; the plug
		// reports a small standby draw when off.
		if s.lightOn {
			return clampF(138+s.noise(2.5), 120, 150), true
		}
		return clampF(0.4+s.noise(0.1), 0, 2), true
	case roomTemp:
		return s.roomTempC, true
	case roomHumid:
		return s.roomHumidity, true
	}
	if fanEntity, ok := s.rpmOf[entity]; ok {
		if f := s.fans[fanEntity]; f != nil {
			return float64(f.rpm), true
		}
	}
	return 0, false
}

func (s *Simulator) SetFan(entity string, speed int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	f := s.fans[entity]
	if f == nil {
		f = &fan{}
		s.fans[entity] = f
	}
	f.speed = clampInt(speed, 0, 100)
	return nil
}

func (s *Simulator) SetSwitch(entity string, on bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if entity == tentLight {
		s.lightOn = on
	}
	return nil
}

func (s *Simulator) SwitchState(entity string) (bool, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if entity == tentLight {
		return s.lightOn, true
	}
	return false, false
}

// Discover advertises the simulated entities so the add-device UI has content
// even without Home Assistant.
func (s *Simulator) Discover() []control.DiscoveredEntity {
	return []control.DiscoveredEntity{
		{Entity: tentTemp, Name: "Tent Temperature", Kind: domain.KindSensor, Measurement: domain.MeasureTemperature},
		{Entity: tentHumid, Name: "Tent Humidity", Kind: domain.KindSensor, Measurement: domain.MeasureHumidity},
		{Entity: tentCO2, Name: "Tent CO₂", Kind: domain.KindSensor, Measurement: domain.MeasureCO2},
		{Entity: roomTemp, Name: "Lung Room Temperature", Kind: domain.KindSensor, Measurement: domain.MeasureTemperature},
		{Entity: roomHumid, Name: "Lung Room Humidity", Kind: domain.KindSensor, Measurement: domain.MeasureHumidity},
		{Entity: tentFan1, Name: "Tent Fan 1", Kind: domain.KindFan},
		{Entity: tentFan2, Name: "Tent Fan 2", Kind: domain.KindFan},
		{Entity: tentLight, Name: "Tent Light", Kind: domain.KindLight},
		{Entity: tentCamera, Name: "Tent Camera", Kind: domain.KindCamera},
	}
}

// SeedTopology returns the demo tent + lung room and their bindings, used to
// populate a fresh database in simulator mode.
func SeedTopology() ([]domain.Environment, []domain.Binding) {
	envs := []domain.Environment{
		{ID: TentID, Name: "Main Grow Tent", Kind: domain.KindTent, AirSourceID: RoomID,
			TargetTempC: 24, TargetHumidity: 55, TargetCO2: 800, EmergencyTempC: 35},
		{ID: RoomID, Name: "Lung Room", Kind: domain.KindRoom},
	}
	b := func(id, env string, kind domain.BindingKind, name, entity string) domain.Binding {
		return domain.Binding{ID: id, DeviceID: id, DeviceName: name, EnvironmentID: env, Kind: kind, Name: name, Entity: entity}
	}
	sensor := func(id, env, name, entity string, m domain.Measurement) domain.Binding {
		x := b(id, env, domain.KindSensor, name, entity)
		x.Measurement = m
		return x
	}
	fanB := func(id, env, name, entity, rpm string, role domain.Role) domain.Binding {
		x := b(id, env, domain.KindFan, name, entity)
		x.Role = role
		x.RPMEntity = rpm
		return x
	}
	lightB := func(id, env, name, entity string, watts float64, primary bool) domain.Binding {
		x := b(id, env, domain.KindLight, name, entity)
		x.Wattage = watts
		x.Primary = primary
		return x
	}
	powerB := func(id, env, name, entity string) domain.Binding {
		return b(id, env, domain.KindPower, name, entity)
	}
	bindings := []domain.Binding{
		sensor("sim-t-temp", TentID, "Temperature", tentTemp, domain.MeasureTemperature),
		sensor("sim-t-humid", TentID, "Humidity", tentHumid, domain.MeasureHumidity),
		sensor("sim-t-co2", TentID, "CO₂", tentCO2, domain.MeasureCO2),
		fanB("sim-t-fan1", TentID, "Fan 1", tentFan1, tentFan1R, domain.RoleExhaust),
		fanB("sim-t-fan2", TentID, "Fan 2", tentFan2, tentFan2R, domain.RoleCirculation),
		lightB("sim-t-light", TentID, "Grow Light", "", 150, true),
		powerB("sim-t-power", TentID, "Grow light plug", tentLight),
		sensor("sim-t-power-meter", TentID, "Grow light power", tentLightW, domain.MeasurePower),
		b("sim-t-cam", TentID, domain.KindCamera, "Tent Camera", tentCamera),
		sensor("sim-r-temp", RoomID, "Temperature", roomTemp, domain.MeasureTemperature),
		sensor("sim-r-humid", RoomID, "Humidity", roomHumid, domain.MeasureHumidity),
	}
	for i := range bindings {
		switch bindings[i].ID {
		case "sim-t-temp", "sim-t-humid", "sim-t-co2":
			bindings[i].DeviceID, bindings[i].DeviceName = "sim-t-climate", "Tent environmental sensor"
		case "sim-t-fan1", "sim-t-fan2":
			bindings[i].DeviceID, bindings[i].DeviceName = "sim-t-fan-controller", "Dual fan controller"
		case "sim-r-temp", "sim-r-humid":
			bindings[i].DeviceID, bindings[i].DeviceName = "sim-r-climate", "Room environmental sensor"
		case "sim-t-power", "sim-t-power-meter":
			// Switch + power-meter are two capabilities of the one smart plug.
			bindings[i].DeviceID, bindings[i].DeviceName = "sim-t-power", "Grow light plug"
		}
		if bindings[i].ID == "sim-t-light" {
			bindings[i].PowerControllerID = "sim-t-power"
		}
	}
	return envs, bindings
}

func (s *Simulator) dominantSpeed() int {
	max := 0
	for _, f := range s.fans {
		if f.speed > max {
			max = f.speed
		}
	}
	return max
}

func (s *Simulator) noise(mag float64) float64 { return (s.rng.Float64()*2 - 1) * mag }

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func clampF(v, lo, hi float64) float64 { return math.Max(lo, math.Min(hi, v)) }
