package checkpoint

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

const densityHeader = "file                                                  added   cmt   diff   file  peers    n\n"

func TestDensityCountsBareCarriageReturnsAsLineBreaks(t *testing.T) {
	root := densityRepository(t, map[string]string{"ci.yml": "a: 1\r# note\rb: 2\r"})
	writeFile(t, root, "ci.yml", "a: 1\r# note\rb: 2\rc: 3\r")

	output := measureDensity(t, root)

	require.Equal(t, densityHeader+expectedRow("ci.yml", 1, 0, 0, 25, -1, 0, ""), output)
}

func TestDensityCreditsAQuotedPathAndIgnoresTheUsersDiffPrefixes(t *testing.T) {
	root := densityRepository(t, map[string]string{"a.go": sourceLines(10, 0), `b"q.go`: sourceLines(10, 0)})
	runGit(t, root, "config", "diff.noprefix", "true")
	writeFile(t, root, `b"q.go`, sourceLines(10, 0)+"// why\n// because\n")

	output := measureDensity(t, root)

	require.Equal(t, densityHeader+expectedRow(`b"q.go`, 2, 2, 100, 17, -1, 0, "  <-- diff denser than its yardstick")+"\n1 file(s) flagged on this diff. In these, treat every added comment as deletable until it survives the sibling, restatement, pointer and audience tests.\n", output)
}

func TestDensityFlagsADiffDenserThanItsFileWithNoPeers(t *testing.T) {
	root := densityRepository(t, map[string]string{"main.go": sourceLines(10, 0)})
	writeFile(t, root, "main.go", sourceLines(10, 0)+"// why\n// because\nx := 1\nx := 2\n")

	output := measureDensity(t, root)

	require.True(t, strings.HasPrefix(output, densityHeader+expectedRow("main.go", 4, 2, 50, 14, -1, 0, "  <-- diff denser than its yardstick")), output)
}

func TestDensityFlagsADiffDenserThanItsYardsticks(t *testing.T) {
	root := densityRepository(t, map[string]string{
		"service/a.go":       sourceLines(10, 0),
		"service/b.go":       sourceLines(10, 0),
		"service/c.go":       sourceLines(10, 0),
		"service/changed.go": sourceLines(18, 2),
	})
	writeFile(t, root, "service/changed.go", sourceLines(18, 2)+"// why\n// because\nx := 1\nx := 2\n")

	output := measureDensity(t, root)

	require.Equal(t, densityHeader+
		expectedRow("service/changed.go", 4, 2, 50, 18, 0, 3, "  <-- diff denser than its yardstick  <-- file denser than its peers")+
		"\n1 file(s) flagged on this diff. In these, treat every added comment as deletable until it survives the sibling, restatement, pointer and audience tests.\n"+
		"\n1 file(s) sit denser than their peers regardless of this diff. Sweep the whole file, not only the added lines — this is the shape a file takes when each sitting matched the last one instead of the neighbours.\n", output)
}

func TestDensityLeavesADiffWithinItsLeadUnflagged(t *testing.T) {
	root := densityRepository(t, map[string]string{"main.go": sourceLines(10, 2)})
	writeFile(t, root, "main.go", sourceLines(10, 2)+"// c\n// d\nx := 1\nx := 2\nx := 3\nx := 4\nx := 5\nx := 6\n")

	output := measureDensity(t, root)

	require.Equal(t, densityHeader+expectedRow("main.go", 8, 2, 25, 22, -1, 0, ""), output, "25% is under 1.8 times the file's 22%")
}

func TestDensityLeavesOneAddedCommentUnflagged(t *testing.T) {
	root := densityRepository(t, map[string]string{"main.go": sourceLines(10, 0)})
	writeFile(t, root, "main.go", sourceLines(10, 0)+"// one\nx := 1\n")

	output := measureDensity(t, root)

	require.Equal(t, densityHeader+expectedRow("main.go", 2, 1, 50, 8, -1, 0, ""), output, "a lone comment is noise, and no peers reads as --")
}

func TestDensityShowsTheLastFiftyTwoCharactersOfALongNonASCIIPath(t *testing.T) {
	long := strings.Repeat("é", 50) + "/x.go"
	root := densityRepository(t, map[string]string{long: sourceLines(1, 0)})
	writeFile(t, root, long, sourceLines(2, 0))

	output := measureDensity(t, root)

	require.Contains(t, output, "\n"+strings.Repeat("é", 47)+"/x.go      1     0     0%     0%     --    0\n")
}

func TestDensitySkipsAnAddedLineStartingWithTwoPluses(t *testing.T) {
	root := densityRepository(t, map[string]string{"main.c": "int x;\n"})
	writeFile(t, root, "main.c", "int x;\n++x;\ny++;\n")

	output := measureDensity(t, root)

	require.Equal(t, densityHeader+expectedRow("main.c", 1, 0, 0, 0, -1, 0, ""), output, "read as a +++ header line")
}

func TestDensitySkipsGeneratedFilesAndBlankOnlyChanges(t *testing.T) {
	root := densityRepository(t, map[string]string{"gone.go": sourceLines(3, 0), "kept.go": sourceLines(3, 0)})
	require.NoError(t, os.Remove(filepath.Join(root, "gone.go")))
	writeFile(t, root, "service_mock.go", "// generated\n// generated\n")
	writeFile(t, root, "kept.go", "\n\n")
	runGit(t, root, "add", "-A")

	output := measureDensity(t, root)

	require.Equal(t, "No comparable files in the diff.\n", output)
}

func TestDensityTakesPeersFromTheNearestAncestorHoldingEnough(t *testing.T) {
	root := densityRepository(t, map[string]string{
		"a.go":                sourceLines(4, 1),
		"b.go":                sourceLines(4, 1),
		"c.go":                sourceLines(4, 1),
		"mid/deep/changed.go": sourceLines(9, 1),
		"mid/deep/e1.go":      "",
		"mid/deep/e2.go":      "",
		"mid/deep/f.go":       sourceLines(10, 1),
		"mid/deep/x_mock.go":  sourceLines(4, 4),
		"mid/m1.go":           sourceLines(4, 2),
		"mid/m2.go":           sourceLines(4, 2),
		"mid/m3.go":           sourceLines(4, 2),
	})
	writeFile(t, root, "mid/deep/changed.go", sourceLines(10, 1))

	output := measureDensity(t, root)

	require.Equal(t, densityHeader+expectedRow("mid/deep/changed.go", 1, 0, 0, 10, 50, 4, ""), output, "deep/ has three candidates but one measurable, so mid/ is used, not the root; a mock is no candidate")
}

func TestExtensionIgnoresLeadingDots(t *testing.T) {
	for file, want := range map[string]string{".bashrc": "", "a.b/c": "", "a/..b": "", "a/..b.sh": ".sh", "main.go": ".go", "x.tar.gz": ".gz"} {
		require.Equal(t, want, extension(file), file)
	}
}

func sourceLines(lines int, comments int) string {
	var text strings.Builder
	for index := range comments {
		fmt.Fprintf(&text, "// comment %d\n", index)
	}
	for index := range lines - comments {
		fmt.Fprintf(&text, "x%d := %d\n", index, index)
	}
	return text.String()
}

func densityRepository(t *testing.T, files map[string]string) string {
	t.Helper()
	root, _ := repository(t)
	for file, content := range files {
		writeFile(t, root, file, content)
	}
	runGit(t, root, "add", "-A")
	runGit(t, root, "commit", "-q", "-m", "files")
	return root
}

// A negative peerShare prints as no peers.
func expectedRow(file string, added int, comments int, diffShare int, fileShare int, peerShare int, peerCount int, marker string) string {
	peers := "    --"
	if peerShare >= 0 {
		peers = fmt.Sprintf("%5d%%", peerShare)
	}
	return fmt.Sprintf("%-52s %6d %5d %5d%% %5d%% %s %4d%s\n", file, added, comments, diffShare, fileShare, peers, peerCount, marker)
}

func measureDensity(t *testing.T, root string) string {
	t.Helper()
	var output bytes.Buffer
	require.NoError(t, Density(context.Background(), &output, root, "HEAD"))
	return output.String()
}

func writeFile(t *testing.T, root string, file string, content string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(filepath.Join(root, file)), 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(root, file), []byte(content), 0o600))
}
