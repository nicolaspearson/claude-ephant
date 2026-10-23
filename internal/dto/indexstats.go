package dto

type EntryCount struct {
	Entries int
	Kind    string
	Owner   string
	Source  string // notes or transcripts
}

type IndexStats struct {
	Counts   []EntryCount
	First    string
	Last     string
	Sessions map[string]int
}
