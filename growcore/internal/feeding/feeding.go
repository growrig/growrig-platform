// Package feeding is Grow Core's built-in database of nutrient feeding
// schedules (e.g. the BioBizz chart): which products to dose, how much, per
// week, across the phases of a grow.
//
// Built-in presets live alongside their species as YAML:
//
//	species/<species-id>/feedings.yaml
//
// where the file holds a `presets:` list. The species id is the directory
// name. These presets are read-only; users create their own editable presets
// in-app (stored in the DB). The loader reads from the same source tree the
// species catalog uses (see species.SourceFS): the on-disk species/ directory
// in development, or the copy embedded into the binary in production — so no
// separate embed or Makefile wiring is needed.
package feeding

import (
	"fmt"
	"io/fs"
	"log"
	"sort"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"

	"github.com/growrig/growrig-platform/growcore/internal/domain"
	"github.com/growrig/growrig-platform/growcore/internal/species"
)

// file is the on-disk shape of species/<id>/feedings.yaml.
type file struct {
	Presets []domain.FeedingPreset `yaml:"presets"`
}

var (
	once   sync.Once
	loaded []domain.FeedingPreset
	byID   map[string]domain.FeedingPreset
)

// All returns every built-in preset, loaded once, sorted by species then id.
func All() []domain.FeedingPreset {
	once.Do(load)
	return loaded
}

// BySpecies returns the built-in presets for one species (case-insensitive).
func BySpecies(speciesID string) []domain.FeedingPreset {
	once.Do(load)
	id := strings.ToLower(strings.TrimSpace(speciesID))
	var out []domain.FeedingPreset
	for _, p := range loaded {
		if p.Species == id {
			out = append(out, p)
		}
	}
	return out
}

// Get returns the built-in preset with the given fully-qualified id
// ("<species>/<yaml-id>"), or false if there is none.
func Get(id string) (domain.FeedingPreset, bool) {
	once.Do(load)
	p, ok := byID[id]
	return p, ok
}

func load() {
	byID = map[string]domain.FeedingPreset{}
	loaded = []domain.FeedingPreset{}
	src := species.SourceFS()
	if src == nil {
		return
	}
	entries, err := fs.ReadDir(src, ".")
	if err != nil {
		log.Printf("feeding: reading species tree: %v", err)
		return
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		speciesID := e.Name()
		path := speciesID + "/feedings.yaml"
		raw, err := fs.ReadFile(src, path)
		if err != nil {
			continue // species without a feedings.yaml is fine
		}
		var f file
		if err := yaml.Unmarshal(raw, &f); err != nil {
			log.Printf("feeding: %s: %v", path, err)
			continue
		}
		for _, p := range f.Presets {
			p.Species = speciesID
			p.Source = "builtin"
			p.ID = fmt.Sprintf("%s/%s", speciesID, p.ID)
			loaded = append(loaded, p)
			byID[p.ID] = p
		}
	}
	sort.Slice(loaded, func(i, j int) bool { return loaded[i].ID < loaded[j].ID })
}
