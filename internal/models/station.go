package models

// Station is a curated or auto-generated track collection for radio playback.
type Station struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Kind        string `json:"kind"`
	Description string `json:"description,omitempty"`
	TrackCount  int    `json:"track_count"`
}
