package store

import (
	"testing"
	"time"

	"github.com/growrig/growrig/growcore/internal/domain"
)

func TestGrowPhotoRoundTripAndRefCount(t *testing.T) {
	st := open(t)

	p1, err := st.AddGrowPhoto(domain.GrowPhoto{GrowID: "grow-1", File: "abc.jpg", ImageType: "image/jpeg", TakenAt: time.Now()})
	if err != nil {
		t.Fatal(err)
	}
	// A second row referencing the same content-addressed file (dedup case).
	_, err = st.AddGrowPhoto(domain.GrowPhoto{GrowID: "grow-1", File: "abc.jpg", ImageType: "image/jpeg", TakenAt: time.Now()})
	if err != nil {
		t.Fatal(err)
	}

	photos, err := st.GrowPhotos("grow-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(photos) != 2 {
		t.Fatalf("expected 2 photos, got %d", len(photos))
	}

	n, err := st.PhotoFileRefCount("grow-1", "abc.jpg")
	if err != nil || n != 2 {
		t.Fatalf("expected refcount 2, got %d (err=%v)", n, err)
	}

	if err := st.DeleteGrowPhoto(p1.ID); err != nil {
		t.Fatal(err)
	}
	n, _ = st.PhotoFileRefCount("grow-1", "abc.jpg")
	if n != 1 {
		t.Fatalf("expected refcount 1 after one delete, got %d", n)
	}
}

func TestStageEventsOrdered(t *testing.T) {
	st := open(t)
	base := time.Now()
	_ = st.AddStageEvent("grow-1", "seedling", base.Add(-48*time.Hour))
	_ = st.AddStageEvent("grow-1", "vegetative", base)

	events, err := st.StageEvents("grow-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 2 || events[0].Stage != "seedling" || events[1].Stage != "vegetative" {
		t.Fatalf("expected ordered [seedling, vegetative], got %+v", events)
	}
}
