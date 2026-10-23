package highlight

import (
	"os"
	"regexp"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"
)

type style struct {
	close string
	open  string
}

// Never \033[0m: \033[39m is what restored the message's own colour in Claude Code, as of 2026-10-17.
var (
	bold    = style{close: "\033[22m", open: "\033[1m"}
	cyan    = style{close: "\033[39m", open: "\033[36m"}
	dim     = style{close: "\033[22m", open: "\033[2m"}
	green   = style{close: "\033[39m", open: "\033[32m"}
	magenta = style{close: "\033[39m", open: "\033[35m"}
	red     = style{close: "\033[39m", open: "\033[31m"}
	yellow  = style{close: "\033[39m", open: "\033[33m"}
)

type rule struct {
	accept     func(text string, end int) bool
	expression *regexp.Regexp
	// One per capture group, or one for the whole match when the expression has none.
	styles []func(match string) style
}

type span struct {
	end   int
	start int
	style style
}

func always(chosen style) func(string) style {
	return func(string) style { return chosen }
}

// /private/tmp is a path, not the command /private.
func commandEnds(text string, end int) bool {
	next, _ := utf8.DecodeRuneInString(text[end:])
	return end == len(text) || next != '/' && next != '_' && !unicode.IsLetter(next) && !unicode.IsDigit(next)
}

func count(chosen style) func(string) style {
	return func(match string) style {
		if strings.HasPrefix(match, "0 ") {
			return dim
		}
		return chosen
	}
}

// Earlier rules win an overlap.
var rules = []rule{
	{expression: regexp.MustCompile(`(?m)^ele(?::| could not) [^;\n]*|plan \S+ is not in this checkout`), styles: []func(string) style{always(yellow)}},
	{expression: regexp.MustCompile("`[^`\n]+`"), styles: []func(string) style{always(cyan)}},
	{expression: regexp.MustCompile(`(\d+ open), (\d+ blocked), (\d+ done), (\d+ in the inbox)`), styles: []func(string) style{count(yellow), count(red), count(green), count(cyan)}},
	{expression: regexp.MustCompile(`\b(\d+ warnings?); ele plan check`), styles: []func(string) style{always(yellow)}},
	{expression: regexp.MustCompile(`(?m)^(?:Handoff from ele|\d+ handoffs from ele|Active plan from ele:|\d+ active plans from ele|Reminders from ele)`), styles: []func(string) style{always(bold)}},
	{expression: regexp.MustCompile(`(?m)^- (?:(↻) )?due (\d{4}-\d{2}-\d{2}), reminder \d{8}-[0-9a-f]{6}(?:, (every \d+ days))?:`), styles: []func(string) style{always(magenta), always(yellow), always(magenta)}},
	{expression: regexp.MustCompile(`(?m)^(Plan) \S+: \d+ open`), styles: []func(string) style{always(bold)}},
	{expression: regexp.MustCompile(`(?m)(?:^|\. )(Previous:|Blocked:|Next, line \d+:|Next:|Context \d+k\.)`), styles: []func(string) style{always(bold)}},
	{expression: regexp.MustCompile(`(?m)^[ \t]+(blocked:)`), styles: []func(string) style{always(red)}},
	{expression: regexp.MustCompile(`note \d{8}-[0-9a-f]{6} of [^,\n]+(?:, session [0-9a-f]+)?`), styles: []func(string) style{always(dim)}},
	{expression: regexp.MustCompile(`\b\d{8}-[0-9a-f]{6}\b`), styles: []func(string) style{always(dim)}},
	{expression: regexp.MustCompile(`\((?:[0-9a-f]{7,40}, \d{4}-\d{2}-\d{2}|uncommitted)\)`), styles: []func(string) style{always(dim)}},
	{expression: regexp.MustCompile(`\bele (?:plan (?:check|done|list|show|start)|handoff (?:close|list|reopen)|show|recall|remember|context)(?: [\w./<>-]*[./<>-][\w./<>-]*[\w>])*`), styles: []func(string) style{always(cyan)}},
	{expression: regexp.MustCompile(`(?m)(?:^|[\s(])([\w.-]*(?:/[\w.-]+)+\.md)\b`), styles: []func(string) style{always(cyan)}},
	{accept: commandEnds, expression: regexp.MustCompile(`(?m)(?:^|[\s(])(/[a-z][a-z-]*(?::[a-z][a-z-]*)?)`), styles: []func(string) style{always(cyan)}},
}

// Message drops control characters, so a plan or note cannot send escape sequences; NO_COLOR leaves it uncoloured whatever coloured says.
func Message(text string, coloured bool) string {
	text = strings.Map(func(character rune) rune {
		if character != '\n' && character != '\t' && unicode.IsControl(character) {
			return -1
		}
		return character
	}, text)
	if !coloured || os.Getenv("NO_COLOR") != "" {
		return text
	}
	var spans []span
	for _, rule := range rules {
		for _, match := range rule.expression.FindAllStringSubmatchIndex(text, -1) {
			groups := len(match)/2 - 1
			for group := range max(groups, 1) {
				start, end := match[0], match[1]
				if groups > 0 {
					start, end = match[2*group+2], match[2*group+3]
				}
				if start < 0 || rule.accept != nil && !rule.accept(text, end) {
					continue
				}
				candidate := span{end: end, start: start, style: rule.styles[min(group, len(rule.styles)-1)](text[start:end])}
				if !slices.ContainsFunc(spans, func(taken span) bool { return candidate.start < taken.end && taken.start < candidate.end }) {
					spans = append(spans, candidate)
				}
			}
		}
	}
	slices.SortFunc(spans, func(a, b span) int { return a.start - b.start })
	var out strings.Builder
	at := 0
	for _, span := range spans {
		out.WriteString(text[at:span.start])
		out.WriteString(span.style.open)
		out.WriteString(text[span.start:span.end])
		out.WriteString(span.style.close)
		at = span.end
	}
	out.WriteString(text[at:])
	return out.String()
}
