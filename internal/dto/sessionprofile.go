package dto

const SessionProfilesFile = "session-profiles.jsonl"

// SessionProfile is one line of the claude-mem export's SessionProfilesFile.
type SessionProfile struct {
	ContentSessionID string `json:"contentSessionId"`
	MemorySessionID  string `json:"memorySessionId"`
	Profile          string `json:"profile"` // empty when no transcript survives
	Project          string `json:"project"`
}
