package main

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"runtime/debug"
	"time"

	"github.com/nicolaspearson/claude-ephant/internal/datadir"
	"github.com/nicolaspearson/claude-ephant/internal/hookversion"
)

// Stamped by the release build: -ldflags "-X main.version=<tag>".
var version string

func buildVersion(stamped string, info *debug.BuildInfo) string {
	switch {
	case stamped != "":
		return stamped
	case info != nil && info.Main.Version != "" && info.Main.Version != "(devel)":
		return info.Main.Version
	default:
		return "devel"
	}
}

// Failure dropped: the record only labels events, never worth failing a hook.
func recordHookVersion(session string) {
	dataDirectory, err := datadir.Path()
	if err != nil {
		return
	}
	info, _ := debug.ReadBuildInfo()
	_ = hookversion.Record(filepath.Join(dataDirectory, hookversion.Directory), session, buildVersion(version, info), time.Now())
}

func versionCommand(_ context.Context, args []string, stdout io.Writer, stderr io.Writer) int {
	flags := newFlagSet("version", stderr)
	if err := flags.Parse(args); err != nil || flags.NArg() > 0 {
		return rejectCall(flags, err, stdout, stderr)
	}
	info, _ := debug.ReadBuildInfo()
	fmt.Fprintln(stdout, buildVersion(version, info))
	return 0
}
