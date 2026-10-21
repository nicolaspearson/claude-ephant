package main

import (
	"runtime/debug"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuildVersion(t *testing.T) {
	tests := []struct {
		info    *debug.BuildInfo
		name    string
		stamped string
		want    string
	}{
		{info: &debug.BuildInfo{Main: debug.Module{Version: "(devel)"}}, name: "a build Go stamped no version of", want: "devel"},
		{info: &debug.BuildInfo{Main: debug.Module{Version: "v0.1.1"}}, name: "a release build's stamp wins", stamped: "v0.2.0", want: "v0.2.0"},
		{info: &debug.BuildInfo{Main: debug.Module{Version: "v0.2.0"}}, name: "go install at a version carries it", want: "v0.2.0"},
		{name: "no build information", want: "devel"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require.Equal(t, test.want, buildVersion(test.stamped, test.info))
		})
	}
}

func TestRunVersionPrintsOneLine(t *testing.T) {
	stdout, stderr, code := runSplit(t, "version")
	require.Equal(t, 0, code)
	require.Empty(t, stderr)
	require.Equal(t, 1, strings.Count(stdout, "\n"))
}
