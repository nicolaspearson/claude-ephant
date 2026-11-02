package redact

import (
	"regexp"
	"strings"
)

type pattern struct {
	expression *regexp.Regexp
	kind       string
}

// A quoted value runs to its closing quote within this many bytes, newlines
// included: a heredoc or a JSON body can hold one.
const quotedLimit = 4096

// Specific formats run before the assignment catch-all, which skips values already redacted; PEM in particular, or a key's body leaks past it. Anthropic before openai, which shares "sk-".
var patterns = []pattern{
	{expression: regexp.MustCompile(`-----BEGIN [A-Z ]*PRIVATE KEY-----[\s\S]*?-----END [A-Z ]*PRIVATE KEY-----`), kind: "pem"},
	{expression: regexp.MustCompile(`\bsk-ant-[A-Za-z0-9_-]{20,}`), kind: "anthropic"},
	{expression: regexp.MustCompile(`\bsk-(?:proj-)?[A-Za-z0-9_-]{32,}`), kind: "openai"},
	{expression: regexp.MustCompile(`\b(?:gh[pousr]_[A-Za-z0-9]{36,}|github_pat_[A-Za-z0-9_]{60,})`), kind: "github"},
	{expression: regexp.MustCompile(`\bgl(?:cbt|dt|ft|oas|pat|ptt|rt)-[A-Za-z0-9_-]{20,}`), kind: "gitlab"},
	{expression: regexp.MustCompile(`\b(?:AKIA|ASIA)[0-9A-Z]{16}\b`), kind: "aws"},
	{expression: regexp.MustCompile(`\beyJ[A-Za-z0-9_-]{10,}\.eyJ[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]{10,}`), kind: "jwt"},
}

// JSON inside a quoted string escapes its quotes: \"password\":\"x\".
var assignment = regexp.MustCompile(`(?i)\b[A-Z0-9_-]*(?:SECRET|TOKEN|PASSWORD|PASSWD|API[_-]?KEY|PRIVATE[_-]?KEY|ACCESS[_-]?KEY)[A-Z0-9_-]*(?:\\?["'])?\s*(?::=|==|[=:])\s*`)

// A backslash stays in the value, as in a shell escape or json.Marshal's &,
// unless it escapes a closing quote.
var unquoted = regexp.MustCompile(`^(?:[^\s"'\\\[]|\\[^"'])+`)

func Redact(text string) string {
	for _, pattern := range patterns {
		text = pattern.expression.ReplaceAllString(text, "[redacted:"+pattern.kind+"]")
	}
	return redactAssignments(text)
}

func redactAssignments(text string) string {
	var redacted strings.Builder
	cursor := 0
	for _, match := range assignment.FindAllStringIndex(text, -1) {
		if match[0] < cursor {
			continue
		}
		end := assignmentValue(text, match[1])
		if end == match[1] {
			continue
		}
		redacted.WriteString(text[cursor:match[1]])
		redacted.WriteString("[redacted:assignment]")
		cursor = end
	}
	redacted.WriteString(text[cursor:])
	return redacted.String()
}

// Returns start when there is no value to redact: empty or already redacted.
func assignmentValue(text string, start int) int {
	rest := text[start:]
	quote := ""
	for _, candidate := range []string{`\"`, `\'`, `"`, `'`} {
		if strings.HasPrefix(rest, candidate) {
			quote = candidate
			break
		}
	}
	if quote == "" {
		return start + len(unquoted.FindString(rest))
	}
	body := rest[len(quote):]
	if body == "" || strings.HasPrefix(body, "[") || strings.HasPrefix(body, quote) {
		return start
	}
	if closing := closingQuote(body[:min(len(body), quotedLimit)], quote); closing >= 0 {
		return start + len(quote) + closing + len(quote)
	}
	// Unclosed or past the limit: redact to the line's end.
	if newline := strings.IndexByte(body, '\n'); newline >= 0 {
		return start + len(quote) + newline
	}
	return len(text)
}

// A quote escaped with a backslash is part of the value, not its end.
func closingQuote(body string, quote string) int {
	for offset := 0; ; {
		found := strings.Index(body[offset:], quote)
		if found < 0 {
			return -1
		}
		at := offset + found
		if strings.HasPrefix(quote, `\`) || at == 0 || body[at-1] != '\\' {
			return at
		}
		offset = at + len(quote)
	}
}
