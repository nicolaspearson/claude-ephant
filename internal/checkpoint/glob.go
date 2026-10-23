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

func match(glob []rune, text []rune) bool {
	for len(glob) > 0 {
		switch glob[0] {
		case '*':
			for len(glob) > 0 && glob[0] == '*' {
				glob = glob[1:]
			}
			for skip := 0; skip <= len(text); skip++ {
				if match(glob, text[skip:]) {
					return true
				}
			}
			return false
		case '?':
			if len(text) == 0 {
				return false
			}
			glob, text = glob[1:], text[1:]
		case '[':
			if matched, rest, ok := matchClass(glob, text); ok {
				if !matched {
					return false
				}
				glob, text = rest, text[1:]
				continue
			}
			if len(text) == 0 || text[0] != '[' {
				return false
			}
			glob, text = glob[1:], text[1:]
		case '\\':
			if len(glob) > 1 {
				glob = glob[1:]
			}
			fallthrough
		default:
			if len(text) == 0 || text[0] != glob[0] {
				return false
			}
			glob, text = glob[1:], text[1:]
		}
	}
	return len(text) == 0
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
