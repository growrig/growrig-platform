package control

import (
	"testing"

	"github.com/growrig/growrig/growcore/internal/domain"
)

func env() domain.Environment {
	return domain.Environment{TargetTempC: 24, TargetHumidity: 55, EmergencyTempC: 35}
}

func TestExhaustAtTargetRunsAtMinimum(t *testing.T) {
	if got := ChannelSpeed(domain.RoleExhaust, env(), 24); got != minExhaust {
		t.Fatalf("exhaust at target = %d, want %d", got, minExhaust)
	}
}

func TestExhaustRampsWithHeat(t *testing.T) {
	cool := ChannelSpeed(domain.RoleExhaust, env(), 24)
	hot := ChannelSpeed(domain.RoleExhaust, env(), 27)
	if hot <= cool {
		t.Fatalf("exhaust should increase with heat: cool=%d hot=%d", cool, hot)
	}
	if hot > 100 {
		t.Fatalf("exhaust exceeded 100: %d", hot)
	}
}

func TestEmergencyForcesMaxOnFans(t *testing.T) {
	if got := ChannelSpeed(domain.RoleExhaust, env(), 36); got != emergencyPWM {
		t.Fatalf("emergency exhaust = %d, want %d", got, emergencyPWM)
	}
	if got := ChannelSpeed(domain.RoleCirculation, env(), 36); got != emergencyPWM {
		t.Fatalf("emergency circulation = %d, want %d", got, emergencyPWM)
	}
	if got := ChannelSpeed(domain.RoleUnassigned, env(), 36); got != 0 {
		t.Fatalf("emergency unassigned = %d, want 0", got)
	}
}

func TestUnassignedStaysOff(t *testing.T) {
	if got := ChannelSpeed(domain.RoleUnassigned, env(), 30); got != 0 {
		t.Fatalf("unassigned = %d, want 0", got)
	}
}

func TestCirculationHasBaseline(t *testing.T) {
	if got := ChannelSpeed(domain.RoleCirculation, env(), 20); got != baseCirc {
		t.Fatalf("circulation baseline = %d, want %d", got, baseCirc)
	}
}

func manualEnv() domain.Environment {
	e := env()
	e.Control.AirExchange = domain.AirExchangeControl{Mode: domain.ControlManual, Exhaust: 70, Circulation: 40}
	return e
}

func TestDesiredChannelSpeedAutoDefersToClimate(t *testing.T) {
	// With no manual mode set, DesiredChannelSpeed matches the climate law.
	if got, want := DesiredChannelSpeed(domain.RoleExhaust, env(), 24), ChannelSpeed(domain.RoleExhaust, env(), 24); got != want {
		t.Fatalf("auto exhaust = %d, want %d", got, want)
	}
}

func TestDesiredChannelSpeedManualHoldsSetpoints(t *testing.T) {
	if got := DesiredChannelSpeed(domain.RoleExhaust, manualEnv(), 24); got != 70 {
		t.Fatalf("manual exhaust = %d, want 70", got)
	}
	if got := DesiredChannelSpeed(domain.RoleIntake, manualEnv(), 24); got != 70 {
		t.Fatalf("manual intake = %d, want 70", got)
	}
	if got := DesiredChannelSpeed(domain.RoleCirculation, manualEnv(), 24); got != 40 {
		t.Fatalf("manual circulation = %d, want 40", got)
	}
	if got := DesiredChannelSpeed(domain.RoleUnassigned, manualEnv(), 24); got != 0 {
		t.Fatalf("manual unassigned = %d, want 0", got)
	}
}

func TestDesiredChannelSpeedManualStillRespectsEmergency(t *testing.T) {
	// The emergency over-temperature floor overrides the manual setpoints.
	if got := DesiredChannelSpeed(domain.RoleExhaust, manualEnv(), 36); got != emergencyPWM {
		t.Fatalf("manual exhaust at emergency = %d, want %d", got, emergencyPWM)
	}
	if got := DesiredChannelSpeed(domain.RoleCirculation, manualEnv(), 36); got != emergencyPWM {
		t.Fatalf("manual circulation at emergency = %d, want %d", got, emergencyPWM)
	}
}
