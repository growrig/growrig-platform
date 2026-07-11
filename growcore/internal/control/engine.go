package control

import (
	"context"
	"fmt"
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

	mu              sync.RWMutex
	latest          domain.Snapshot
	onSnap          func(domain.Snapshot)
	persist         int
	tick            int
	fanCommands     map[string]int
	sensorStates    map[string]bool
	emergencyStates map[string]bool
	issueStates     map[string]bool
	lastHealth      domain.ControllerHealth
}

func New(st *store.Store, adapter Adapter, onSnapshot func(domain.Snapshot)) *Engine {
	return &Engine{store: st, adapter: adapter, onSnap: onSnapshot, persist: 5,
		fanCommands: map[string]int{}, sensorStates: map[string]bool{}, emergencyStates: map[string]bool{}, issueStates: map[string]bool{}}
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
	powerEntityByDevice := map[string]string{}
	controllerChannels := map[string]domain.Binding{}
	for _, b := range bindings {
		byEnv[b.EnvironmentID] = append(byEnv[b.EnvironmentID], b)
		if b.Kind == domain.KindPower {
			powerEntityByDevice[b.DeviceID] = b.Entity
		}
		if b.Kind == domain.KindController {
			controllerChannels[b.ID] = b
		}
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
		if e.lastHealth != health {
			level, eventType := "info", "notice"
			if health != domain.HealthOnline {
				level, eventType = "warning", "warning"
			}
			e.activity(env.ID, "", level, eventType, "Home Assistant connection is "+string(health))
		}
		for _, sensor := range sensorsByEnv[env.ID] {
			previous, seen := e.sensorStates[sensor.ID]
			if (!seen && !sensor.OK) || (seen && previous != sensor.OK) {
				if sensor.OK {
					e.activity(env.ID, sensor.ID, "info", "notice", sensor.Name+" is reporting again")
				} else {
					e.activity(env.ID, sensor.ID, "warning", "warning", sensor.Name+" is unavailable")
				}
			}
			e.sensorStates[sensor.ID] = sensor.OK
		}
		emergency := c.hasTemp && env.EmergencyTempC > 0 && c.tempC >= env.EmergencyTempC
		if previous := e.emergencyStates[env.ID]; emergency != previous {
			if emergency {
				e.activity(env.ID, "", "warning", "warning", "Emergency temperature reached")
			} else if previous {
				e.activity(env.ID, "", "info", "notice", "Temperature returned below the emergency limit")
			}
			e.emergencyStates[env.ID] = emergency
		}
		hasFans := false
		for _, binding := range byEnv[env.ID] {
			if binding.Kind == domain.KindFan {
				hasFans = true
				break
			}
		}
		e.issue(env.ID+":fan-climate", hasFans && !c.hasTemp, env.ID, "", "Fan control is paused because no temperature reading is available", "Temperature reading restored; fan control resumed")
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
				channel := controllerChannels[b.ControllerChannelID]
				if channel.Entity == "" {
					channel = b
				}
				cs := domain.ControlState{ID: b.ID, Name: b.Name, Kind: domain.KindFan, Role: b.Role, Entity: channel.Entity}
				if c.hasTemp {
					speed := ChannelSpeed(b.Role, env, c.tempC)
					cs.DesiredSpeed = speed
					err := e.adapter.SetFan(channel.Entity, speed)
					if err != nil {
						log.Printf("control: set fan %s: %v", channel.Entity, err)
					}
					if previous, seen := e.fanCommands[channel.Entity]; !seen || previous != speed {
						if err != nil {
							e.activity(env.ID, b.DeviceID, "error", "control", "Failed to set "+b.Name+" speed")
						} else {
							e.activity(env.ID, b.DeviceID, "info", "control", fmt.Sprintf("Set %s to %d%% via %s", b.Name, speed, channel.Name))
						}
						e.fanCommands[channel.Entity] = speed
					}
					if (b.Role == domain.RoleExhaust || b.Role == domain.RoleIntake) && speed > exhaust {
						exhaust = speed
					}
				}
				if rpm, ok := e.adapter.Value(channel.RPMEntity); ok {
					cs.RPM = int(rpm)
				}
				view.Controls = append(view.Controls, cs)
			case domain.KindLight:
				entity := powerEntityByDevice[b.PowerControllerID]
				e.issue(env.ID+":light:"+b.ID, entity == "", env.ID, b.DeviceID, b.Name+" has no power controller assigned", b.Name+" power controller is connected")
				cs := domain.ControlState{
					ID: b.ID, Name: b.Name, Kind: domain.KindLight, Entity: entity,
					Wattage: b.Wattage, Primary: b.Primary,
				}
				if on, ok := e.adapter.SwitchState(entity); ok {
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
	e.lastHealth = health

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

func (e *Engine) activity(envID, deviceID, level, eventType, message string) {
	if err := e.store.AddActivity(domain.Activity{EnvironmentID: envID, DeviceID: deviceID, Level: level, Type: eventType, Message: message}); err != nil {
		log.Printf("activity: %v", err)
	}
}

func (e *Engine) issue(key string, active bool, envID, deviceID, warning, resolved string) {
	previous, seen := e.issueStates[key]
	if active && (!seen || !previous) {
		e.activity(envID, deviceID, "warning", "warning", warning)
	}
	if !active && seen && previous {
		e.activity(envID, deviceID, "info", "notice", resolved)
	}
	e.issueStates[key] = active
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
