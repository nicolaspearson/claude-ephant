package dto

import "encoding/json"

type Event struct {
	Attributes map[string]any `json:"attributes,omitempty"`
	Kind       string         `json:"kind"`
	Name       string         `json:"name,omitempty"`
	Owner      string         `json:"owner,omitempty"`
	Profile    string         `json:"profile,omitempty"`
	Project    string         `json:"project,omitempty"`
	Session    string         `json:"session"`
	Source     string         `json:"-"` // archive path of the transcript holding it, relative to archive/transcripts
	Subagent   bool           `json:"subagent,omitempty"`
	Timestamp  string         `json:"timestamp"`
}

func (e *Event) DecodeAttributes(encoded string) error {
	var attributes map[string]any
	if err := json.Unmarshal([]byte(encoded), &attributes); err != nil {
		return err
	}
	if len(attributes) > 0 {
		e.Attributes = attributes
	}
	return nil
}

// EventQuery matches everything where a field is empty, and returns every match where Limit is 0.
type EventQuery struct {
	During  string // a skill or command name: only events inside one of its runs
	Kind    string
	Limit   int
	Name    string
	Owner   string
	Project string
	Session string
	Since   string
}

type TranscriptEvent struct {
	Attributes       map[string]any
	Entrypoint       string
	Key              string // overrides the line's id and place as the event's key
	Kind             string
	Name             string
	SessionID        string
	Timestamp        string
	UUID             string
	WorkingDirectory string
}
