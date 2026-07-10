package control

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/growrig/growrig-platform/growcore/internal/domain"
	"github.com/growrig/growrig-platform/growcore/internal/store"
)

// Engine ties storage, an adapter and the control law together into a periodic
// reconciliation loop, and publishes a live snapshot each tick.
type Engine struct {
	store   *store.Store
	adapter Adapter

	mu      sync.RWMutex
	latest  domain.Snapshot
	onSnap  func(domain.Snapshot)
	persist int
	tick    int
}

func New(st *store.Store, adapter Adapter, onSnapshot func(domain.Snapshot)) *Engine {
	return &Engine{store: st, adapter: adapter, onSnap: onSnapshot, persist: 5}
}

func (e *Engine) Run(ctx context.Context, interval time.Duration) {
	if err := e.step(interval); err != nil {
		log.Printf("control: initial step: %v", err)
	}
	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			if err := e.step(interval); err != nil {
				log.Printf("control: step: %v", err)
			}
		}
	}
}

func (e *Engine) Latest() domain.Snapshot {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.latest
}

// climate is an environment's aggregated sensor state for one tick.
type climate struct {
	tempC, humidity, co2    float64
	hasTemp, hasHum, hasCO2 bool
	vpd                     float64
}

func (c climate) hasClimate() bool { return c.hasTemp && c.hasHum }

// step runs one reconciliation cycle.
func (e *Engine) step(dt time.Duration) error {
	e.adapter.Tick(dt)

	envs, err := e.store.Environments()
	if err != nil {
		return err
	}
	bindings, err := e.store.Bindings()
	if err != nil {
		return err
	}
	byEnv := map[string][]domain.Binding{}
	for _, b := range bindings {
		byEnv[b.EnvironmentID] = append(byEnv[b.EnvironmentID], b)
	}
	cycles, err := e.store.Cycles()
	if err != nil {
		return err
	}
	cycleByEnv := map[string]domain.Cycle{}
	for _, c := range cycles {
		cycleByEnv[c.EnvironmentID] = c
	}

	// Pass 1: aggregate sensor climate per environment (needed before tents can
	// reference their lung room's readings).
	climates := map[string]climate{}
	sensorsByEnv := map[string][]domain.SensorReading{}
	for _, env := range envs {
		c, sensors := e.readSensors(byEnv[env.ID])
		if c.hasTemp && c.hasHum {
			c.vpd = round2(domain.VPD(c.tempC, c.humidity))
		}
		climates[env.ID] = c
		sensorsByEnv[env.ID] = sensors
	}

	// Pass 2: drive controls and assemble the live views.
	now := time.Now()
	health := e.adapter.Health()
	views := make([]domain.EnvironmentView, 0, len(envs))
	for _, env := range envs {
		c := climates[env.ID]
		view := domain.EnvironmentView{
			Environment: env,
			Health:      health,
			HasClimate:  c.hasClimate(),
			HasTemp:     c.hasTemp,
			HasHum:      c.hasHum,
			TempC:       round1(c.tempC),
			Humidity:    round1(c.humidity),
			CO2:         round0(c.co2),
			HasCO2:      c.hasCO2,
			VPD:         c.vpd,
			Sensors:     sensorsByEnv[env.ID],
		}

		exhaust := 0
		for _, b := range byEnv[env.ID] {
			switch b.Kind {
			case domain.KindFan:
				cs := domain.ControlState{ID: b.ID, Name: b.Name, Kind: domain.KindFan, Role: b.Role, Entity: b.Entity}
				if c.hasTemp {
					speed := ChannelSpeed(b.Role, env, c.tempC)
					cs.DesiredSpeed = speed
					if err := e.adapter.SetFan(b.Entity, speed); err != nil {
						log.Printf("control: set fan %s: %v", b.Entity, err)
					}
					if (b.Role == domain.RoleExhaust || b.Role == domain.RoleIntake) && speed > exhaust {
						exhaust = speed
					}
				}
				if rpm, ok := e.adapter.Value(b.RPMEntity); ok {
					cs.RPM = int(rpm)
				}
				view.Controls = append(view.Controls, cs)
			case domain.KindLight:
				cs := domain.ControlState{
					ID: b.ID, Name: b.Name, Kind: domain.KindLight, Entity: b.Entity,
					Wattage: b.Wattage, Primary: b.Primary,
				}
				if on, ok := e.adapter.SwitchState(b.Entity); ok {
					cs.On = on
				}
				view.Controls = append(view.Controls, cs)
			case domain.KindCamera:
				view.Cameras = append(view.Cameras, domain.CameraRef{ID: b.ID, Name: b.Name, Entity: b.Entity})
			}
		}

		if c, ok := cycleByEnv[env.ID]; ok {
			cycle := c
			view.Cycle = &cycle
		}

		if env.Kind == domain.KindTent && env.AirSourceID != "" {
			if src := findEnv(envs, env.AirSourceID); src != nil {
				sc := climates[src.ID]
				view.AirSource = &domain.AirSourceView{
					ID: src.ID, Name: src.Name,
					TempC: round1(sc.tempC), Humidity: round1(sc.humidity),
					VPD: sc.vpd, OK: sc.hasClimate(),
				}
			}
		}

		views = append(views, view)

		if e.tick%e.persist == 0 && c.hasClimate() {
			e.store.InsertReading(domain.Reading{
				EnvironmentID: env.ID, Time: now,
				TempC: round1(c.tempC), Humidity: round1(c.humidity),
				CO2: round0(c.co2), VPD: c.vpd, ExhaustSpeed: exhaust,
			})
		}
	}

	e.tick++
	snap := domain.Snapshot{Time: now, Environments: views}
	e.mu.Lock()
	e.latest = snap
	e.mu.Unlock()
	if e.onSnap != nil {
		e.onSnap(snap)
	}
	return nil
}

// readSensors reads every sensor binding and aggregates temperature, humidity
// and CO2 (averaging when several sensors of the same kind are present).
func (e *Engine) readSensors(bindings []domain.Binding) (climate, []domain.SensorReading) {
	var c climate
	var tempSum, humSum, co2Sum float64
	var tempN, humN, co2N int
	var readings []domain.SensorReading

	for _, b := range bindings {
		if b.Kind != domain.KindSensor {
			continue
		}
		v, ok := e.adapter.Value(b.Entity)
		readings = append(readings, domain.SensorReading{
			ID: b.ID, Name: b.Name, Measurement: b.Measurement, Entity: b.Entity,
			Value: round1(v), OK: ok,
		})
		if !ok {
			continue
		}
		switch b.Measurement {
		case domain.MeasureTemperature:
			tempSum += v
			tempN++
		case domain.MeasureHumidity:
			humSum += v
			humN++
		case domain.MeasureCO2:
			co2Sum += v
			co2N++
		}
	}
	if tempN > 0 {
		c.tempC, c.hasTemp = tempSum/float64(tempN), true
	}
	if humN > 0 {
		c.humidity, c.hasHum = humSum/float64(humN), true
	}
	if co2N > 0 {
		c.co2, c.hasCO2 = co2Sum/float64(co2N), true
	}
	return c, readings
}

func findEnv(envs []domain.Environment, id string) *domain.Environment {
	for i := range envs {
		if envs[i].ID == id {
			return &envs[i]
		}
	}
	return nil
}

func round0(v float64) float64 { return float64(int(v + 0.5)) }
func round1(v float64) float64 { return float64(int(v*10+0.5)) / 10 }
func round2(v float64) float64 { return float64(int(v*100+0.5)) / 100 }
