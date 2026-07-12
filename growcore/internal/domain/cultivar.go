package domain

import "time"

// Cultivar is a user-defined strain/variety within a species. Beyond the common
// fields (name, creator/breeder, description, optional image) it carries a set
// of species-specific attribute values keyed by the attribute keys declared in
// that species' definition (see internal/species). Keeping the extras in a
// generic map keeps the model data-driven: adding a field to a species' YAML
// schema needs no code or schema change here.
type Cultivar struct {
	ID          string            `json:"id"`
	Species     string            `json:"species"`
	Name        string            `json:"name"`
	Creator     string            `json:"creator"`
	Description string            `json:"description"`
	Attributes  map[string]string `json:"attributes"`
	// ImageType is the stored image's MIME type (e.g. "image/jpeg"), or empty
	// when the cultivar has no image. The bytes are fetched separately.
	ImageType string    `json:"imageType,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
}

// HasImage reports whether an image is stored for this cultivar.
func (c Cultivar) HasImage() bool { return c.ImageType != "" }
