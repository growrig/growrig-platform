// Package control is Grow Core's reconciliation engine.
//
// Given actual state, desired targets and safety constraints, it computes the
// desired state of each device channel. The control law is expressed as pure
// functions so it can be reasoned about and unit-tested independently of the
// adapters and storage.
package control

import "github.com/growrig/growrig/growcore/internal/domain"

const (
	minExhaust   = 15  // always keep some air exchange
	exhaustGain  = 18  // % per °C over target
	baseCirc     = 35  // steady circulation airflow
	circBoost    = 8   // extra circulation per °C over target
	emergencyPWM = 100 // full airflow in an emergency
)

// DesiredChannelSpeed picks the PWM speed (0-100) for a fan channel, honoring
// the environment's air-exchange control mode. In automatic mode it defers to
// the climate control law (ChannelSpeed). In manual mode it holds the grower's
// configured fixed exhaust/circulation speeds — except that an emergency
// over-temperature still forces every fan to full as a safety floor.
func DesiredChannelSpeed(role domain.Role, env domain.Environment, tempC float64) int {
	if env.Control.AirExchange.Mode != domain.ControlManual {
		return ChannelSpeed(role, env, tempC)
	}
	if env.EmergencyTempC > 0 && tempC >= env.EmergencyTempC {
		if role == domain.RoleUnassigned {
			return 0
		}
		return emergencyPWM
	}
	switch role {
	case domain.RoleExhaust, domain.RoleIntake:
		return clamp(env.Control.AirExchange.Exhaust, 0, 100)
	case domain.RoleCirculation:
		return clamp(env.Control.AirExchange.Circulation, 0, 100)
	default:
		return 0
	}
}

// ChannelSpeed computes the desired PWM speed (0-100) for a channel given its
// role and the current vs. target climate of its environment.
func ChannelSpeed(role domain.Role, env domain.Environment, tempC float64) int {
	// Safety first: an over-temperature environment drives every fan to max.
	if env.EmergencyTempC > 0 && tempC >= env.EmergencyTempC {
		if role == domain.RoleUnassigned {
			return 0
		}
		return emergencyPWM
	}

	over := tempC - env.TargetTempC // positive when too hot

	switch role {
	case domain.RoleExhaust, domain.RoleIntake:
		if over <= 0 {
			return minExhaust
		}
		return clamp(minExhaust+int(over*exhaustGain), minExhaust, 100)
	case domain.RoleCirculation:
		if over <= 0 {
			return baseCirc
		}
		return clamp(baseCirc+int(over*circBoost), baseCirc, 100)
	default: // unassigned
		return 0
	}
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
