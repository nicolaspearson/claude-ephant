package checkpoint

import (
	"strings"
	"unicode"
)

var posixClasses = map[string]func(rune) bool{
	"alnum":  func(r rune) bool { return unicode.IsLetter(r) || unicode.IsDigit(r) },
	"alpha":  unicode.IsLetter,
	"blank":  func(r rune) bool { return r == ' ' || r == '\t' },
	"cntrl":  unicode.IsControl,
	"digit":  unicode.IsDigit,
	"graph":  func(r rune) bool { return unicode.IsGraphic(r) && !unicode.IsSpace(r) },
	"lower":  unicode.IsLower,
	"print":  unicode.IsPrint,
	"punct":  unicode.IsPunct,
	"space":  unicode.IsSpace,
	"upper":  unicode.IsUpper,
	"xdigit": func(r rune) bool { return strings.ContainsRune("0123456789abcdefABCDEF", r) },
}

// MatchGlob matches as bash's `[[ path == glob ]]` does: `*` crosses `/`, unlike path.Match.
func MatchGlob(glob string, text string) bool {
	return match([]rune(glob), []rune(text))
}

// Only the last `*` is retried: every other token takes one rune, and retrying each star is exponential.
func match(glob []rune, text []rune) bool {
	g, t := 0, 0
	star, starText := -1, 0
	for g < len(glob) || t < len(text) {
		if g < len(glob) && glob[g] == '*' {
			star, starText = g, t
			g++
			continue
		}
		if g < len(glob) {
			if rest, ok := matchOne(glob[g:], text[t:]); ok {
				g, t = len(glob)-len(rest), t+1
				continue
			}
		}
		if star < 0 || starText == len(text) {
			return false
		}
		starText++
		g, t = star+1, starText
	}
	return true
}

// matchClass is not ok where the bracket never closes, which bash reads as a literal `[`.
func matchClass(glob []rune, text []rune) (matched bool, rest []rune, ok bool) {
	at := 1
	negated := at < len(glob) && (glob[at] == '!' || glob[at] == '^')
	if negated {
		at++
	}
	var r rune
	if len(text) > 0 {
		r = text[0]
	}
	found := false
	for first := true; at < len(glob); first = false {
		if glob[at] == ']' && !first {
			return len(text) > 0 && found != negated, glob[at+1:], true
		}
		if glob[at] == '[' && at+1 < len(glob) && glob[at+1] == ':' {
			if end := strings.Index(string(glob[at+2:]), ":]"); end >= 0 {
				name := string(glob[at+2:])[:end]
				if test, known := posixClasses[name]; known && len(text) > 0 && test(r) {
					found = true
				}
				at += 2 + len([]rune(name)) + 2
				continue
			}
		}
		low := glob[at]
		if low == '\\' && at+1 < len(glob) {
			at++
			low = glob[at]
		}
		at++
		high := low
		if at+1 < len(glob) && glob[at] == '-' && glob[at+1] != ']' {
			high = glob[at+1]
			at += 2
		}
		if len(text) > 0 && low <= r && r <= high {
			found = true
		}
	}
	return false, nil, false
}

func matchOne(glob []rune, text []rune) (rest []rune, ok bool) {
	if len(text) == 0 {
		return nil, false
	}
	switch glob[0] {
	case '?':
		return glob[1:], true
	case '[':
		if matched, rest, ok := matchClass(glob, text); ok {
			return rest, matched
		}
	case '\\':
		if len(glob) > 1 {
			glob = glob[1:]
		}
	}
	return glob[1:], text[0] == glob[0]
}
