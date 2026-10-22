package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/nicolaspearson/claude-ephant/internal/datadir"
	"github.com/nicolaspearson/claude-ephant/internal/dto"
	"github.com/nicolaspearson/claude-ephant/internal/index"
	"github.com/nicolaspearson/claude-ephant/internal/mining"
	"github.com/nicolaspearson/claude-ephant/internal/transcript"
)

var analyses = []string{"checkpoint-runs", "review-cycles"}

func mine(ctx context.Context, args []string, stdout io.Writer, stderr io.Writer) int {
	flags := newFlagSet("mine", stderr)
	digests := flags.String("digests", "", "write each listed unit's digest into this directory, or with --judgments read them back from it")
	judgmentsFile := flags.String("judgments", "", "read the classifications to store from this JSON Lines file, one per unit")
	noSync := flags.Bool("no-sync", false, "read without syncing first")
	promptFile := flags.String("prompt", "", "the classifier prompt the judgments were made with")
	report := flags.Bool("report", false, "report figures from the last stored judgment per unit, both analyses together, with each subagent-era unit's timing read from its archived transcript")
	unitsFile := flags.String("units", "", "list the units this JSON Lines file names by session and timestamp, judged or not")
	err := flags.Parse(args)
	if err == nil && *report {
		if flags.NArg() != 0 || *digests != "" || *judgmentsFile != "" || *promptFile != "" || *unitsFile != "" {
			return rejectCall(flags, nil, stdout, stderr)
		}
		return reportJudgments(ctx, *noSync, stdout, stderr)
	}
	if err != nil || flags.NArg() != 1 || (*judgmentsFile == "") != (*promptFile == "") || (*judgmentsFile != "" && (*digests == "" || *unitsFile != "")) {
		return rejectCall(flags, err, stdout, stderr)
	}
	analysis := flags.Arg(0)
	if !slices.Contains(analyses, analysis) {
		fmt.Fprintf(stderr, "ele: unknown analysis %q; use %s\n", analysis, strings.Join(analyses, ", "))
		return 2
	}
	dataDirectory, err := datadir.Path()
	if err != nil {
		fmt.Fprintf(stderr, "ele: %v\n", err)
		return 1
	}
	cache := filepath.Join(dataDirectory, "mining", "cache")

	searchIndex, err := syncedIndex(ctx, *noSync, stderr, "these units")
	if err != nil {
		fmt.Fprintf(stderr, "ele: %v\n", err)
		return 1
	}
	defer searchIndex.Close()
	units, err := miningUnits(ctx, searchIndex, analysis)
	if err != nil {
		fmt.Fprintf(stderr, "ele: %v\n", err)
		return 1
	}
	archiveDirectory := filepath.Join(dataDirectory, "archive", "transcripts")
	if *judgmentsFile != "" {
		return takeJudgments(cache, *digests, analysis, *promptFile, *judgmentsFile, units, stdout, stderr)
	}
	var pending []dto.MiningUnit
	behind := 0
	var watermark time.Time
	if *unitsFile != "" {
		pending, err = namedUnits(*unitsFile, units)
	} else {
		pending, behind, watermark, err = pendingUnits(cache, analysis, units)
	}
	if err != nil {
		fmt.Fprintf(stderr, "ele: %v\n", err)
		return 1
	}
	if *digests != "" {
		if err = writeDigests(archiveDirectory, *digests, analysis, pending); err != nil {
			fmt.Fprintf(stderr, "ele: %v\n", err)
			return 1
		}
	}
	encoder := json.NewEncoder(stdout)
	for _, unit := range pending {
		if err = encoder.Encode(unit); err != nil {
			fmt.Fprintf(stderr, "ele: %v\n", err)
			return 1
		}
	}
	if behind > 0 {
		fmt.Fprintf(stderr, "ele: unjudged units in %s at or before the watermark %s: %d\n", analysis, watermark.Format(time.RFC3339Nano), behind)
	}
	return 0
}

func miningUnits(ctx context.Context, searchIndex *index.Index, analysis string) ([]dto.MiningUnit, error) {
	var invocations []dto.Event
	for _, kind := range []string{transcript.EventCommand, transcript.EventSkill} {
		for _, name := range checkpointNames {
			found, _, err := searchIndex.Events(ctx, dto.EventQuery{Kind: kind, Name: name})
			if err != nil {
				return nil, err
			}
			invocations = append(invocations, found...)
		}
	}
	runs, err := mining.CheckpointRuns(invocations)
	if err != nil {
		return nil, err
	}
	launches, _, err := searchIndex.Events(ctx, dto.EventQuery{Kind: transcript.EventAgent})
	if err != nil {
		return nil, err
	}
	if analysis == "checkpoint-runs" {
		return mining.BoundRuns(launches, runs)
	}
	return mining.ReviewCycles(launches, runs)
}

func namedUnits(unitsFile string, units []dto.MiningUnit) ([]dto.MiningUnit, error) {
	lines, err := os.ReadFile(unitsFile)
	if err != nil {
		return nil, err
	}
	return mining.Named(units, lines)
}

func pendingUnits(cache string, analysis string, units []dto.MiningUnit) ([]dto.MiningUnit, int, time.Time, error) {
	judgments, err := mining.Load(cache, analysis)
	if err != nil {
		return nil, 0, time.Time{}, err
	}
	watermark, err := mining.Watermark(cache, analysis)
	if err != nil {
		return nil, 0, time.Time{}, err
	}
	pending, behind, err := mining.Pending(units, judgments, watermark)
	return pending, behind, watermark, err
}

func reportJudgments(ctx context.Context, noSync bool, stdout io.Writer, stderr io.Writer) int {
	dataDirectory, err := datadir.Path()
	if err != nil {
		fmt.Fprintf(stderr, "ele: %v\n", err)
		return 1
	}
	cache := filepath.Join(dataDirectory, "mining", "cache")
	runs, err := mining.Load(cache, "checkpoint-runs")
	if err != nil {
		fmt.Fprintf(stderr, "ele: %v\n", err)
		return 1
	}
	cycles, err := mining.Load(cache, "review-cycles")
	if err != nil {
		fmt.Fprintf(stderr, "ele: %v\n", err)
		return 1
	}
	timings, err := timeJudgedUnits(ctx, noSync, dataDirectory, map[string][]mining.Judgment{"checkpoint-runs": runs, "review-cycles": cycles}, stderr)
	if err != nil {
		fmt.Fprintf(stderr, "ele: %v\n", err)
		return 1
	}
	if err = mining.Report(stdout, runs, cycles, timings); err != nil {
		fmt.Fprintf(stderr, "ele: %v\n", err)
		return 1
	}
	return 0
}

// timeJudgedUnits times only subagent-era units: an inline-era run has no reviewer subagents to time.
func timeJudgedUnits(ctx context.Context, noSync bool, dataDirectory string, judgments map[string][]mining.Judgment, stderr io.Writer) (map[mining.UnitKey]mining.Timing, error) {
	searchIndex, err := syncedIndex(ctx, noSync, stderr, "these units")
	if err != nil {
		return nil, err
	}
	defer searchIndex.Close()
	var timed []mining.TimedUnit
	for _, analysis := range analyses {
		reported := map[[2]string]string{}
		for _, judgment := range judgments[analysis] {
			if judgment.Era != "inline" {
				reported[[2]string{judgment.Session, judgment.Timestamp}] = mining.ReportedAt(judgment)
			}
		}
		units, err := miningUnits(ctx, searchIndex, analysis)
		if err != nil {
			return nil, err
		}
		for _, unit := range units {
			if reportedAt, judged := reported[[2]string{unit.Session, unit.Timestamp}]; judged {
				timed = append(timed, mining.TimedUnit{Analysis: analysis, ReportedAt: reportedAt, Unit: unit})
			}
		}
	}
	return mining.Time(filepath.Join(dataDirectory, "archive", "transcripts"), timed), nil
}

func takeJudgments(cache string, digests string, analysis string, promptFile string, judgmentsFile string, units []dto.MiningUnit, stdout io.Writer, stderr io.Writer) int {
	prompt, err := os.ReadFile(promptFile)
	if err != nil {
		fmt.Fprintf(stderr, "ele: %v\n", err)
		return 1
	}
	judgmentLines, err := os.ReadFile(judgmentsFile)
	if err != nil {
		fmt.Fprintf(stderr, "ele: %v\n", err)
		return 1
	}
	digest := func(unit dto.MiningUnit) (string, error) {
		content, readErr := os.ReadFile(filepath.Join(digests, digestName(unit)))
		return string(content), readErr
	}
	taken, err := mining.Take(cache, analysis, prompt, digest, judgmentLines, units)
	if err != nil {
		fmt.Fprintf(stderr, "ele: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "Stored %d of %d judgments in %s under classifier %s; watermark %s\n", taken.Added, taken.Judgments, analysis, taken.Classifier[:12], taken.Watermark.Format(time.RFC3339Nano))
	return 0
}

// writeDigests names each file for its unit, so a re-run rewrites the same file.
func writeDigests(archiveDirectory string, directory string, analysis string, units []dto.MiningUnit) error {
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return err
	}
	for position := range units {
		digest, err := mining.Digest(archiveDirectory, analysis, units[position])
		if err != nil {
			return err
		}
		path, err := filepath.Abs(filepath.Join(directory, digestName(units[position])))
		if err != nil {
			return err
		}
		if err = os.WriteFile(path, []byte(digest), 0o600); err != nil {
			return err
		}
		units[position].Digest = path
	}
	return nil
}

func digestName(unit dto.MiningUnit) string {
	return unit.Session + "-" + strings.NewReplacer(":", "", ".", "").Replace(unit.Timestamp) + ".txt"
}
