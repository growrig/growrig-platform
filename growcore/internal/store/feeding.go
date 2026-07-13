package store

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/growrig/growrig/growcore/internal/domain"
)

// User feeding recipes are stored as one YAML file per recipe under
// <dataDir>/feedings/<id>.yaml, so they live on the user's filesystem next to
// their environment configs — portable, hand-editable, and versionable. This
// mirrors how environments persist (see yaml.go). Built-in recipe templates are a
// separate, read-only catalog under species/<id>/feedings.yaml (see
// internal/feeding) and are only used as templates when creating a user recipe.

// feedingDoc is the on-disk shape of a user recipe file. It carries every field
// (unlike domain.FeedingRecipe's YAML tags, which are tuned for the built-in
// catalog where species is the directory name and there is no created time).
type feedingDoc struct {
	Version     int                     `yaml:"version"`
	ID          string                  `yaml:"id"`
	Species     string                  `yaml:"species"`
	Name        string                  `yaml:"name"`
	Brand       string                  `yaml:"brand,omitempty"`
	Description string                  `yaml:"description,omitempty"`
	Unit        string                  `yaml:"unit"`
	CreatedAt   time.Time               `yaml:"createdAt"`
	Products    []domain.FeedingProduct `yaml:"products"`
	Phases      []domain.FeedingPhase   `yaml:"phases"`
}

func (d feedingDoc) toRecipe() domain.FeedingRecipe {
	return domain.FeedingRecipe{
		ID:          d.ID,
		Species:     d.Species,
		Name:        d.Name,
		Brand:       d.Brand,
		Description: d.Description,
		Source:      "user",
		Unit:        d.Unit,
		Products:    d.Products,
		Phases:      d.Phases,
		CreatedAt:   d.CreatedAt,
	}
}

func (s *Store) feedingPath(id string) string {
	return filepath.Join(s.feedingDir, id+".yaml")
}

// FeedingRecipes returns all user recipes, newest first.
func (s *Store) FeedingRecipes() ([]domain.FeedingRecipe, error) {
	paths, err := filepath.Glob(filepath.Join(s.feedingDir, "*.yaml"))
	if err != nil {
		return nil, err
	}
	out := make([]domain.FeedingRecipe, 0, len(paths))
	for _, p := range paths {
		raw, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		var doc feedingDoc
		if err := yaml.Unmarshal(raw, &doc); err != nil {
			continue // skip a corrupt file rather than fail the whole list
		}
		if doc.ID == "" {
			// Fall back to the filename stem so hand-created files still load.
			doc.ID = filepath.Base(p[:len(p)-len(".yaml")])
		}
		out = append(out, doc.toRecipe())
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	return out, nil
}

// FeedingRecipe returns one user recipe by id.
func (s *Store) FeedingRecipe(id string) (domain.FeedingRecipe, bool, error) {
	raw, err := os.ReadFile(s.feedingPath(id))
	if os.IsNotExist(err) {
		return domain.FeedingRecipe{}, false, nil
	}
	if err != nil {
		return domain.FeedingRecipe{}, false, err
	}
	var doc feedingDoc
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		return domain.FeedingRecipe{}, false, fmt.Errorf("invalid recipe YAML: %w", err)
	}
	if doc.ID == "" {
		doc.ID = id
	}
	return doc.toRecipe(), true, nil
}

// SaveFeedingRecipe writes a user recipe atomically to its YAML file.
func (s *Store) SaveFeedingRecipe(p domain.FeedingRecipe) error {
	if p.ID == "" {
		return fmt.Errorf("recipe id is required")
	}
	doc := feedingDoc{
		Version:     1,
		ID:          p.ID,
		Species:     p.Species,
		Name:        p.Name,
		Brand:       p.Brand,
		Description: p.Description,
		Unit:        p.Unit,
		CreatedAt:   p.CreatedAt,
		Products:    p.Products,
		Phases:      p.Phases,
	}
	raw, err := yaml.Marshal(doc)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(s.feedingDir, 0o755); err != nil {
		return err
	}
	tmp := filepath.Join(s.feedingDir, "."+p.ID+".yaml.tmp")
	if err := os.WriteFile(tmp, raw, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, s.feedingPath(p.ID))
}

// DeleteFeedingRecipe removes a user recipe file.
func (s *Store) DeleteFeedingRecipe(id string) error {
	err := os.Remove(s.feedingPath(id))
	if os.IsNotExist(err) {
		return nil
	}
	return err
}
