package domain

import "time"

// StockLine is one pack size (variant) of an inventory item with its own
// on-hand quantity and optional low-stock threshold. An item carries a list of
// these, so a nutrient can be tracked as "3 × 1 L" and "1 × 5 L" at once, and a
// simple item is just a single line with a blank size.
type StockLine struct {
	Size       string  `json:"size"`
	Quantity   float64 `json:"quantity"`
	LowStockAt float64 `json:"lowStockAt,omitempty"`
}

// LowStock reports whether this line is at or below its low-stock threshold.
func (l StockLine) LowStock() bool { return l.LowStockAt > 0 && l.Quantity <= l.LowStockAt }

// InventoryItem is one stock record the grower owns, belonging to a category
// from the inventory catalog (see internal/inventory). It carries category-
// specific values keyed by the category's column keys (a generic map, mirroring
// how a Cultivar carries species attributes) and a list of size variants, each
// with its own quantity.
type InventoryItem struct {
	ID         string            `json:"id"`
	Category   string            `json:"category"`
	Name       string            `json:"name"`
	Variants   []StockLine       `json:"variants"`
	Location   string            `json:"location"`
	Status     InventoryStatus   `json:"status"`
	Notes      string            `json:"notes"`
	Attributes map[string]string `json:"attributes"`
	// ProductID binds the item to a built-in product template
	// ("<category>/<product-id>"), or is empty for a free-form item. The binding
	// lets the item reflect the template (image, description, variant codes) as
	// definitions evolve.
	ProductID string `json:"productId,omitempty"`
	// ImageType is the MIME type of a user-uploaded image, or empty when none.
	// The bytes are fetched separately; the UI falls back to the bound product's
	// image when the item has no image of its own.
	ImageType string    `json:"imageType,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// InventoryStatus is the lifecycle state of a stock record.
type InventoryStatus string

const (
	InventoryActive   InventoryStatus = "active"   // in stock / in use
	InventoryOrdered  InventoryStatus = "ordered"  // reordered, not yet arrived
	InventoryArchived InventoryStatus = "archived" // no longer tracked
)

// TotalQuantity sums the quantities across all size variants.
func (i InventoryItem) TotalQuantity() float64 {
	var sum float64
	for _, v := range i.Variants {
		sum += v.Quantity
	}
	return sum
}

// AnyLowStock reports whether any size variant is at or below its threshold.
func (i InventoryItem) AnyLowStock() bool {
	for _, v := range i.Variants {
		if v.LowStock() {
			return true
		}
	}
	return false
}
