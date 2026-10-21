package redact

import "regexp"

type pattern struct {
	expression  *regexp.Regexp
	kind        string
	replacement string
}

// Specific formats run before the assignment catch-all, which skips values already redacted; PEM in particular, or a key's body leaks past it. Anthropic before openai, which shares "sk-".
var patterns = []pattern{
	{expression: regexp.MustCompile(`-----BEGIN [A-Z ]*PRIVATE KEY-----[\s\S]*?-----END [A-Z ]*PRIVATE KEY-----`), kind: "pem"},
	{expression: regexp.MustCompile(`\bsk-ant-[A-Za-z0-9_-]{20,}`), kind: "anthropic"},
	{expression: regexp.MustCompile(`\bsk-(?:proj-)?[A-Za-z0-9_-]{32,}`), kind: "openai"},
	{expression: regexp.MustCompile(`\b(?:gh[pousr]_[A-Za-z0-9]{36,}|github_pat_[A-Za-z0-9_]{60,})`), kind: "github"},
	{expression: regexp.MustCompile(`\bglpat-[A-Za-z0-9_-]{20,}`), kind: "gitlab"},
	{expression: regexp.MustCompile(`\b(?:AKIA|ASIA)[0-9A-Z]{16}\b`), kind: "aws"},
	{expression: regexp.MustCompile(`\beyJ[A-Za-z0-9_-]{10,}\.eyJ[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]{10,}`), kind: "jwt"},
	{
		expression:  regexp.MustCompile(`(?i)\b([A-Z0-9_]*(?:SECRET|TOKEN|PASSWORD|PASSWD|API_?KEY|PRIVATE_?KEY|ACCESS_?KEY)[A-Z0-9_]*"?\s*[=:]\s*)(?:"[^"\n\[][^"\n]*"|'[^'\n\[][^'\n]*'|[^\s"'\[\n]+)`),
		kind:        "assignment",
		replacement: "${1}[redacted:assignment]",
	},
}

func Redact(text string) string {
	for _, pattern := range patterns {
		replacement := pattern.replacement
		if replacement == "" {
			replacement = "[redacted:" + pattern.kind + "]"
		}
		text = pattern.expression.ReplaceAllString(text, replacement)
	}
	return text
}
