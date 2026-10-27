package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"text/tabwriter"

	"github.com/nicolaspearson/claude-ephant/internal/config"
	"github.com/nicolaspearson/claude-ephant/internal/datadir"
	"github.com/nicolaspearson/claude-ephant/internal/index"
	"github.com/nicolaspearson/claude-ephant/internal/ownermap"
)

// As of 2026-10-22 set by Claude Code in every process it starts, `!` commands included.
const sessionVariable = "CLAUDECODE"

type configScope struct {
	dataDirectory string
	owner         string
}

var configCommands = map[string]struct {
	arguments int
	run       func(scope configScope, arguments []string, stdout io.Writer, stderr io.Writer) error
	writes    bool
}{
	"get":   {arguments: 1, run: configGet},
	"list":  {arguments: 0, run: configList},
	"set":   {arguments: 2, run: configSet, writes: true},
	"unset": {arguments: 1, run: configUnset, writes: true},
}

func colourOn(settings config.Config, owner string) bool {
	value, err := settings.Value("colour", owner)
	return err != nil || value.Value == true
}

func compactHintAt(settings config.Config, owner string) int {
	value, err := settings.Value("compact_hint_at", owner)
	if threshold, ok := value.Value.(int); err == nil && ok {
		return threshold
	}
	option, _ := config.Lookup("compact_hint_at")
	return option.Default.(int)
}

func configCommand(_ context.Context, args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		return misuse("config", args, stdout, stderr)
	}
	command, ok := configCommands[args[0]]
	if !ok {
		return misuse("config", args, stdout, stderr)
	}
	flags := newFlagSet("config "+args[0], stderr)
	owner := flags.String("owner", "", "a listed owner; get and list default to the owner here, set and unset to the global value")
	arguments, err := parseInterspersed(flags, args[1:])
	if err != nil || len(arguments) != command.arguments {
		return rejectCall(flags, err, stdout, stderr)
	}
	// A deterrent, not a lock: Claude can still write config.json. A changed option shows in the start block.
	if command.writes && os.Getenv(sessionVariable) != "" {
		fmt.Fprintf(stderr, "ele: config %s changes ele for every session, so it refuses to run inside one; the user runs it outside the session: ele config %s\n", args[0], strings.Join(args, " "))
		return 1
	}
	scope, err := resolveConfigScope(*owner, !command.writes)
	if err == nil {
		err = command.run(scope, arguments, stdout, stderr)
	}
	if err != nil {
		fmt.Fprintf(stderr, "ele: %v\n", err)
		return 1
	}
	return 0
}

func configGet(scope configScope, arguments []string, stdout io.Writer, stderr io.Writer) error {
	value, err := loadConfig(scope.dataDirectory, stderr).Value(arguments[0], scope.owner)
	if err != nil {
		return err
	}
	fmt.Fprintln(stdout, value.Value)
	return nil
}

func configList(scope configScope, _ []string, stdout io.Writer, stderr io.Writer) error {
	settings := loadConfig(scope.dataDirectory, stderr)
	table := tabwriter.NewWriter(stdout, 0, 4, 2, ' ', 0)
	for _, option := range config.Options {
		value, err := settings.Value(option.Name, scope.owner)
		if err != nil {
			return err
		}
		fmt.Fprintf(table, "%s\t%s\t(%s)\n", option.Name, displayValue(value.Value), describeSource(settings, value))
		fmt.Fprintf(table, "  %s\n", option.Description)
	}
	if err := table.Flush(); err != nil {
		return err
	}
	if unknown := settings.Unknown(); len(unknown) > 0 {
		fmt.Fprintf(stdout, "Ignored, no such option: %s\n", strings.Join(unknown, ", "))
	}
	return nil
}

func configPath(dataDirectory string) string {
	return filepath.Join(dataDirectory, "config.json")
}

func configSet(scope configScope, arguments []string, stdout io.Writer, _ io.Writer) error {
	path := configPath(scope.dataDirectory)
	settings, err := config.Load(path)
	if err != nil {
		return fmt.Errorf("%w; fix or remove it before setting an option", err)
	}
	if err = settings.Set(arguments[0], scope.owner, arguments[1]); err != nil {
		return err
	}
	if err = os.MkdirAll(scope.dataDirectory, 0o700); err != nil {
		return err
	}
	if err = config.Save(path, settings); err != nil {
		return err
	}
	value, err := settings.Value(arguments[0], scope.owner)
	if err != nil {
		return err
	}
	fmt.Fprintf(stdout, "Set %s to %v %s.\n", arguments[0], value.Value, scopeName(scope.owner))
	return nil
}

func configUnset(scope configScope, arguments []string, stdout io.Writer, _ io.Writer) error {
	path := configPath(scope.dataDirectory)
	settings, err := config.Load(path)
	if err != nil {
		return fmt.Errorf("%w; fix or remove it before unsetting an option", err)
	}
	removed, err := settings.Unset(arguments[0], scope.owner)
	if err != nil {
		return err
	}
	if !removed {
		fmt.Fprintf(stdout, "%s has no value set %s.\n", arguments[0], scopeName(scope.owner))
		return nil
	}
	if err = config.Save(path, settings); err != nil {
		return err
	}
	fmt.Fprintf(stdout, "Unset %s %s.\n", arguments[0], scopeName(scope.owner))
	return nil
}

func describeSource(settings config.Config, value config.Value) string {
	if value.Source != config.SourceOwner {
		return value.Source
	}
	global, err := settings.Value(value.Name, "")
	if err != nil {
		return "owner " + value.Owner
	}
	return fmt.Sprintf("owner %s; %s %s", value.Owner, global.Source, displayValue(global.Value))
}

// Quoted so an empty value does not read as missing.
func displayValue(value any) string {
	if value == "" {
		return `""`
	}
	return fmt.Sprint(value)
}

// A file that does not parse reads as defaults, so a broken edit leaves the checkpoint gate on.
func loadConfig(dataDirectory string, stderr io.Writer) config.Config {
	settings, err := config.Load(configPath(dataDirectory))
	if err != nil {
		fmt.Fprintf(stderr, "ele: %v; every option reads as its default\n", err)
		return config.Defaults()
	}
	return settings
}

func notesAskOn(settings config.Config, owner string) bool {
	value, err := settings.Value("notes_ask", owner)
	return err != nil || value.Value == true
}

func offDefaults(settings config.Config, owner string) []string {
	var lines []string
	for _, option := range config.Options {
		value, err := settings.Value(option.Name, owner)
		if err != nil || value.Value == option.Default {
			continue
		}
		lines = append(lines, fmt.Sprintf("%s %v (%s)", option.Name, value.Value, describeSource(settings, value)))
	}
	return lines
}

func resolveConfigScope(owner string, here bool) (configScope, error) {
	dataDirectory, err := datadir.Path()
	if err != nil {
		return configScope{}, err
	}
	ownerMap, err := ownermap.Load(filepath.Join(dataDirectory, "owner-map.json"))
	if err != nil {
		return configScope{}, err
	}
	if owner != "" {
		if !slices.Contains(ownerMap.Owners, owner) {
			return configScope{}, fmt.Errorf("owner-map.json does not list owner %q; ele owner add %s lists it", owner, owner)
		}
		return configScope{dataDirectory: dataDirectory, owner: owner}, nil
	}
	if !here {
		return configScope{dataDirectory: dataDirectory}, nil
	}
	directory, err := os.Getwd()
	if err != nil {
		return configScope{}, err
	}
	_, owner = index.NewProjects(ownerMap).Resolve(directory)
	return configScope{dataDirectory: dataDirectory, owner: owner}, nil
}

func scopeName(owner string) string {
	if owner == "" {
		return "globally"
	}
	return "for owner " + owner
}
