package store

import (
	"testing"

	"github.com/growrig/growrig/growcore/internal/domain"
)

func TestAlertOpenDedupByKey(t *testing.T) {
	st := open(t)

	if err := st.OpenAlert(domain.Alert{Key: "env-1:temp", EnvironmentID: "env-1", Title: "Too hot", Message: "31°C"}); err != nil {
		t.Fatal(err)
	}
	// Re-opening the same key must update the existing row, not add a second.
	if err := st.OpenAlert(domain.Alert{Key: "env-1:temp", EnvironmentID: "env-1", Title: "Too hot", Message: "33°C", Severity: domain.AlertCritical}); err != nil {
		t.Fatal(err)
	}

	open, err := st.OpenAlerts()
	if err != nil {
		t.Fatal(err)
	}
	if len(open) != 1 {
		t.Fatalf("expected 1 open alert after dedup, got %d", len(open))
	}
	if open[0].Message != "33°C" || open[0].Severity != domain.AlertCritical {
		t.Fatalf("expected refreshed message/severity, got %+v", open[0])
	}
}

func TestAlertResolveAndReopen(t *testing.T) {
	st := open(t)
	_ = st.OpenAlert(domain.Alert{Key: "env-1:fan", Title: "Fan stalled"})

	if err := st.ResolveAlert("env-1:fan"); err != nil {
		t.Fatal(err)
	}
	openAlerts, _ := st.OpenAlerts()
	if len(openAlerts) != 0 {
		t.Fatalf("expected no open alerts after resolve, got %d", len(openAlerts))
	}

	// The unique-open-key index must permit a fresh open row once resolved.
	if err := st.OpenAlert(domain.Alert{Key: "env-1:fan", Title: "Fan stalled again"}); err != nil {
		t.Fatalf("reopen after resolve failed: %v", err)
	}
	openAlerts, _ = st.OpenAlerts()
	if len(openAlerts) != 1 {
		t.Fatalf("expected 1 open alert after reopen, got %d", len(openAlerts))
	}
}
