package api

import (
	"net/http"
	"strings"
	"time"

	"github.com/growrig/growrig/growcore/internal/domain"
	"github.com/growrig/growrig/growcore/internal/inventory"
)

// --- Inventory catalog (categories + their extra columns) ---

func (s *Server) getInventoryCategories(w http.ResponseWriter, r *http.Request) {
	cats := inventory.All()
	if cats == nil {
		cats = []inventory.Category{}
	}
	writeJSON(w, http.StatusOK, cats)
}

// getInventoryProducts returns built-in product templates, optionally filtered
// to one category. They seed a new item and bind it to the product.
func (s *Server) getInventoryProducts(w http.ResponseWriter, r *http.Request) {
	cat := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("category")))
	var prods []inventory.Product
	if cat != "" {
		prods = inventory.ProductsForCategory(cat)
	} else {
		prods = inventory.Products()
	}
	if prods == nil {
		prods = []inventory.Product{}
	}
	writeJSON(w, http.StatusOK, prods)
}

// getInventoryProductImage serves a built-in product's image. The product id is
// fully-qualified as "<category>/<id>" — both path segments recombine to it.
func (s *Server) getInventoryProductImage(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("category") + "/" + r.PathValue("id")
	data, mime, ok := inventory.ProductImage(id)
	if !ok {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", mime)
	w.Header().Set("Cache-Control", "no-cache")
	_, _ = w.Write(data)
}

// --- Inventory items ---

func (s *Server) getInventoryItems(w http.ResponseWriter, r *http.Request) {
	cat := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("category")))
	items, err := s.store.InventoryItems(cat)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	if items == nil {
		items = []domain.InventoryItem{}
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *Server) getInventoryItem(w http.ResponseWriter, r *http.Request) {
	it, ok, err := s.store.InventoryItem(r.PathValue("id"))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	if !ok {
		writeJSON(w, http.StatusNotFound, errBody("inventory item not found"))
		return
	}
	writeJSON(w, http.StatusOK, it)
}

type inventoryItemBody struct {
	Category   string             `json:"category"`
	Name       string             `json:"name"`
	Variants   []domain.StockLine `json:"variants"`
	Location   string             `json:"location"`
	Status     string             `json:"status"`
	Notes      string             `json:"notes"`
	Attributes map[string]string  `json:"attributes"`
	// ProductID binds the item to a built-in product template; ignored if it
	// doesn't resolve to a product in the item's category.
	ProductID string `json:"productId"`
	// Image is an optional data URL ("data:image/png;base64,…"). Empty leaves the
	// existing image unchanged on update; RemoveImage explicitly clears it.
	Image       string `json:"image"`
	RemoveImage bool   `json:"removeImage"`
}

// boundProductID validates a product binding against the category and returns
// the fully-qualified id to store, or "" when there is no valid binding.
func boundProductID(cat inventory.Category, productID string) string {
	productID = strings.TrimSpace(productID)
	if productID == "" {
		return ""
	}
	p, ok := inventory.GetProduct(productID)
	if !ok || p.Category != cat.ID {
		return ""
	}
	return p.ID
}

// sanitizeColumns keeps only the column keys declared by the category and, for
// enum columns, only values present in the declared options.
func sanitizeColumns(cat inventory.Category, in map[string]string) map[string]string {
	out := map[string]string{}
	for _, col := range cat.Columns {
		v := strings.TrimSpace(in[col.Key])
		if v == "" {
			continue
		}
		if col.Type == inventory.ColEnum && len(col.Options) > 0 && !containsStr(col.Options, v) {
			continue
		}
		out[col.Key] = v
	}
	return out
}

// sanitizeVariants trims sizes, drops entirely-empty rows, and clamps negative
// quantities. An item may legitimately have zero variants.
func sanitizeVariants(in []domain.StockLine) []domain.StockLine {
	out := []domain.StockLine{}
	for _, v := range in {
		v.Size = strings.TrimSpace(v.Size)
		if v.Size == "" && v.Quantity == 0 && v.LowStockAt == 0 {
			continue
		}
		if v.Quantity < 0 {
			v.Quantity = 0
		}
		if v.LowStockAt < 0 {
			v.LowStockAt = 0
		}
		out = append(out, v)
	}
	return out
}

func inventoryStatus(v string) domain.InventoryStatus {
	switch domain.InventoryStatus(strings.TrimSpace(v)) {
	case domain.InventoryOrdered:
		return domain.InventoryOrdered
	case domain.InventoryArchived:
		return domain.InventoryArchived
	default:
		return domain.InventoryActive
	}
}

func (s *Server) createInventoryItem(w http.ResponseWriter, r *http.Request) {
	var b inventoryItemBody
	if err := decode(r, &b); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	if strings.TrimSpace(b.Name) == "" {
		writeJSON(w, http.StatusBadRequest, errBody("name is required"))
		return
	}
	cat, ok := inventory.Get(b.Category)
	if !ok {
		writeJSON(w, http.StatusBadRequest, errBody("category must be one of the inventory categories"))
		return
	}
	now := time.Now()
	it := domain.InventoryItem{
		ID:         id(b.Name, "item"),
		Category:   cat.ID,
		Name:       strings.TrimSpace(b.Name),
		Variants:   sanitizeVariants(b.Variants),
		Location:   strings.TrimSpace(b.Location),
		Status:     inventoryStatus(b.Status),
		Notes:      strings.TrimSpace(b.Notes),
		Attributes: sanitizeColumns(cat, b.Attributes),
		ProductID:  boundProductID(cat, b.ProductID),
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	if err := s.store.SaveInventoryItem(it); err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	if data, mime, ok := decodeDataURL(b.Image); ok {
		if err := s.store.SetInventoryItemImage(it.ID, data, mime); err != nil {
			writeErr(w, http.StatusInternalServerError, err)
			return
		}
		it.ImageType = mime
	}
	s.activity("", "", "info", "configuration", "Added inventory item "+it.Name)
	writeJSON(w, http.StatusOK, it)
}

func (s *Server) updateInventoryItem(w http.ResponseWriter, r *http.Request) {
	it, ok, err := s.store.InventoryItem(r.PathValue("id"))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	if !ok {
		writeJSON(w, http.StatusNotFound, errBody("inventory item not found"))
		return
	}
	var b inventoryItemBody
	if err := decode(r, &b); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	if strings.TrimSpace(b.Name) == "" {
		writeJSON(w, http.StatusBadRequest, errBody("name is required"))
		return
	}
	cat, ok := inventory.Get(b.Category)
	if !ok {
		writeJSON(w, http.StatusBadRequest, errBody("category must be one of the inventory categories"))
		return
	}
	it.Category = cat.ID
	it.Name = strings.TrimSpace(b.Name)
	it.Variants = sanitizeVariants(b.Variants)
	it.Location = strings.TrimSpace(b.Location)
	it.Status = inventoryStatus(b.Status)
	it.Notes = strings.TrimSpace(b.Notes)
	it.Attributes = sanitizeColumns(cat, b.Attributes)
	it.ProductID = boundProductID(cat, b.ProductID)
	it.UpdatedAt = time.Now()
	if err := s.store.SaveInventoryItem(it); err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	switch {
	case b.RemoveImage:
		if err := s.store.ClearInventoryItemImage(it.ID); err != nil {
			writeErr(w, http.StatusInternalServerError, err)
			return
		}
		it.ImageType = ""
	default:
		if data, mime, ok := decodeDataURL(b.Image); ok {
			if err := s.store.SetInventoryItemImage(it.ID, data, mime); err != nil {
				writeErr(w, http.StatusInternalServerError, err)
				return
			}
			it.ImageType = mime
		}
	}
	writeJSON(w, http.StatusOK, it)
}

func (s *Server) getInventoryItemImage(w http.ResponseWriter, r *http.Request) {
	data, mime, ok, err := s.store.InventoryItemImage(r.PathValue("id"))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	if !ok {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", mime)
	w.Header().Set("Cache-Control", "no-cache")
	_, _ = w.Write(data)
}

func (s *Server) deleteInventoryItem(w http.ResponseWriter, r *http.Request) {
	if err := s.store.DeleteInventoryItem(r.PathValue("id")); err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
