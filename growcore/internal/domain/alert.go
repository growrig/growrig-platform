package domain

import "time"

// Alerts capture a system-detected condition — "something is wrong" — as opposed
// to activity_log (history: "something happened") or tasks ("something should be
// done"). The control engine opens an alert when a condition becomes active and
// resolves it when the condition clears, deduplicating by a stable Key so a
// long-running problem is a single row that survives restarts (unlike the
// engine's in-memory issueStates).

type AlertSeverity string

const (
	AlertInfo     AlertSeverity = "info"
	AlertWarning  AlertSeverity = "warning"
	AlertCritical AlertSeverity = "critical"
)

type AlertStatus string

const (
	AlertOpen         AlertStatus = "open"
	AlertAcknowledged AlertStatus = "acknowledged"
	AlertResolved     AlertStatus = "resolved"
)

// Alert is one system-detected condition. Key is the stable deduplication
// identity (e.g. "climate:env-123:temp-high"); at most one non-resolved row
// exists per key at a time.
type Alert struct {
	ID             string        `json:"id"`
	Key            string        `json:"key"`
	EnvironmentID  string        `json:"environmentId,omitempty"`
	GrowID         string        `json:"growId,omitempty"`
	DeviceID       string        `json:"deviceId,omitempty"`
	Severity       AlertSeverity `json:"severity"`
	Kind           string        `json:"kind"` // sensor_offline, climate, integration, …
	Title          string        `json:"title"`
	Message        string        `json:"message,omitempty"`
	Status         AlertStatus   `json:"status"`
	FirstSeenAt    time.Time     `json:"firstSeenAt"`
	LastSeenAt     time.Time     `json:"lastSeenAt"`
	AcknowledgedAt *time.Time    `json:"acknowledgedAt,omitempty"`
	ResolvedAt     *time.Time    `json:"resolvedAt,omitempty"`
}
