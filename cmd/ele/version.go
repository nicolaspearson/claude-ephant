package main

import (
	"context"
	"fmt"
	"io"
	"runtime/debug"
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

func versionCommand(_ context.Context, args []string, stdout io.Writer, stderr io.Writer) int {
	flags := newFlagSet("version", stderr)
	if err := flags.Parse(args); err != nil || flags.NArg() > 0 {
		return rejectCall(flags, err, stdout, stderr)
	}
	info, _ := debug.ReadBuildInfo()
	fmt.Fprintln(stdout, buildVersion(version, info))
	return 0
}
