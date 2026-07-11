package domain

import (
	"testing"
	"time"
)

func at(hhmm string) time.Time {
	t, _ := time.ParseInLocation("15:04", hhmm, time.Local)
	// Anchor to a fixed date; only the clock time matters.
	return time.Date(2026, 7, 11, t.Hour(), t.Minute(), 0, 0, time.Local)
}

func TestDesiredOn_PhaseMode(t *testing.T) {
	s := LightSchedule{Mode: LightSchedulePhase, LightsOnAt: "06:00", PhaseOnHours: map[Phase]float64{}}

	// Vegetative default is 18/6: on 06:00–00:00.
	cases := []struct {
		now   string
		phase Phase
		want  bool
	}{
		{"06:00", PhaseVegetative, true},
		{"23:59", PhaseVegetative, true},
		{"00:00", PhaseVegetative, false}, // exactly off boundary (06:00+18h)
		{"03:00", PhaseVegetative, false},
		{"05:59", PhaseVegetative, false},
		// Flowering default is 12/12: on 06:00–18:00.
		{"12:00", PhaseFlowering, true},
		{"18:00", PhaseFlowering, false},
		{"19:00", PhaseFlowering, false},
		// Drying default is 0h: always dark.
		{"12:00", PhaseDrying, false},
	}
	for _, c := range cases {
		got, ok := s.DesiredOn(c.phase, at(c.now))
		if !ok {
			t.Fatalf("phase mode should drive the light")
		}
		if got != c.want {
			t.Errorf("DesiredOn(%s, %s) = %v, want %v", c.phase, c.now, got, c.want)
		}
	}
}

func TestDesiredOn_WrapMidnight(t *testing.T) {
	// Lights on at 20:00 for 12h -> on 20:00–08:00, wrapping midnight.
	s := LightSchedule{Mode: LightScheduleCustom, LightsOnAt: "20:00", OnHours: 12}
	on := []string{"20:00", "23:30", "00:30", "07:59"}
	off := []string{"08:00", "12:00", "19:59"}
	for _, hhmm := range on {
		if got, _ := s.DesiredOn(PhaseVegetative, at(hhmm)); !got {
			t.Errorf("expected light ON at %s", hhmm)
		}
	}
	for _, hhmm := range off {
		if got, _ := s.DesiredOn(PhaseVegetative, at(hhmm)); got {
			t.Errorf("expected light OFF at %s", hhmm)
		}
	}
}

func TestDesiredOn_OffMode(t *testing.T) {
	s := LightSchedule{Mode: LightScheduleOff, LightsOnAt: "06:00", OnHours: 18}
	if _, ok := s.DesiredOn(PhaseVegetative, at("12:00")); ok {
		t.Errorf("off mode must not drive the light")
	}
}

func TestNextTransition(t *testing.T) {
	// 12/12 from 06:00 -> transitions at 06:00 and 18:00.
	s := LightSchedule{Mode: LightScheduleCustom, LightsOnAt: "06:00", OnHours: 12}
	if got := s.NextTransition(PhaseVegetative, at("10:00")); got.Hour() != 18 {
		t.Errorf("next transition after 10:00 should be 18:00, got %v", got)
	}
	if got := s.NextTransition(PhaseVegetative, at("20:00")); got.Hour() != 6 {
		t.Errorf("next transition after 20:00 should be next 06:00, got %v", got)
	}
	// Always-on / always-off schedules have no boundary.
	dark := LightSchedule{Mode: LightScheduleCustom, LightsOnAt: "06:00", OnHours: 0}
	if got := dark.NextTransition(PhaseVegetative, at("10:00")); !got.IsZero() {
		t.Errorf("always-off schedule should have no transition, got %v", got)
	}
}

func TestEffectiveOnHours_Override(t *testing.T) {
	s := LightSchedule{Mode: LightSchedulePhase, PhaseOnHours: map[Phase]float64{PhaseFlowering: 11}}
	if h := s.EffectiveOnHours(PhaseFlowering); h != 11 {
		t.Errorf("override should win: got %v", h)
	}
	if h := s.EffectiveOnHours(PhaseVegetative); h != 18 {
		t.Errorf("veg should fall back to default 18: got %v", h)
	}
}
