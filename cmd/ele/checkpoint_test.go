package main

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/nicolaspearson/claude-ephant/internal/checkpoint"
)

const checkpointRecord = `{"change":{"files":1,"lines":2},"findings":[],"mode":"small","mutations":{"failed":0,"inconclusive":0,"run":0},"notes":{"approved":0,"proposed":0},"rereview":"skipped","scope":"uncommitted"}`

func TestRunCheckpointDensityMeasuresTheWholeRepositoryFromASubdirectory(t *testing.T) {
	root, _ := checkpointRepository(t)
	require.NoError(t, os.WriteFile(filepath.Join(root, "main.go"), []byte("x := 1\n"), 0o600))
	added, err := exec.Command("git", "-C", root, "add", "main.go").CombinedOutput()
	require.NoError(t, err, string(added))

	output, code := runCheckpointDensity(t, filepath.Join(root, "sub"))

	require.Equal(t, 0, code, output)
	require.True(t, strings.HasPrefix(output, "file "), output)
	require.Contains(t, output, "\nmain.go ")
}

func TestRunCheckpointDensityRefusesOutsideARepositoryAndAnUnknownRef(t *testing.T) {
	root, _ := checkpointRepository(t)
	t.Chdir(t.TempDir())

	output, code := runCheckpointDensity(t)
	require.Equal(t, 2, code)
	require.Equal(t, "ele: the working directory is not inside a git repository\n", output)

	output, code = runCheckpointDensity(t, "--ref", "nowhere", root)
	require.Equal(t, 1, code)
	require.True(t, strings.HasPrefix(output, "ele: git "), output)
	require.Contains(t, output, "unknown revision")
}

func TestRunCheckpointLogAppendsFromASubdirectory(t *testing.T) {
	root, logPath := checkpointRepository(t)

	output, code := runCheckpointLog(t, checkpointRecord, filepath.Join(root, "sub"))

	require.Equal(t, 0, code, output)
	require.Equal(t, "logged: "+root+"\n", output)
	written, err := os.ReadFile(logPath)
	require.NoError(t, err)
	require.Contains(t, string(written), `"repository":"`+root+`"`)
}

func TestRunCheckpointLogRefusesARecordAndKeepsTheFreezeTimeForTheRetry(t *testing.T) {
	root, logPath := checkpointRepository(t)
	marker, err := checkpoint.ReviewedFile(context.Background(), root)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(marker+".started", []byte("1\n"), 0o600))

	output, code := runCheckpointLog(t, strings.Replace(checkpointRecord, `"small"`, `"medium"`, 1), root)

	require.Equal(t, 2, code)
	require.Equal(t, "ele: record refused:\n  mode \"medium\" is not one of full, small\n", output)
	require.NoFileExists(t, logPath)
	require.FileExists(t, marker+".started")

	output, code = runCheckpointLog(t, "{", root)
	require.Equal(t, 2, code)
	require.True(t, strings.HasPrefix(output, "ele: the record is not JSON: "), output)
}

func TestRunCheckpointLogRefusesOutsideARepository(t *testing.T) {
	t.Chdir(t.TempDir())

	for _, args := range [][]string{{}, {""}} {
		output, code := runCheckpointLog(t, checkpointRecord, args...)

		require.Equal(t, 2, code, args)
		require.Equal(t, "ele: the working directory is not inside a git repository\n", output, "an empty root reads as none, as the script's ${1:-.} did")
	}
}

func TestRunCheckpointStatsReadsTheBaselineOnlyWhenAsked(t *testing.T) {
	directory := t.TempDir()
	t.Setenv("CHECKPOINT_LOG", filepath.Join(directory, "checkpoints.jsonl"))
	baselinePath := filepath.Join(directory, "baseline.jsonl")
	t.Setenv("CHECKPOINT_BASELINE", baselinePath)

	output, code := runCheckpointStats(t, "7", "--baseline")
	require.Equal(t, 1, code, output)
	require.True(t, strings.HasSuffix(output, "\nNo baseline at "+baselinePath+".\n"), output)

	require.NoError(t, os.WriteFile(baselinePath, nil, 0o600))
	output, code = runCheckpointStats(t)
	require.Equal(t, 0, code, output)
	require.NotContains(t, output, "\n\nBaseline,")

	output, code = runCheckpointStats(t, "--baseline", "7")
	require.Equal(t, 0, code, output)
	require.Contains(t, output, "\n\nBaseline, whatever the window: 0 checkpoints mined from transcripts, not logged; confidence high 0, medium 0, low 0.\n")
}

func TestRunCheckpointStatsRefusesDaysThatAreNotOneWholeNumber(t *testing.T) {
	t.Setenv("CHECKPOINT_LOG", filepath.Join(t.TempDir(), "checkpoints.jsonl"))

	output, code := runCheckpointStats(t, "abc")
	require.Equal(t, 2, code)
	require.Equal(t, "ele: days \"abc\" is not a whole number\n", output)

	output, code = runCheckpointStats(t, "1", "2")
	require.Equal(t, 2, code)
	require.True(t, strings.HasPrefix(output, "ele checkpoint stats [--baseline] [<days>]\n"), output)
}

func checkpointRepository(t *testing.T) (string, string) {
	t.Helper()
	directory := t.TempDir()
	for _, args := range [][]string{{"init", "-q"}, {"commit", "-q", "--allow-empty", "-m", "first"}} {
		output, err := exec.Command("git", append([]string{"-C", directory, "-c", "user.name=t", "-c", "user.email=t@example.com", "-c", "commit.gpgsign=false"}, args...)...).CombinedOutput()
		require.NoError(t, err, string(output))
	}
	require.NoError(t, os.MkdirAll(filepath.Join(directory, "sub"), 0o700))
	root, err := checkpoint.Root(context.Background(), directory)
	require.NoError(t, err)
	logPath := filepath.Join(t.TempDir(), "checkpoints.jsonl")
	t.Setenv("CHECKPOINT_LOG", logPath)
	t.Setenv("CHECKPOINT_REVIEWED_DIR", filepath.Join(t.TempDir(), "reviewed"))
	return root, logPath
}

func runCheckpointDensity(t *testing.T, args ...string) (string, int) {
	t.Helper()
	var output bytes.Buffer
	code := checkpointCommand(context.Background(), append([]string{"density"}, args...), strings.NewReader(""), &output, &output)
	return output.String(), code
}

func runCheckpointLog(t *testing.T, record string, args ...string) (string, int) {
	t.Helper()
	var output bytes.Buffer
	code := checkpointCommand(context.Background(), append([]string{"log"}, args...), strings.NewReader(record), &output, &output)
	return output.String(), code
}

func runCheckpointStats(t *testing.T, args ...string) (string, int) {
	t.Helper()
	var output bytes.Buffer
	code := checkpointCommand(context.Background(), append([]string{"stats"}, args...), strings.NewReader(""), &output, &output)
	return output.String(), code
}
