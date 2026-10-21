package dto

type Usage struct {
	Injections      []UsageCount
	MainTokens      UsageTokens
	Prompts         int
	Replies         int
	ReplyCharacters int
	Sessions        int
	SubagentTokens  UsageTokens
	Tools           []UsageCount
}

// UsageCount splits a total between the main thread and subagents: calls for a tool, bytes for an injection.
type UsageCount struct {
	Main     int
	Name     string
	Subagent int
}

// UsageTokens as the API reports them: Input excludes cache reads and writes.
type UsageTokens struct {
	CacheRead  int
	CacheWrite int
	Input      int
	Output     int
}

// UsageQuery matches everything where a field is empty.
type UsageQuery struct {
	Owner   string
	Project string
	Since   string
}
