package api

import (
	"encoding/base64"
	"encoding/json"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/growrig/growrig/growcore/internal/domain"
	"github.com/growrig/growrig/growcore/internal/store"
)

// Exercises the on-disk photo lifecycle directly against the handlers (no auth
// middleware): upload writes a content-addressed file, list/image read it, and
// delete removes the last-referenced file.
func TestGrowPhotoLifecycle(t *testing.T) {
	st, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	if err := st.SaveGrow(domain.Grow{ID: "grow-1", Name: "G", Species: "cannabis", Stage: "seedling", Stages: []string{"seedling"}, Status: domain.GrowActive}); err != nil {
		t.Fatal(err)
	}
	mediaDir := t.TempDir()
	s := &Server{store: st, growMediaDir: mediaDir}

	payload := base64.StdEncoding.EncodeToString([]byte("PNGDATA"))
	body := `{"image":"data:image/png;base64,` + payload + `","caption":"week 1"}`
	req := httptest.NewRequest("POST", "/api/grows/grow-1/photos", strings.NewReader(body))
	req.SetPathValue("id", "grow-1")
	rec := httptest.NewRecorder()
	s.uploadGrowPhoto(rec, req)
	if rec.Code != 201 {
		t.Fatalf("upload: expected 201, got %d (%s)", rec.Code, rec.Body.String())
	}
	var created domain.GrowPhoto
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}

	// The file exists under grows/<growID>/.
	entries, _ := os.ReadDir(filepath.Join(mediaDir, "grow-1"))
	if len(entries) != 1 {
		t.Fatalf("expected 1 file written, got %d", len(entries))
	}

	// Image serves with the stored content type.
	ireq := httptest.NewRequest("GET", "/", nil)
	ireq.SetPathValue("id", "grow-1")
	ireq.SetPathValue("photoId", created.ID)
	irec := httptest.NewRecorder()
	s.getGrowPhotoImage(irec, ireq)
	if irec.Code != 200 || irec.Body.String() != "PNGDATA" {
		t.Fatalf("image: code=%d body=%q", irec.Code, irec.Body.String())
	}

	// Delete removes the row and (last ref) the file.
	dreq := httptest.NewRequest("DELETE", "/", nil)
	dreq.SetPathValue("id", "grow-1")
	dreq.SetPathValue("photoId", created.ID)
	drec := httptest.NewRecorder()
	s.deleteGrowPhoto(drec, dreq)
	if drec.Code != 204 {
		t.Fatalf("delete: expected 204, got %d", drec.Code)
	}
	entries, _ = os.ReadDir(filepath.Join(mediaDir, "grow-1"))
	if len(entries) != 0 {
		t.Fatalf("expected file unlinked, got %d entries", len(entries))
	}
}
