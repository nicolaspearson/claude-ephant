package dto

// MiningUnit is keyed by the session and time of the event opening it. Reviewers and Rereviews are set for review cycles only.
type MiningUnit struct {
	Digest    string `json:"digest,omitempty"`
	Owner     string `json:"owner,omitempty"`
	Project   string `json:"project,omitempty"`
	Rereviews int    `json:"rereviews,omitempty"`
	Reviewers int    `json:"reviewers,omitempty"`
	Session   string `json:"session"`
	Source    string `json:"-"`
	Timestamp string `json:"timestamp"`
	Until     string `json:"-"` // next cycle's start in its session: for a cycle, covered or not; for a run, after the one it covers
}
