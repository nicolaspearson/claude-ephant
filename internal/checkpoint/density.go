package checkpoint

import (
	"context"
	"fmt"
	"io"
	"maps"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
)

const (
	// Enough lead over the habit compared with to be worth a second look, not one comment's noise on a short change.
	densityLead          = 1.8
	minimumAddedComments = 2
	// Below this a median is one or two files' taste, not a habit.
	minimumPeers = 3
	peerScanCap  = 400
)

var (
	commentStarts = []string{"//", "#", "*", "/*", "--"}

	// Generated files are nobody's prose.
	skipSuffixes = []string{"_mock.go", "_templ.go", ".pb.go", ".sql.go", ".gen.go", ".lock", ".json"}
)

type densityRow struct {
	added        int
	comments     int
	diffShare    float64
	drift        bool
	fileShare    float64
	flag         bool
	path         string
	peerCount    int
	peerShare    float64
	peersMeasure bool
}

func Density(ctx context.Context, w io.Writer, root string, ref string) error {
	// Fixed prefixes: a user's diff.noprefix or diff.mnemonicPrefix would leave every header unmatched.
	diff, err := git(ctx, root, "-c", "core.quotePath=false", "diff", "--no-color", "--no-ext-diff", "--src-prefix=a/", "--dst-prefix=b/", ref, "-U0")
	if err != nil {
		return err
	}
	listed, err := git(ctx, root, "-c", "core.quotePath=false", "ls-files")
	if err != nil {
		return err
	}
	tracked := splitLines(listed, "\n")
	added, comments := addedLines(diff)

	var rows []densityRow
	for _, changed := range slices.Sorted(maps.Keys(added)) {
		if changed == "" || skipped(changed) {
			continue
		}
		if _, err := os.Stat(filepath.Join(root, changed)); err != nil {
			continue
		}
		fileShare, ok := density(filepath.Join(root, changed))
		if !ok {
			continue
		}
		row := densityRow{added: added[changed], comments: comments[changed], fileShare: fileShare, path: changed}
		row.peerShare, row.peerCount, row.peersMeasure = peerDensity(root, changed, tracked)
		row.diffShare = 100 * float64(row.comments) / float64(row.added)
		row.flag = row.comments >= minimumAddedComments && (row.diffShare > fileShare*densityLead || row.peersMeasure && row.diffShare > row.peerShare*densityLead)
		row.drift = row.peersMeasure && fileShare > row.peerShare*densityLead
		rows = append(rows, row)
	}

	if len(rows) == 0 {
		fmt.Fprintln(w, "No comparable files in the diff.")
		return nil
	}
	fmt.Fprintf(w, "%-52s %6s %5s %6s %6s %6s %4s\n", "file", "added", "cmt", "diff", "file", "peers", "n")
	flagged, drifted := 0, 0
	for _, row := range rows {
		peerText := "    --"
		if row.peersMeasure {
			peerText = fmt.Sprintf("%5.0f%%", row.peerShare)
		}
		marker := ""
		if row.flag {
			marker += "  <-- diff denser than its yardstick"
			flagged++
		}
		if row.drift {
			marker += "  <-- file denser than its peers"
			drifted++
		}
		shown := []rune(row.path)
		fmt.Fprintf(w, "%-52s %6d %5d %5.0f%% %5.0f%% %s %4d%s\n", string(shown[max(0, len(shown)-52):]), row.added, row.comments, row.diffShare, row.fileShare, peerText, row.peerCount, marker)
	}
	if flagged > 0 {
		fmt.Fprintf(w, "\n%d file(s) flagged on this diff. In these, treat every added comment as deletable until it survives the sibling, restatement, pointer and audience tests.\n", flagged)
	}
	if drifted > 0 {
		fmt.Fprintf(w, "\n%d file(s) sit denser than their peers regardless of this diff. Sweep the whole file, not only the added lines — this is the shape a file takes when each sitting matched the last one instead of the neighbours.\n", drifted)
	}
	return nil
}

func addedLines(diff string) (map[string]int, map[string]int) {
	added, comments := map[string]int{}, map[string]int{}
	current := ""
	for _, line := range splitLines(diff, "\n") {
		if after, ok := strings.CutPrefix(line, "+++ b/"); ok {
			current = after
			continue
		}
		// git still quotes a path holding a quote, backslash or control character.
		if quoted, ok := strings.CutPrefix(line, `+++ "b/`); ok {
			if unquoted, err := strconv.Unquote(`"` + quoted); err == nil {
				current = unquoted
			}
			continue
		}
		if !strings.HasPrefix(line, "+") || strings.HasPrefix(line, "+++") {
			continue
		}
		body := strings.TrimSpace(line[1:])
		if body == "" {
			continue
		}
		added[current]++
		if isComment(body) {
			comments[current]++
		}
	}
	return added, comments
}

func density(file string) (float64, bool) {
	content, err := os.ReadFile(file)
	if err != nil {
		return 0, false
	}
	comments, total := 0, 0
	// A file saved with bare \r endings is many lines, not one.
	for _, line := range splitLines(string(content), "\n\r") {
		if body := strings.TrimSpace(line); body != "" {
			total++
			if isComment(body) {
				comments++
			}
		}
	}
	if total == 0 {
		return 0, false
	}
	return 100 * float64(comments) / float64(total), true
}

// A name's leading dots, as in .bashrc, start no extension.
func extension(file string) string {
	name := path.Base(file)
	dot := strings.LastIndex(name, ".")
	if dot <= 0 || strings.Trim(name[:dot], ".") == "" {
		return ""
	}
	return name[dot:]
}

func isComment(line string) bool {
	return slices.ContainsFunc(commentStarts, func(start string) bool { return strings.HasPrefix(line, start) })
}

func peerDensity(root string, changed string, tracked []string) (float64, int, bool) {
	suffix := extension(changed)
	if suffix == "" {
		return 0, 0, false
	}
	var candidates []string
	for _, other := range tracked {
		if other != changed && strings.HasSuffix(other, suffix) && !skipped(other) {
			candidates = append(candidates, other)
		}
	}
	directory := path.Dir(changed)
	if directory == "." {
		directory = ""
	}
	for {
		prefix := ""
		if directory != "" {
			prefix = directory + "/"
		}
		var nearby []string
		for _, other := range candidates {
			if strings.HasPrefix(other, prefix) {
				nearby = append(nearby, other)
			}
		}
		if len(nearby) >= minimumPeers {
			var densities []float64
			for _, other := range nearby[:min(peerScanCap, len(nearby))] {
				if share, ok := density(filepath.Join(root, other)); ok {
					densities = append(densities, share)
				}
			}
			if len(densities) >= minimumPeers {
				return median(densities), len(densities), true
			}
		}
		if directory == "" {
			return 0, 0, false
		}
		if directory = path.Dir(directory); directory == "." {
			directory = ""
		}
	}
}

func skipped(file string) bool {
	return slices.ContainsFunc(skipSuffixes, func(suffix string) bool { return strings.HasSuffix(file, suffix) })
}

// Drops empty lines; neither a diff nor a file counts them.
func splitLines(text string, breaks string) []string {
	return strings.FieldsFunc(text, func(character rune) bool { return strings.ContainsRune(breaks, character) })
}
