package domain

import "time"

// FeedingPreset is a nutrient schedule: an ordered set of products (nutrient
// lines) dosed across an ordered set of phases, each phase spanning one or more
// weeks. It is deliberately flexible so it can model any brand's chart (e.g.
// BioBizz): phases are free-form and only optionally linked to a species stage.
//
// Built-in presets are defined as YAML under species/<id>/feedings.yaml and are
// read-only (Source == "builtin"); user presets are created in-app and stored
// in the DB (Source == "user"). Both share this shape and one API surface.
type FeedingPreset struct {
	ID          string           `json:"id" yaml:"id"`
	Species     string           `json:"species" yaml:"-"`
	Name        string           `json:"name" yaml:"name"`
	Brand       string           `json:"brand" yaml:"brand"`
	Description string           `json:"description" yaml:"description"`
	// Source is "builtin" (YAML, read-only) or "user" (DB, editable). It is
	// derived at load time, never persisted in the preset body.
	Source string `json:"source" yaml:"-"`
	// Unit is the default dose unit (e.g. "ml/L"); a product may override it.
	Unit      string           `json:"unit" yaml:"unit"`
	Products  []FeedingProduct `json:"products" yaml:"products"`
	Phases    []FeedingPhase   `json:"phases" yaml:"phases"`
	CreatedAt time.Time        `json:"createdAt" yaml:"-"`
}

// FeedingProduct is one nutrient line in a schedule. Key identifies its dose in
// each week's Doses map; Unit overrides the preset default when set.
type FeedingProduct struct {
	Key   string `json:"key" yaml:"key"`
	Label string `json:"label" yaml:"label"`
	Unit  string `json:"unit,omitempty" yaml:"unit,omitempty"`
}

// FeedingPhase is a named span of the schedule (e.g. "Vegetative", "Flowering
// wk1-8"). Stage optionally links it to one of the species' cultivation stages.
// Weeks holds the per-week doses in order.
type FeedingPhase struct {
	Name  string        `json:"name" yaml:"name"`
	Stage string        `json:"stage,omitempty" yaml:"stage,omitempty"`
	Weeks []FeedingWeek `json:"weeks" yaml:"weeks"`
}

// FeedingWeek is one week of dosing: product key -> dose amount in the product's
// (or preset's) unit. A missing/zero key means that product isn't fed that week.
type FeedingWeek struct {
	Doses map[string]float64 `json:"doses" yaml:"doses"`
}
