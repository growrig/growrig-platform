// Package species is Grow Core's built-in database of crop families.
//
// A species defines its ordered cultivation stages (with per-stage default
// light hours) and the schema of species-specific attributes a cultivar of
// that species carries (e.g. cannabis genetics / THC / flowering weeks). The
// grow form derives its stage sequence from the chosen species, and the
// cultivar form renders attribute inputs dynamically from the species schema.
//
// Species are defined as YAML files under the repo-root species/ tree, one per
// species:
//
//	species/<species-id>/species.yaml
//
// The species id is the directory name. At runtime the loader prefers that
// on-disk tree (so edits are live in development), and falls back to the copy
// embedded into the binary — synced from species/ by `make build` — so the
// add-on still ships as one file. This mirrors internal/catalog.
package species

import (
	"embed"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
)

// AttrType is the input kind of a cultivar attribute.
type AttrType string

const (
	AttrText    AttrType = "text"
	AttrNumber  AttrType = "number"
	AttrPercent AttrType = "percent"
	AttrEnum    AttrType = "enum"
)

// Attribute declares one species-specific cultivar field.
type Attribute struct {
	Key     string   `json:"key" yaml:"key"`
	Label   string   `json:"label" yaml:"label"`
	Type    AttrType `json:"type" yaml:"type"`
	Options []string `json:"options,omitempty" yaml:"options,omitempty"`
	Unit    string   `json:"unit,omitempty" yaml:"unit,omitempty"`
}

// Stage is one cultivation phase with its default daily light hours.
type Stage struct {
	Name       string  `json:"name" yaml:"name"`
	LightHours float64 `json:"lightHours" yaml:"lightHours"`
}

// Species is a crop family: an ordered stage sequence plus the cultivar
// attribute schema. ID is the directory name; Label is the display name.
type Species struct {
	ID                 string      `json:"id"`
	Label              string      `json:"label" yaml:"label"`
	Stages             []Stage     `json:"stages" yaml:"stages"`
	CultivarAttributes []Attribute `json:"cultivarAttributes,omitempty" yaml:"cultivarAttributes"`
}

// StageNames returns the ordered stage names.
func (s Species) StageNames() []string {
	names := make([]string, len(s.Stages))
	for i, st := range s.Stages {
		names[i] = st.Name
	}
	return names
}

// LightHours maps each stage name to its default daily light hours.
func (s Species) LightHours() map[string]float64 {
	m := make(map[string]float64, len(s.Stages))
	for _, st := range s.Stages {
		m[st.Name] = st.LightHours
	}
	return m
}

//go:embed all:data
var data embed.FS

var (
	once   sync.Once
	loaded []Species
	byID   map[string]Species
)

// All returns the species catalog, loaded once, sorted by id.
func All() []Species {
	once.Do(load)
	return loaded
}

// SourceFS returns the file system the species catalog is read from: the
// repo-root species/ directory when present (so edits are live in development),
// otherwise the tree embedded into the binary. Sibling loaders (e.g. feeding
// schedules under species/<id>/feedings.yaml) reuse this so they resolve the
// same tree without duplicating the embed or the disk-discovery logic.
func SourceFS() fs.FS {
	if dir := diskDir(); dir != "" {
		return os.DirFS(dir)
	}
	if sub, err := fs.Sub(data, "data"); err == nil {
		return sub
	}
	return nil
}

// Get returns the species with the given id (case-insensitive).
func Get(id string) (Species, bool) {
	once.Do(load)
	s, ok := byID[strings.ToLower(strings.TrimSpace(id))]
	return s, ok
}

// StageNames returns a species' ordered stage sequence.
func StageNames(id string) ([]string, bool) {
	s, ok := Get(id)
	if !ok {
		return nil, false
	}
	return s.StageNames(), true
}

// StagePresets returns the legacy id -> stage-names map (GET /api/stage-presets).
func StagePresets() map[string][]string {
	out := map[string][]string{}
	for _, s := range All() {
		out[s.ID] = s.StageNames()
	}
	return out
}

func load() {
	byID = map[string]Species{}
	if dir := diskDir(); dir != "" {
		if sp, err := loadTree(os.DirFS(dir)); err != nil {
			log.Printf("species: reading %s: %v", dir, err)
		} else if len(sp) > 0 {
			set(sp)
			return
		}
	}
	if sub, err := fs.Sub(data, "data"); err == nil {
		if sp, err := loadTree(sub); err != nil {
			log.Printf("species: reading embedded tree: %v", err)
		} else {
			set(sp)
			return
		}
	}
	loaded = []Species{}
}

func set(sp []Species) {
	loaded = sp
	for _, s := range sp {
		byID[s.ID] = s
	}
}

// diskDir locates the repo-root species/ directory, or "" if not found.
func diskDir() string {
	if d := os.Getenv("GROWCORE_SPECIES_DIR"); d != "" {
		return d
	}
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}
	for i := 0; i < 8; i++ {
		cand := filepath.Join(dir, "species")
		if isSpeciesDir(cand) {
			return cand
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return ""
}

// isSpeciesDir reports whether p holds at least one <id>/species.yaml.
func isSpeciesDir(p string) bool {
	entries, err := os.ReadDir(p)
	if err != nil {
		return false
	}
	for _, e := range entries {
		if e.IsDir() {
			if _, err := os.Stat(filepath.Join(p, e.Name(), "species.yaml")); err == nil {
				return true
			}
		}
	}
	return false
}

func loadTree(fsys fs.FS) ([]Species, error) {
	entries, err := fs.ReadDir(fsys, ".")
	if err != nil {
		return nil, err
	}
	var out []Species
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		path := e.Name() + "/species.yaml"
		raw, err := fs.ReadFile(fsys, path)
		if err != nil {
			continue // directory without a species.yaml is skipped
		}
		var sp Species
		if err := yaml.Unmarshal(raw, &sp); err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		}
		sp.ID = e.Name()
		if sp.Label == "" {
			sp.Label = strings.Title(e.Name())
		}
		out = append(out, sp)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}
