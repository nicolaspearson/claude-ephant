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

// Specific formats run before the assignment catch-all, which skips values already redacted; PEM in particular, or a key's body leaks past it.
// Anthropic before openai, which shares "sk-"; jwt before bearer, so a JWT keeps its label.
// Groups keep and after survive the replacement.
var patterns = []pattern{
	{expression: regexp.MustCompile(`-----BEGIN [A-Z ]*PRIVATE KEY-----[\s\S]*?-----END [A-Z ]*PRIVATE KEY-----`), kind: "pem"},
	// Cut before its END line, as truncated tool output leaves it; 16 characters a line, so a short word after the key is not taken for one.
	{expression: regexp.MustCompile(`-----BEGIN [A-Z ]*PRIVATE KEY-----(?:(?:\r?\n|\\n| )[A-Za-z0-9+/]{16,}={0,2})*`), kind: "pem"},
	{expression: regexp.MustCompile(`\bsk-ant-[A-Za-z0-9_-]{20,}`), kind: "anthropic"},
	{expression: regexp.MustCompile(`\bsk-(?:proj-)?[A-Za-z0-9_-]{32,}`), kind: "openai"},
	{expression: regexp.MustCompile(`\b(?:gh[pousr]_[A-Za-z0-9]{36,}|github_pat_[A-Za-z0-9_]{60,})`), kind: "github"},
	{expression: regexp.MustCompile(`\bgl(?:cbt|dt|ft|oas|pat|ptt|rt)-[A-Za-z0-9_-]{20,}`), kind: "gitlab"},
	{expression: regexp.MustCompile(`\b(?:AKIA|ASIA)[0-9A-Z]{16}\b`), kind: "aws"},
	{expression: regexp.MustCompile(`\bxox[abposr]-[A-Za-z0-9-]{10,}`), kind: "slack"},
	{expression: regexp.MustCompile(`\b[rs]k_(?:live|test)_[A-Za-z0-9]{16,}`), kind: "stripe"},
	{expression: regexp.MustCompile(`\bAIza[0-9A-Za-z_-]{35}`), kind: "google"},
	{expression: regexp.MustCompile(`\bnpm_[A-Za-z0-9]{36}`), kind: "npm"},
	{expression: regexp.MustCompile(`\beyJ[A-Za-z0-9_-]{10,}\.eyJ[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]{10,}`), kind: "jwt"},
	// A digit or symbol in the token, so "Bearer authentication" stays prose.
	{expression: regexp.MustCompile(`(?P<keep>\bBearer[ \t]+)[A-Za-z0-9._~+/=-]*[0-9._~+/=-][A-Za-z0-9._~+/=-]*`), kind: "bearer"},
	// The password stops at ? and #, so host:port before an @ in the query is not taken for one.
	{expression: regexp.MustCompile(`(?P<keep>\b[A-Za-z][A-Za-z0-9+.-]*://[^\s:/@]*:)[^\s/@?#]+(?P<after>@)`), kind: "url"},
}

const secretNames = `SECRET|TOKEN|PASSWORD|PASSWD|PASSPHRASE|CREDENTIALS?|API[_-]?KEY|PRIVATE[_-]?KEY|ACCESS[_-]?KEY`

// JSON inside a quoted string escapes its quotes: \"password\":\"x\".
var assignment = regexp.MustCompile(`(?i)\b(?P<key>[A-Z0-9_-]*(?:` + secretNames + `)[A-Z0-9_-]*)(?:\\?["'])?\s*(?::=|==|[=:])\s*`)

// The byte before the flag is checked in code, so a valueless flag's trailing space still starts the next one.
var secretFlag = regexp.MustCompile(`(?i)(?P<name>--[A-Z0-9-]*(?:` + secretNames + `)[A-Z0-9-]*)[ \t]+`)

// A flag naming the secret, or where it lives, rather than holding it.
var flagReference = regexp.MustCompile(`(?i)(?:^--no-|-(?:arn|count|dir|env|file|id|name|path|ref|stdin)$)`)

// A backslash stays in the value, as in a shell escape or json.Marshal's &,
// unless it escapes a closing quote. A leading bracket group runs to its
// closing bracket, then on to the value's end, stopping at the list's comma or brace.
var unquoted = regexp.MustCompile(`^(?:\[[^\]\n]*\](?:[^\s"'\\,}\]]|\\[^"'])*|(?:[^\s"'\\]|\\[^"'])*)`)

var placeholder = regexp.MustCompile(`^\[redacted:[a-z]+\](?:[\s"',;)\]}]|\\|$)`)

// Plural counts (max_tokens: 4096) are not secrets; token: 123456 is.
var tokenCountKey = regexp.MustCompile(`(?i)tokens$`)

var digitsOnly = regexp.MustCompile(`^["']?[0-9]+["']?[,;)\]}]*$`)

func Redact(text string) string {
	for _, pattern := range patterns {
		text = pattern.expression.ReplaceAllString(text, "${keep}[redacted:"+pattern.kind+"]${after}")
	}
	return redactFlags(redactAssignments(text))
}

func redactAssignments(text string) string {
	var out strings.Builder
	cursor := 0
	for _, match := range assignment.FindAllStringSubmatchIndex(text, -1) {
		if match[0] < cursor {
			continue
		}
		end := assignmentValue(text, match[1])
		if end == match[1] || isCount(text[match[2]:match[3]], text[match[1]:end]) {
			continue
		}
		out.WriteString(text[cursor:match[1]])
		out.WriteString("[redacted:assignment]")
		cursor = end
	}
	out.WriteString(text[cursor:])
	return out.String()
}

func redactFlags(text string) string {
	var out strings.Builder
	cursor := 0
	for _, match := range secretFlag.FindAllStringSubmatchIndex(text, -1) {
		name := text[match[2]:match[3]]
		if match[0] < cursor || !flagStart(text, match[0]) || flagReference.MatchString(name) {
			continue
		}
		end := match[1]
		if !strings.HasPrefix(text[match[1]:], "-") {
			end = assignmentValue(text, match[1])
		}
		if end == match[1] || isCount(name, text[match[1]:end]) {
			continue
		}
		out.WriteString(text[cursor:match[1]])
		out.WriteString("[redacted:flag]")
		cursor = end
	}
	out.WriteString(text[cursor:])
	return out.String()
}

// After a JSON-escaped \n or \t too, as a command inside a JSON string carries one.
func flagStart(text string, at int) bool {
	if at == 0 || strings.ContainsRune(" \t\n\"'=(,;{[|&", rune(text[at-1])) {
		return true
	}
	return at >= 2 && text[at-2] == '\\' && (text[at-1] == 'n' || text[at-1] == 't')
}

func isCount(key string, value string) bool {
	return tokenCountKey.MatchString(key) && digitsOnly.MatchString(value)
}

// Returns start when there is no value to redact: empty or already redacted.
func assignmentValue(text string, start int) int {
	rest := text[start:]
	if placeholder.MatchString(rest) {
		return start
	}
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
	if body == "" || placeholder.MatchString(body) || strings.HasPrefix(body, quote) {
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
