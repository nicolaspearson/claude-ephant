package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/nicolaspearson/claude-ephant/internal/datadir"
	"github.com/nicolaspearson/claude-ephant/internal/dto"
	"github.com/nicolaspearson/claude-ephant/internal/index"
	"github.com/nicolaspearson/claude-ephant/internal/ownermap"
	"github.com/nicolaspearson/claude-ephant/internal/transcript"
)

var eventKinds = []string{transcript.EventAgent, transcript.EventCommand, transcript.EventCompaction, transcript.EventHookBlock, transcript.EventInjection, transcript.EventReply, transcript.EventSkill, transcript.EventTokens, transcript.EventToolError}

func events(ctx context.Context, args []string, stdout io.Writer, stderr io.Writer) int {
	flags := newFlagSet("events", stderr)
	kind := flags.String("kind", "", "only this kind: "+strings.Join(eventKinds, ", "))
	limit := flags.Int("limit", 200, "print only the newest `n`; 0 prints every match")
	name := flags.String("name", "", "only this name: the skill, command, subagent type, hook (an injection's includes its matcher, PostToolUse:Edit), or skills for the skill list")
	noSync := flags.Bool("no-sync", false, "read without syncing first")
	owner := flags.String("owner", "", "only this owner")
	project := flags.String("project", "", "only this project")
	session := flags.String("session", "", "only this session")
	since := flags.String("since", "", "only events from this date (YYYY-MM-DD) or this many days back (7d)")
	if err := flags.Parse(args); err != nil || flags.NArg() > 0 || *limit < 0 {
		return rejectCall(flags, err, stdout, stderr)
	}
	if *kind != "" && !slices.Contains(eventKinds, *kind) {
		fmt.Fprintf(stderr, "ele: unknown kind %q; use %s\n", *kind, strings.Join(eventKinds, ", "))
		return 2
	}
	from, err := parseSince(*since, time.Now())
	if err != nil {
		fmt.Fprintf(stderr, "ele: %v\n", err)
		return 2
	}
	dataDirectory, err := datadir.Path()
	if err != nil {
		fmt.Fprintf(stderr, "ele: %v\n", err)
		return 1
	}
	ownerMap, err := ownermap.Load(filepath.Join(dataDirectory, "owner-map.json"))
	if err != nil {
		fmt.Fprintf(stderr, "ele: %v\n", err)
		return 1
	}
	if *owner != "" && !slices.Contains(ownerMap.Owners, *owner) {
		fmt.Fprintf(stderr, "ele: unknown owner %q; owner-map.json lists %s\n", *owner, strings.Join(ownerMap.Owners, ", "))
		return 2
	}

	searchIndex, err := syncedIndex(ctx, *noSync, stderr, "these events")
	if err != nil {
		fmt.Fprintf(stderr, "ele: %v\n", err)
		return 1
	}
	defer searchIndex.Close()
	found, total, err := searchIndex.Events(ctx, dto.EventQuery{Kind: *kind, Limit: *limit, Name: *name, Owner: *owner, Project: *project, Session: *session, Since: from})
	if err != nil {
		fmt.Fprintf(stderr, "ele: %v\n", err)
		return 1
	}
	encoder := json.NewEncoder(stdout)
	for _, event := range found {
		if err = encoder.Encode(event); err != nil {
			fmt.Fprintf(stderr, "ele: %v\n", err)
			return 1
		}
	}
	if total > len(found) {
		fmt.Fprintf(stderr, "ele: %d of %d events shown, the newest; --limit 0 prints all, --since narrows\n", len(found), total)
	}
	return 0
}

func syncedIndex(ctx context.Context, noSync bool, stderr io.Writer, what string) (*index.Index, error) {
	if !noSync {
		switch syncErr := syncAll(ctx, io.Discard, false, false); {
		case errors.Is(syncErr, errSyncBusy):
		case syncErr != nil:
			fmt.Fprintf(stderr, "ele: sync failed, reading what is already indexed: %v\n", syncErr)
		}
	}
	searchIndex, err := openIndex()
	if err != nil {
		return nil, err
	}
	version, err := searchIndex.Setting(ctx, "index-version")
	if err != nil {
		searchIndex.Close()
		return nil, err
	}
	if version != index.Version {
		fmt.Fprintf(stderr, "ele: the index is not complete for this version of ele, so %s may be incomplete; run ele sync and wait for it\n", what)
	}
	return searchIndex, nil
}
