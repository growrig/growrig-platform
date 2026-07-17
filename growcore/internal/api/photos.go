package api

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/growrig/growrig/growcore/internal/domain"
)

// Grow photos are stored on the filesystem under the data directory
// (grows/<growID>/<sha256>.<ext>) and referenced by metadata rows. Content
// addressing means identical uploads share one file; a file is unlinked only
// when its last referencing row is deleted.

func imageExt(mime string) string {
	switch mime {
	case "image/png":
		return ".png"
	case "image/webp":
		return ".webp"
	case "image/gif":
		return ".gif"
	default:
		return ".jpg"
	}
}

func (s *Server) growPhotoDir(growID string) string {
	return filepath.Join(s.growMediaDir, growID)
}

type createPhotoBody struct {
	Image       string `json:"image"` // data:<mime>;base64,<payload>
	Caption     string `json:"caption"`
	PlantUnitID string `json:"plantUnitId"`
	TakenAt     string `json:"takenAt"` // RFC3339 or YYYY-MM-DD; empty = now
}

func (s *Server) uploadGrowPhoto(w http.ResponseWriter, r *http.Request) {
	growID := r.PathValue("id")
	_, ok, err := s.store.Grow(growID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	if !ok {
		writeJSON(w, http.StatusNotFound, errBody("grow not found"))
		return
	}
	var b createPhotoBody
	if err := decode(r, &b); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	data, mime, valid := decodeDataURL(b.Image)
	if !valid {
		writeJSON(w, http.StatusBadRequest, errBody("image must be a base64 image data URL"))
		return
	}

	sum := sha256.Sum256(data)
	file := hex.EncodeToString(sum[:]) + imageExt(mime)
	dir := s.growPhotoDir(growID)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	path := filepath.Join(dir, file)
	if _, statErr := os.Stat(path); statErr != nil {
		// Write atomically: temp file in the same dir, then rename.
		tmp, err := os.CreateTemp(dir, ".upload-*")
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err)
			return
		}
		tmpName := tmp.Name()
		if _, err := tmp.Write(data); err != nil {
			tmp.Close()
			_ = os.Remove(tmpName)
			writeErr(w, http.StatusInternalServerError, err)
			return
		}
		if err := tmp.Close(); err != nil {
			_ = os.Remove(tmpName)
			writeErr(w, http.StatusInternalServerError, err)
			return
		}
		if err := os.Rename(tmpName, path); err != nil {
			_ = os.Remove(tmpName)
			writeErr(w, http.StatusInternalServerError, err)
			return
		}
	}

	photo := domain.GrowPhoto{
		GrowID:      growID,
		PlantUnitID: b.PlantUnitID,
		Caption:     strings.TrimSpace(b.Caption),
		File:        file,
		ImageType:   mime,
	}
	if b.TakenAt != "" {
		photo.TakenAt = parseDate(b.TakenAt)
	} else {
		photo.TakenAt = time.Now()
	}
	saved, err := s.store.AddGrowPhoto(photo)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	s.growActivity(growID, "", "info", "notice", "Added a photo")
	writeJSON(w, http.StatusCreated, saved)
}

func (s *Server) getGrowPhotos(w http.ResponseWriter, r *http.Request) {
	photos, err := s.store.GrowPhotos(r.PathValue("id"))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, photos)
}

func (s *Server) getGrowPhotoImage(w http.ResponseWriter, r *http.Request) {
	photo, ok, err := s.store.GrowPhoto(r.PathValue("photoId"))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	if !ok || photo.GrowID != r.PathValue("id") {
		http.NotFound(w, r)
		return
	}
	data, err := os.ReadFile(filepath.Join(s.growPhotoDir(photo.GrowID), photo.File))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	serveImage(w, r, data, photo.ImageType, true, "public, max-age=31536000, immutable")
}

func (s *Server) deleteGrowPhoto(w http.ResponseWriter, r *http.Request) {
	photo, ok, err := s.store.GrowPhoto(r.PathValue("photoId"))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	if !ok || photo.GrowID != r.PathValue("id") {
		http.NotFound(w, r)
		return
	}
	if err := s.store.DeleteGrowPhoto(photo.ID); err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	// Unlink the file only when no other metadata row references it.
	if n, err := s.store.PhotoFileRefCount(photo.GrowID, photo.File); err == nil && n == 0 {
		_ = os.Remove(filepath.Join(s.growPhotoDir(photo.GrowID), photo.File))
	}
	w.WriteHeader(http.StatusNoContent)
}
