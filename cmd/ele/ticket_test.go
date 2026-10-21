package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExpandHomeKeepsDotDotAndLeavesOtherUsersAlone(t *testing.T) {
	t.Setenv("HOME", "/home/me")
	for path, want := range map[string]string{"/abs/x": "/abs/x", "x": "x", "~": "/home/me", "~/../x": "/home/me/../x", "~other/x": "~other/x"} {
		require.Equal(t, want, expandHome(path), path)
	}
}

func TestRunTicketClaimsRefusesATicketThatIsNotANumber(t *testing.T) {
	var output bytes.Buffer

	code := ticketCommand(context.Background(), []string{"claims", "--quiet", "12", "#36"}, &output, &output)

	require.Equal(t, 2, code)
	require.Equal(t, "ele: ticket \"#36\" is not a number\n", output.String())
}

func TestRunTicketClaimsRefusesAnUnknownFlag(t *testing.T) {
	var output bytes.Buffer

	code := ticketCommand(context.Background(), []string{"claims", "--all"}, &output, &output)

	require.Equal(t, 2, code)
	require.True(t, strings.HasPrefix(output.String(), "flag provided but not defined: -all\nele ticket claims [--quiet] [<number>...]\n"), output.String())
}

func TestRunTicketCorpusRefusesOptionsThatDoNotGoTogether(t *testing.T) {
	for args, message := range map[string]string{
		"--cli jira --tickets 1 --out o": "ele: --cli \"jira\" is not glab or gh\n",
		"--compare 1":                    "ele: --compare takes --expect and no corpus options\n",
		"--compare 1 --expect e --clean": "ele: --compare takes --expect and no corpus options\n",
		"--compare x --expect e":         "ele: --compare \"x\" is not a number\n",
		"--map 0 --out o":                "ele: one of --plan, or --map and --tickets, is required\n",
		"--map 1 --plan p --out o":       "ele: one of --plan, or --map and --tickets, is required\n",
		"--map x --out o":                "ele: --map \"x\" is not a number\n",
		"--note --tickets 1 --out o":     "ele: --note and --expect go with --compare\n",
		"--tickets 1":                    "ele: --out is required\n",
		"--tickets 1,#2 --out o":         "ele: ticket \"#2\" is not a number\n",
	} {
		var output bytes.Buffer

		code := ticketCommand(context.Background(), append([]string{"corpus"}, strings.Fields(args)...), &output, &output)

		require.Equal(t, 2, code, args)
		require.Equal(t, message, output.String(), args)
	}
}

func TestRunTicketLedgerNeedsACommandAManifestAndForRecordTheRecords(t *testing.T) {
	for _, args := range [][]string{{"ledger"}, {"ledger", "record", "rulings", "--manifest", "m.json"}, {"ledger", "show", "--manifest", "m.json"}} {
		var output bytes.Buffer

		code := ticketCommand(context.Background(), args, &output, &output)

		require.Equal(t, 2, code, args)
		require.True(t, strings.HasPrefix(output.String(), "ele ticket ledger record --manifest <file> --records <file> [--ledger <file>]\n"), output.String())
	}
	for args, message := range map[string]string{"record --manifest m.json": "ele: ticket ledger record needs --records\n", "rulings": "ele: ticket ledger needs --manifest\n"} {
		var output bytes.Buffer

		code := ticketCommand(context.Background(), append([]string{"ledger"}, strings.Fields(args)...), &output, &output)

		require.Equal(t, 2, code, args)
		require.Equal(t, message, output.String(), args)
	}
}

func TestRunTicketLedgerReadsTheDefaultLedgerUnderHome(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	require.NoError(t, os.MkdirAll(filepath.Join(home, ".claude", "logs"), 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(home, ".claude", "logs", "ticket-audits.jsonl"), []byte(`{"audited":"2026-10-01","hash":"h","key":"k","methods":[],"rulings":[{"finding":"f","ruling":"r"}]}`+"\n"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(home, "manifest.json"), []byte(`[{"hash":"h","key":"k"}]`), 0o600))
	var output bytes.Buffer

	code := ticketCommand(context.Background(), []string{"ledger", "rulings", "--manifest", "~/manifest.json"}, &output, &output)

	require.Equal(t, 0, code, output.String())
	require.Equal(t, "k\n  ruled 2026-10-01: f — r\n  text unchanged since\n", output.String())
}
