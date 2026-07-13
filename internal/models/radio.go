package models

// RadioQueue is a shuffled list of tracks for radio playback.
type RadioQueue struct {
	Items           []Track `json:"items"`
	TotalAvailable  int     `json:"total_available"`
	Returned        int     `json:"returned"`
}
