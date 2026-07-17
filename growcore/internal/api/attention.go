package api

import (
	"net/http"
	"time"

	"github.com/growrig/growrig/growcore/internal/domain"
)

// The attention endpoint is a live projection of everything a grower may need to
// act on right now. It is intentionally not persisted (it would go stale) — each
// request recomputes it from the durable sources: open alerts, due/overdue
// tasks, low inventory, and unhealthy integrations. See the alerts/tasks stores
// for the "something is wrong" / "something should be done" split.

type lowStockItem struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	Category string  `json:"category"`
	Quantity float64 `json:"quantity"`
}

type unhealthyIntegration struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

type attentionResponse struct {
	Alerts       []domain.Alert         `json:"alerts"`
	Tasks        []domain.Task          `json:"tasks"`
	LowStock     []lowStockItem         `json:"lowStock"`
	Integrations []unhealthyIntegration `json:"integrations"`
}

func (s *Server) getAttention(w http.ResponseWriter, r *http.Request) {
	out := attentionResponse{
		Alerts:       []domain.Alert{},
		Tasks:        []domain.Task{},
		LowStock:     []lowStockItem{},
		Integrations: []unhealthyIntegration{},
	}

	alerts, err := s.store.OpenAlerts()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	out.Alerts = alerts

	// Overdue and due-today tasks: everything open with a due date up to the end
	// of the local day.
	endOfToday := time.Now().Truncate(24 * time.Hour).Add(24 * time.Hour)
	tasks, err := s.store.DueTasks(endOfToday)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	out.Tasks = tasks

	items, err := s.store.InventoryItems("")
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	for _, it := range items {
		if it.Status == domain.InventoryArchived || !it.AnyLowStock() {
			continue
		}
		out.LowStock = append(out.LowStock, lowStockItem{
			ID: it.ID, Name: it.Name, Category: it.Category, Quantity: it.TotalQuantity(),
		})
	}

	recs, err := s.store.IntegrationInstances()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	for _, rec := range recs {
		if !rec.Instance.Enabled {
			continue
		}
		switch rec.Instance.Status {
		case "ok", "healthy", "connected", "":
			continue
		}
		out.Integrations = append(out.Integrations, unhealthyIntegration{
			ID: rec.Instance.ID, Name: rec.Instance.Name,
			Status: rec.Instance.Status, Message: rec.Instance.StatusMessage,
		})
	}

	writeJSON(w, http.StatusOK, out)
}
