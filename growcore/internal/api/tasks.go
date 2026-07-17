package api

import (
	"net/http"
	"strings"
	"time"

	"github.com/growrig/growrig/growcore/internal/domain"
)

func (s *Server) getTasks(w http.ResponseWriter, r *http.Request) {
	status := domain.TaskStatus(r.URL.Query().Get("status"))
	tasks, err := s.store.ListTasks(status)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, tasks)
}

type createTaskBody struct {
	GrowID        string `json:"growId"`
	EnvironmentID string `json:"environmentId"`
	PlantUnitID   string `json:"plantUnitId"`
	ActionType    string `json:"actionType"`
	Title         string `json:"title"`
	DueAt         string `json:"dueAt"` // RFC3339 or YYYY-MM-DD; empty = no due date
	Source        string `json:"source"`
}

func (s *Server) createTask(w http.ResponseWriter, r *http.Request) {
	var b createTaskBody
	if err := decode(r, &b); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	title := strings.TrimSpace(b.Title)
	if title == "" {
		writeJSON(w, http.StatusBadRequest, errBody("task title is required"))
		return
	}
	t := domain.Task{
		ID:            id(title, "task"),
		GrowID:        b.GrowID,
		EnvironmentID: b.EnvironmentID,
		PlantUnitID:   b.PlantUnitID,
		ActionType:    strings.TrimSpace(b.ActionType),
		Title:         title,
		Source:        domain.TaskSource(b.Source),
	}
	if b.DueAt != "" {
		due := parseDate(b.DueAt)
		t.DueAt = &due
	}
	created, err := s.store.CreateTask(t)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

// completeTask marks a task done and, when it targets a grow, records a care
// event for the action so completed planned work lands in the journal.
func (s *Server) completeTask(w http.ResponseWriter, r *http.Request) {
	task, err := s.store.Task(r.PathValue("id"))
	if err != nil {
		writeJSON(w, http.StatusNotFound, errBody("task not found"))
		return
	}
	if task.Status != domain.TaskOpen {
		writeJSON(w, http.StatusConflict, errBody("task is not open"))
		return
	}

	var careEventID string
	if task.GrowID != "" && task.ActionType != "" {
		event := domain.CareEvent{
			ID:         id(task.Title, "care"),
			GrowID:     task.GrowID,
			Type:       task.ActionType,
			OccurredAt: time.Now(),
			Source:     domain.CareManual,
			Notes:      "Completed task: " + task.Title,
		}
		// Target the task's plant unit, else broadcast to the grow's active plants.
		var ids []string
		if task.PlantUnitID != "" {
			ids = []string{task.PlantUnitID}
		} else {
			plants, err := s.growPlants(task.GrowID)
			if err != nil {
				writeErr(w, http.StatusInternalServerError, err)
				return
			}
			for _, u := range plants {
				if u.Status == domain.PlantActive {
					ids = append(ids, u.ID)
				}
			}
		}
		for _, pid := range ids {
			event.Applications = append(event.Applications, domain.CareApplication{
				ID: newCareApplicationID(), CareEventID: event.ID, PlantUnitID: pid,
			})
		}
		if err := s.store.SaveCareEvent(event); err != nil {
			writeErr(w, http.StatusInternalServerError, err)
			return
		}
		careEventID = event.ID
	}

	if err := s.store.CompleteTask(task.ID, careEventID); err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	updated, err := s.store.Task(task.ID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (s *Server) skipTask(w http.ResponseWriter, r *http.Request) {
	if err := s.store.SkipTask(r.PathValue("id")); err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func newCareApplicationID() string { return id("app", "app") }
