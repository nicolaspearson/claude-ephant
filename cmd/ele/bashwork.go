package main

import (
	"regexp"
	"slices"
	"strings"

	"github.com/nicolaspearson/claude-ephant/internal/transcript"
)

var (
	assignment = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*=`)
	// Reads, or ele's own bookkeeping, which changes no repository file. Nil takes any subcommand.
	eleNotWork = map[string][]string{
		"checkpoint": {"check", "hash"},
		"config":     {"get"},
		"context":    nil,
		"events":     nil,
		"handoff":    {"close", "list", "reopen"},
		"plan":       {"check", "list", "show"},
		"recall":     nil,
		"reminder":   {"close", "list", "reopen"},
		"show":       nil,
		"status":     nil,
		"version":    nil,
	}
	findWrites = []string{"-delete", "-exec", "-execdir", "-fls", "-fprint", "-fprint0", "-fprintf", "-ok", "-okdir"}
	gitReads   = []string{"blame", "diff", "for-each-ref", "log", "ls-files", "rev-parse", "show", "status"}
	// A quoted delimiter keeps the body literal; anything after it on the line is an argument or redirect.
	heredocHead = regexp.MustCompile(`^cat <<-? *('[^'\n]+'|"[^"\n]+") *$`)
	loopWords   = []string{"!", "do", "done", "elif", "else", "fi", "if", "then", "until", "while", "{", "}"}
	plainReads  = []string{"[", "basename", "cat", "cd", "cut", "date", "diff", "dirname", "echo", "find", "grep", "head", "jq", "ls", "nl", "printf", "pwd", "rg", "sed", "sort", "stat", "tail", "test", "uniq", "wc"}
	// Prints, deletes from the output, quits or substitutes with / and no w or e flag; sed's w and e write and run.
	sedPrint = regexp.MustCompile(`^(((\d+|\$|/[^/]*/)(,(\d+|\$|/[^/]*/))?)?[dpq=]|s/([^/\\]|\\.)*/([^/\\]|\\.)*/[gpiI0-9]*)$`)
	variable = regexp.MustCompile(`^\$\{?([A-Za-z_][A-Za-z0-9_]*)\}?`)
)

type shellScan struct {
	assigned map[string]string
	commands [][]string
	nested   []string
	targets  []string
}

// Whether a note's command runs anything beside its ele remember and cd calls, and whether any of it is work.
func besideNotes(command string) (other bool, work bool) {
	work = scanIsWork(command, func(words []string) bool {
		if slices.Equal(words[:min(2, len(words))], []string{"ele", "remember"}) || words[0] == "cd" {
			return false
		}
		other = true
		return simpleCommandIsWork(words)
	})
	if scan, ok := scanShell(command); ok && !other {
		other = slices.ContainsFunc(scan.nested, func(body string) bool { return !heredocCat(body) })
	}
	return other, work
}

// Defaults to work: a command it cannot read whole may write.
func commandIsWork(command string) bool {
	return scanIsWork(command, simpleCommandIsWork)
}

func eleIsWork(args []string) bool {
	if len(args) == 0 {
		return true
	}
	subcommands, ok := eleNotWork[args[0]]
	if !ok {
		return true
	}
	return subcommands != nil && (len(args) < 2 || !slices.Contains(subcommands, args[1]))
}

func gitIsWork(args []string) bool {
globals:
	for len(args) > 0 {
		switch args[0] {
		// -c can set an alias or diff.external that runs a command.
		case "-c":
			return true
		case "-C":
			args = args[min(2, len(args)):]
		case "--no-pager":
			args = args[1:]
		default:
			break globals
		}
	}
	if len(args) == 0 {
		return true
	}
	if args[0] == "branch" {
		return !slices.Equal(args[1:], []string{"--show-current"})
	}
	return !slices.Contains(gitReads, args[0]) || slices.ContainsFunc(args[1:], func(arg string) bool { return strings.HasPrefix(arg, "--output") })
}

// A cat of a heredoc whose delimiter is quoted, so its body is literal text: the usual way to pass a multi-line note.
// The first delimiter line must end the body: lines after it run as commands.
func heredocCat(body string) bool {
	trimmed := strings.TrimSpace(body)
	head, _, _ := strings.Cut(trimmed, "\n")
	if !heredocHead.MatchString(head) {
		return false
	}
	runes := []rune(trimmed)
	end, ok := heredocEnd(runes, len("cat "))
	return ok && end == len(runes)-1
}

// The index of the last character of the closing delimiter line of the heredoc whose << is at start.
func heredocEnd(runes []rune, start int) (int, bool) {
	index := start + 2
	strip := index < len(runes) && runes[index] == '-'
	if strip {
		index++
	}
	for index < len(runes) && runes[index] == ' ' {
		index++
	}
	var delimiter []rune
	if index < len(runes) && (runes[index] == '\'' || runes[index] == '"') {
		closing := slices.Index(runes[index+1:], runes[index])
		if closing < 0 {
			return 0, false
		}
		delimiter = runes[index+1 : index+1+closing]
	} else {
		for index < len(runes) && !strings.ContainsRune(" \t\n;&|()", runes[index]) {
			delimiter = append(delimiter, runes[index])
			index++
		}
	}
	newline := slices.Index(runes[index:], '\n')
	if len(delimiter) == 0 || newline < 0 {
		return 0, false
	}
	offset := index + newline + 1
	for line := range strings.SplitSeq(string(runes[offset:]), "\n") {
		text := line
		if strip {
			text = strings.TrimLeft(text, "\t")
		}
		length := len([]rune(line))
		if text == string(delimiter) {
			return offset + length - 1, true
		}
		offset += length + 1
	}
	return 0, false
}

// The body of the $( or backtick substitution at start, and the index of its closing character.
func nestedBody(runes []rune, start int) (string, int, bool) {
	if runes[start] == '`' {
		for end := start + 1; end < len(runes); end++ {
			switch runes[end] {
			case '\\':
				end++
			case '`':
				return string(runes[start+1 : end]), end, true
			}
		}
		return "", 0, false
	}
	depth := 0
	for end := start + 1; end < len(runes); end++ {
		switch runes[end] {
		case '\\':
			end++
		case '<':
			if end+1 < len(runes) && runes[end+1] == '<' {
				closing, ok := heredocEnd(runes, end)
				if !ok {
					return "", 0, false
				}
				end = closing
			}
		case '\'':
			closing := slices.Index(runes[end+1:], '\'')
			if closing < 0 {
				return "", 0, false
			}
			end += closing + 1
		case '"':
			closing := end + 1
			for ; closing < len(runes) && runes[closing] != '"'; closing++ {
				switch {
				case runes[closing] == '\\':
					closing++
				case runes[closing] == '`' || runes[closing] == '$' && closing+1 < len(runes) && runes[closing+1] == '(':
					_, inner, ok := nestedBody(runes, closing)
					if !ok {
						return "", 0, false
					}
					closing = inner
				}
			}
			if closing >= len(runes) {
				return "", 0, false
			}
			end = closing
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				return string(runes[start+2 : end]), end, true
			}
		}
	}
	return "", 0, false
}

func safeTarget(target string) bool {
	switch {
	case target == "/dev/null" || target == "&1" || target == "&2" || target == "&-":
		return true
	// A variable left after expand can hold ../ or any path.
	case strings.Contains(target, "..") || strings.ContainsAny(target, "$`"):
		return false
	}
	return strings.HasPrefix(target, "/tmp/") || strings.HasPrefix(target, "/private/tmp/")
}

func scanIsWork(command string, isWork func(words []string) bool) bool {
	scan, ok := scanShell(command)
	if !ok || len(scan.commands) == 0 {
		return true
	}
	return slices.ContainsFunc(scan.targets, func(target string) bool { return !safeTarget(target) }) ||
		slices.ContainsFunc(scan.nested, func(body string) bool { return !heredocCat(body) && commandIsWork(body) }) ||
		slices.ContainsFunc(scan.commands, isWork)
}

// Simple commands split at ; & | ( ) and newlines, with redirect targets and substitution bodies apart. False for a
// heredoc, whose body a quote can carry past its delimiter; an unclosed quote or substitution; a >& to a file; or a
// redirect with no target, as a process substitution's is.
func scanShell(command string) (shellScan, bool) {
	scan := shellScan{assigned: map[string]string{}}
	runes := []rune(command)
	var current []string
	var word strings.Builder
	inWord, redirect, output := false, false, false
	endWord := func() {
		if !inWord {
			return
		}
		text := word.String()
		word.Reset()
		inWord = false
		if redirect {
			if output {
				scan.targets = append(scan.targets, scan.expand(text))
			}
			redirect = false
			return
		}
		current = append(current, text)
	}
	endCommand := func() {
		endWord()
		if len(current) == 0 {
			return
		}
		if !slices.ContainsFunc(current, func(word string) bool { return !assignment.MatchString(word) }) {
			for _, pair := range current {
				name, value, _ := strings.Cut(pair, "=")
				scan.assigned[name] = value
			}
		}
		scan.commands = append(scan.commands, current)
		current = nil
	}
	substitute := func(start int) (int, bool) {
		body, end, ok := nestedBody(runes, start)
		if ok {
			scan.nested = append(scan.nested, body)
			inWord = true
		}
		return end, ok
	}
	next := func(i int) rune {
		if i+1 < len(runes) {
			return runes[i+1]
		}
		return 0
	}
	for i := 0; i < len(runes); i++ {
		character := runes[i]
		switch {
		case character == '\\':
			if i++; i < len(runes) && runes[i] != '\n' {
				word.WriteRune(runes[i])
				inWord = true
			}
		case character == '\'':
			closing := slices.Index(runes[i+1:], '\'')
			if closing < 0 {
				return scan, false
			}
			word.WriteString(string(runes[i+1 : i+1+closing]))
			inWord, i = true, i+1+closing
		case character == '"':
			j := i + 1
			for ; j < len(runes) && runes[j] != '"'; j++ {
				switch {
				case runes[j] == '\\' && j+1 < len(runes):
					j++
					word.WriteRune(runes[j])
				case runes[j] == '`' || runes[j] == '$' && next(j) == '(':
					end, ok := substitute(j)
					if !ok {
						return scan, false
					}
					j = end
				default:
					word.WriteRune(runes[j])
				}
			}
			if j >= len(runes) {
				return scan, false
			}
			inWord, i = true, j
		case character == '`' || character == '$' && next(i) == '(':
			end, ok := substitute(i)
			if !ok {
				return scan, false
			}
			i = end
		case character == '#' && !inWord:
			if newline := slices.Index(runes[i:], '\n'); newline >= 0 {
				i += newline - 1
			} else {
				i = len(runes)
			}
		case character == ' ' || character == '\t':
			endWord()
		case character == '>' || character == '<' || character == '&' && next(i) == '>':
			if character == '<' && next(i) == '<' {
				return scan, false
			}
			if inWord && strings.Trim(word.String(), "0123456789") == "" {
				word.Reset()
				inWord = false
			}
			endWord()
			output = character != '<'
			if character == '&' {
				i++
			}
			switch next(i) {
			case '>':
				output = true
				i++
			case '|':
				i++
			}
			if output && next(i) == '&' {
				end := i + 2
				for end < len(runes) && (runes[end] >= '0' && runes[end] <= '9' || runes[end] == '-') {
					end++
				}
				if end == i+2 || end < len(runes) && !strings.ContainsRune(" \t\n;&|()", runes[end]) {
					return scan, false
				}
				scan.targets = append(scan.targets, "&"+string(runes[i+2:end]))
				i = end - 1
				continue
			}
			redirect = true
		case strings.ContainsRune("\n;&|()", character):
			if redirect && !inWord {
				return scan, false
			}
			endCommand()
		default:
			word.WriteRune(character)
			inWord = true
		}
	}
	if redirect && !inWord {
		return scan, false
	}
	endCommand()
	return scan, true
}

func sedIsWork(args []string) bool {
	var operands, scripts []string
	for index := 0; index < len(args); index++ {
		arg := args[index]
		short := strings.HasPrefix(arg, "-") && !strings.HasPrefix(arg, "--") && arg != "-"
		switch {
		case arg == "--quiet" || arg == "--regexp-extended" || arg == "--silent":
		case arg == "--expression" || short && strings.HasSuffix(arg, "e") && !strings.ContainsAny(arg, "fi"):
			if index++; index >= len(args) {
				return true
			}
			scripts = append(scripts, args[index])
		case strings.HasPrefix(arg, "--") || short && strings.ContainsAny(arg, "efi"):
			return true
		case short:
		default:
			operands = append(operands, arg)
		}
	}
	if len(scripts) == 0 && len(operands) > 0 {
		scripts = operands[:1]
	}
	return len(scripts) == 0 || slices.ContainsFunc(scripts, func(script string) bool {
		return slices.ContainsFunc(strings.Split(script, ";"), func(command string) bool { return !sedPrint.MatchString(strings.TrimSpace(command)) })
	})
}

func simpleCommandIsWork(words []string) bool {
	for len(words) > 0 && slices.Contains(loopWords, words[0]) {
		words = words[1:]
	}
	// An assignment before a command can change what it runs, as GIT_EXTERNAL_DIFF does.
	if len(words) > 0 && assignment.MatchString(words[0]) {
		return slices.ContainsFunc(words, func(word string) bool { return !assignment.MatchString(word) })
	}
	if len(words) == 0 || words[0] == "for" {
		return false
	}
	args := words[1:]
	switch words[0] {
	case "ele":
		return eleIsWork(args)
	case "find":
		return slices.ContainsFunc(args, func(arg string) bool { return slices.Contains(findWrites, arg) })
	case "git":
		return gitIsWork(args)
	case "rg":
		return slices.ContainsFunc(args, func(arg string) bool {
			return strings.HasPrefix(arg, "--hostname-bin") || strings.HasPrefix(arg, "--pre")
		})
	case "sed":
		return sedIsWork(args)
	case "sort":
		return slices.ContainsFunc(args, func(arg string) bool {
			return strings.HasPrefix(arg, "--compress-program") || strings.HasPrefix(arg, "--output") || strings.HasPrefix(arg, "-") && !strings.HasPrefix(arg, "--") && strings.Contains(arg, "o")
		})
	case "uniq":
		operands := slices.DeleteFunc(slices.Clone(args), func(arg string) bool { return strings.HasPrefix(arg, "-") })
		return len(operands) > 1
	}
	return !slices.Contains(plainReads, words[0])
}

// A target naming a variable this command assigned reads as its value; any other variable stays unexpanded.
func (scan shellScan) expand(target string) string {
	match := variable.FindStringSubmatch(target)
	if match == nil {
		return target
	}
	value, ok := scan.assigned[match[1]]
	if !ok {
		return target
	}
	return value + target[len(match[0]):]
}

func logsCheckpoint(command string) bool {
	return transcript.StartsCommand(command, "ele checkpoint log ")
}

func writesNote(command string) bool {
	return transcript.StartsCommand(command, "ele remember ")
}
