package store

import (
	"testing"
	"time"

	"github.com/growrig/growrig/growcore/internal/domain"
)

func TestTaskCreateAndDue(t *testing.T) {
	st := open(t)
	past := time.Now().Add(-time.Hour)
	future := time.Now().Add(48 * time.Hour)

	overdue, err := st.CreateTask(domain.Task{Title: "Water now", ActionType: "water", DueAt: &past})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := st.CreateTask(domain.Task{Title: "Later", ActionType: "inspect", DueAt: &future}); err != nil {
		t.Fatal(err)
	}
	if _, err := st.CreateTask(domain.Task{Title: "Someday", ActionType: "inspect"}); err != nil {
		t.Fatal(err)
	}

	due, err := st.DueTasks(time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if len(due) != 1 || due[0].ID != overdue.ID {
		t.Fatalf("expected only the overdue task, got %+v", due)
	}
}

func TestTaskCompleteLinksCareEvent(t *testing.T) {
	st := open(t)
	task, err := st.CreateTask(domain.Task{Title: "Feed", ActionType: "feed"})
	if err != nil {
		t.Fatal(err)
	}

	if err := st.CompleteTask(task.ID, "care-xyz"); err != nil {
		t.Fatal(err)
	}
	got, err := st.Task(task.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != domain.TaskCompleted || got.CompletedCareEventID != "care-xyz" {
		t.Fatalf("expected completed task linked to care event, got %+v", got)
	}
	if got.CompletedAt == nil {
		t.Fatal("expected CompletedAt to be set")
	}

	// A second completion is a no-op (task is no longer open).
	if err := st.CompleteTask(task.ID, "care-other"); err != nil {
		t.Fatal(err)
	}
	got, _ = st.Task(task.ID)
	if got.CompletedCareEventID != "care-xyz" {
		t.Fatalf("completing a non-open task should be a no-op, got %+v", got)
	}
}
