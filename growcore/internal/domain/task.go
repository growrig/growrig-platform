package domain

import "time"

// A Task is something the grower should do — "something should be done" — as
// opposed to alerts ("something is wrong") or the care journal ("something
// happened"). Completing a task records a CareEvent and links it via
// CompletedCareEventID, tying planned work to the journal.

type TaskStatus string

const (
	TaskOpen      TaskStatus = "open"
	TaskCompleted TaskStatus = "completed"
	TaskSkipped   TaskStatus = "skipped"
)

type TaskSource string

const (
	TaskManual     TaskSource = "manual"
	TaskRecipe     TaskSource = "recipe"
	TaskAutomation TaskSource = "automation"
)

// Task is one actionable item for the grower, optionally scoped to a grow,
// environment, or plant unit, and optionally due at a time.
type Task struct {
	ID                   string     `json:"id"`
	GrowID               string     `json:"growId,omitempty"`
	EnvironmentID        string     `json:"environmentId,omitempty"`
	PlantUnitID          string     `json:"plantUnitId,omitempty"`
	ActionType           string     `json:"actionType"` // water, feed, inspect, calibrate, …
	Title                string     `json:"title"`
	DueAt                *time.Time `json:"dueAt,omitempty"`
	Status               TaskStatus `json:"status"`
	Source               TaskSource `json:"source"`
	CompletedCareEventID string     `json:"completedCareEventId,omitempty"`
	CreatedAt            time.Time  `json:"createdAt"`
	CompletedAt          *time.Time `json:"completedAt,omitempty"`
}
