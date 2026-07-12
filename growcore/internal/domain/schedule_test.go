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
	s := LightSchedule{Mode: LightSchedulePhase, LightsOnAt: "06:00", StageOnHours: map[string]float64{}}

	// Vegetative default is 18/6: on 06:00–00:00.
	cases := []struct {
		now   string
		stage string
		want  bool
	}{
		{"06:00", "vegetative", true},
		{"23:59", "vegetative", true},
		{"00:00", "vegetative", false}, // exactly off boundary (06:00+18h)
		{"03:00", "vegetative", false},
		{"05:59", "vegetative", false},
		// Flowering default is 12/12: on 06:00–18:00.
		{"12:00", "flowering", true},
		{"18:00", "flowering", false},
		{"19:00", "flowering", false},
		// Drying default is 0h: always dark.
		{"12:00", "drying", false},
		// An unknown stage falls back to 18h (like vegetative).
		{"12:00", "fruiting", true},
	}
	for _, c := range cases {
		got, ok := s.DesiredOn(c.stage, at(c.now))
		if !ok {
			t.Fatalf("phase mode should drive the light")
		}
		if got != c.want {
			t.Errorf("DesiredOn(%s, %s) = %v, want %v", c.stage, c.now, got, c.want)
		}
	}
}

func TestDesiredOn_WrapMidnight(t *testing.T) {
	// Lights on at 20:00 for 12h -> on 20:00–08:00, wrapping midnight.
	s := LightSchedule{Mode: LightScheduleCustom, LightsOnAt: "20:00", OnHours: 12}
	on := []string{"20:00", "23:30", "00:30", "07:59"}
	off := []string{"08:00", "12:00", "19:59"}
	for _, hhmm := range on {
		if got, _ := s.DesiredOn("vegetative", at(hhmm)); !got {
			t.Errorf("expected light ON at %s", hhmm)
		}
	}
	for _, hhmm := range off {
		if got, _ := s.DesiredOn("vegetative", at(hhmm)); got {
			t.Errorf("expected light OFF at %s", hhmm)
		}
	}
}

func TestDesiredOn_OffMode(t *testing.T) {
	s := LightSchedule{Mode: LightScheduleOff, LightsOnAt: "06:00", OnHours: 18}
	if _, ok := s.DesiredOn("vegetative", at("12:00")); ok {
		t.Errorf("off mode must not drive the light")
	}
}

func TestNextTransition(t *testing.T) {
	// 12/12 from 06:00 -> transitions at 06:00 and 18:00.
	s := LightSchedule{Mode: LightScheduleCustom, LightsOnAt: "06:00", OnHours: 12}
	if got := s.NextTransition("vegetative", at("10:00")); got.Hour() != 18 {
		t.Errorf("next transition after 10:00 should be 18:00, got %v", got)
	}
	if got := s.NextTransition("vegetative", at("20:00")); got.Hour() != 6 {
		t.Errorf("next transition after 20:00 should be next 06:00, got %v", got)
	}
	// Always-on / always-off schedules have no boundary.
	dark := LightSchedule{Mode: LightScheduleCustom, LightsOnAt: "06:00", OnHours: 0}
	if got := dark.NextTransition("vegetative", at("10:00")); !got.IsZero() {
		t.Errorf("always-off schedule should have no transition, got %v", got)
	}
}

func TestEffectiveOnHours_Override(t *testing.T) {
	s := LightSchedule{Mode: LightSchedulePhase, StageOnHours: map[string]float64{"flowering": 11}}
	if h := s.EffectiveOnHours("flowering"); h != 11 {
		t.Errorf("override should win: got %v", h)
	}
	if h := s.EffectiveOnHours("vegetative"); h != 18 {
		t.Errorf("veg should fall back to default 18: got %v", h)
	}
}
