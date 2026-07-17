package domain

import "time"

// GrowPhoto is one photo attached to a grow (optionally to a specific plant
// unit). The image bytes live on the filesystem under the data directory
// (grows/<growID>/<file>); only this metadata is stored in the database. File is
// the content-addressed basename ("<sha256>.<ext>"), so identical uploads share
// one file on disk.
type GrowPhoto struct {
	ID          string    `json:"id"`
	GrowID      string    `json:"growId"`
	PlantUnitID string    `json:"plantUnitId,omitempty"`
	Caption     string    `json:"caption,omitempty"`
	TakenAt     time.Time `json:"takenAt"`
	File        string    `json:"-"` // on-disk basename; not exposed to clients
	ImageType   string    `json:"imageType"`
	CreatedAt   time.Time `json:"createdAt"`
}
