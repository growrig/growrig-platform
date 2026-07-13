package inventory

import (
	"fmt"
	"io/fs"
	"log"
	"path"
	"sort"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
)

// A Product is a built-in template for a common item in a category (e.g. a
// specific nutrient bottle). Picking one in the item form pre-fills the item's
// name, unit and category columns, and binds the item to the product by ID so
// it can later reflect updated definitions. Products are read-only catalog data
// defined alongside their category:
//
//	inventory/<category-id>/products.yaml   (a `products:` list)
//	inventory/<category-id>/<image>         (optional product images)
//
// The loader reads from the same source tree the category catalog uses (see
// SourceFS): the on-disk inventory/ directory in development, or the copy
// embedded into the binary in production — mirroring internal/feeding.
type Product struct {
	// ID is fully-qualified as "<category>/<yaml-id>" so it is unique across
	// categories and self-describing when stored on an item.
	ID          string `json:"id"`
	Category    string `json:"category"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Unit        string `json:"unit,omitempty"`
	// Variants are the pack sizes this product is sold in. When present, the
	// item form offers them as a size picker instead of a free unit field.
	Variants   []Variant         `json:"variants,omitempty"`
	Attributes map[string]string `json:"attributes,omitempty"`
	// HasImage reports whether the product declares an image file that exists in
	// the source tree; the bytes are served separately.
	HasImage bool `json:"hasImage"`
	// image is the image path relative to the category directory (unexported;
	// resolved by ProductImage).
	image string
}

// Variant is one pack size of a product, with an optional product code (SKU or
// barcode — typically distinct per size).
type Variant struct {
	Size string `json:"size" yaml:"size"`
	Code string `json:"code,omitempty" yaml:"code,omitempty"`
}

// productFile is the on-disk shape of inventory/<category>/products.yaml.
type productFile struct {
	Products []struct {
		ID          string            `yaml:"id"`
		Name        string            `yaml:"name"`
		Description string            `yaml:"description"`
		Unit        string            `yaml:"unit"`
		Variants    []Variant         `yaml:"variants"`
		Image       string            `yaml:"image"`
		Attributes  map[string]string `yaml:"attributes"`
	} `yaml:"products"`
}

var (
	productsOnce sync.Once
	products     []Product
	productsByID map[string]Product
)

// Products returns every built-in product, loaded once, sorted by id.
func Products() []Product {
	productsOnce.Do(loadProducts)
	return products
}

// ProductsForCategory returns the built-in products of one category.
func ProductsForCategory(categoryID string) []Product {
	productsOnce.Do(loadProducts)
	id := strings.ToLower(strings.TrimSpace(categoryID))
	var out []Product
	for _, p := range products {
		if p.Category == id {
			out = append(out, p)
		}
	}
	return out
}

// Product returns the product with the given fully-qualified id
// ("<category>/<yaml-id>"), or false if there is none.
func GetProduct(id string) (Product, bool) {
	productsOnce.Do(loadProducts)
	p, ok := productsByID[strings.TrimSpace(id)]
	return p, ok
}

// ProductImage returns the image bytes and a best-effort MIME type for a
// product, or ok=false when the product has no image.
func ProductImage(id string) (data []byte, mime string, ok bool) {
	p, found := GetProduct(id)
	if !found || p.image == "" {
		return nil, "", false
	}
	src := SourceFS()
	if src == nil {
		return nil, "", false
	}
	raw, err := fs.ReadFile(src, path.Join(p.Category, p.image))
	if err != nil {
		return nil, "", false
	}
	return raw, mimeFromExt(p.image), true
}

func loadProducts() {
	productsByID = map[string]Product{}
	products = []Product{}
	src := SourceFS()
	if src == nil {
		return
	}
	entries, err := fs.ReadDir(src, ".")
	if err != nil {
		log.Printf("inventory: reading tree for products: %v", err)
		return
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		categoryID := e.Name()
		p := categoryID + "/products.yaml"
		raw, err := fs.ReadFile(src, p)
		if err != nil {
			continue // category without a products.yaml is fine
		}
		var f productFile
		if err := yaml.Unmarshal(raw, &f); err != nil {
			log.Printf("inventory: %s: %v", p, err)
			continue
		}
		for _, pr := range f.Products {
			if pr.ID == "" || pr.Name == "" {
				continue
			}
			prod := Product{
				ID:          fmt.Sprintf("%s/%s", categoryID, pr.ID),
				Category:    categoryID,
				Name:        pr.Name,
				Description: pr.Description,
				Unit:        pr.Unit,
				Variants:    pr.Variants,
				Attributes:  pr.Attributes,
				image:       pr.Image,
			}
			if pr.Image != "" {
				if _, err := fs.Stat(src, path.Join(categoryID, pr.Image)); err == nil {
					prod.HasImage = true
				} else {
					prod.image = "" // declared but missing; treat as imageless
				}
			}
			products = append(products, prod)
			productsByID[prod.ID] = prod
		}
	}
	sort.Slice(products, func(i, j int) bool { return products[i].ID < products[j].ID })
}

func mimeFromExt(name string) string {
	switch strings.ToLower(path.Ext(name)) {
	case ".png":
		return "image/png"
	case ".webp":
		return "image/webp"
	case ".gif":
		return "image/gif"
	case ".svg":
		return "image/svg+xml"
	default:
		return "image/jpeg"
	}
}
