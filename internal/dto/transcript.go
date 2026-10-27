package dto

// Transcript is a main transcript in the archive. First is the earliest line the index keeps from it: a resumed
// transcript's copied lines are kept under the transcript indexed first, so its own start later.
type Transcript struct {
	First   string
	Owner   string
	Project string
	Session string
	Source  string // archive path, relative to archive/transcripts
}
