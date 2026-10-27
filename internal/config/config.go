package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
)

const (
	KindBoolean = "boolean"
	KindInteger = "integer"
	KindPath    = "path"

	SourceDefault = "default"
	SourceGlobal  = "global"
	SourceOwner   = "owner"
)

// Top-level key holding the per-owner overrides; no option may take it.
const ownersKey = "owners"

type Option struct {
	Default     any
	Description string
	Kind        string
	Name        string
}

// Config keeps unknown keys, so a file written by a newer ele survives a set from an older one.
type Config struct {
	Global map[string]json.RawMessage
	Owners map[string]map[string]json.RawMessage
}

type Value struct {
	Name   string
	Owner  string // set when Source is SourceOwner
	Source string
	Value  any
}

var Options = []Option{
	{
		Default:     false,
		Description: "Writes the notes ele:remember proposes without asking Write or Skip, then reports each.",
		Kind:        KindBoolean,
		Name:        "auto_note_writes",
	},
	{
		Default:     true,
		Description: "Blocks a turn that changed files and claims done without running ele:checkpoint.",
		Kind:        KindBoolean,
		Name:        "checkpoint_gate",
	},
	{
		Default:     true,
		Description: "Colours the messages ele shows the user, the start block's and the Stop hook's; NO_COLOR turns it off whatever this says.",
		Kind:        KindBoolean,
		Name:        "colour",
	},
	{
		Default:     150000,
		Description: "Context tokens from which the Stop hook suggests /compact or a handoff, again every 100k past it; 0 turns the hint off.",
		Kind:        KindInteger,
		Name:        "compact_hint_at",
	},
	{
		Default:     true,
		Description: "Asks for notes in the Stop hook once per stretch of work; off, /ele:remember still works.",
		Kind:        KindBoolean,
		Name:        "notes_ask",
	},
	{
		Default:     true,
		Description: "Shows each open reminder in the start block from its due date until closed.",
		Kind:        KindBoolean,
		Name:        "reminders",
	},
	{
		Default:     "",
		Description: "An absolute or ~/ path to the voice file ele:review drafts comments from, in place of the shipped one; empty for the shipped one.",
		Kind:        KindPath,
		Name:        "review_voice",
	},
}

func Defaults() Config {
	return Config{Global: map[string]json.RawMessage{}, Owners: map[string]map[string]json.RawMessage{}}
}

func Load(path string) (Config, error) {
	config := Defaults()
	content, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return config, nil
	}
	if err != nil {
		return config, fmt.Errorf("read %s: %w", path, err)
	}
	var top map[string]json.RawMessage
	if err = json.Unmarshal(content, &top); err != nil {
		return config, fmt.Errorf("parse %s: %w", path, err)
	}
	if owners, ok := top[ownersKey]; ok {
		delete(top, ownersKey)
		if err = json.Unmarshal(owners, &config.Owners); err != nil {
			return config, fmt.Errorf("parse %s: %s: %w", path, ownersKey, err)
		}
	}
	if top != nil {
		config.Global = top
	}
	// "owners": null unmarshals to a nil map, which Set would write to.
	if config.Owners == nil {
		config.Owners = map[string]map[string]json.RawMessage{}
	}
	for owner, values := range config.Owners {
		if values == nil {
			config.Owners[owner] = map[string]json.RawMessage{}
		}
	}
	for scope, values := range config.scopes() {
		for name, raw := range values {
			option, known := Lookup(name)
			if !known {
				continue
			}
			if _, err = decode(option, raw); err != nil {
				return config, fmt.Errorf("parse %s: %s%s: %w", path, scope, name, err)
			}
		}
	}
	return config, nil
}

func Lookup(name string) (Option, bool) {
	index := slices.IndexFunc(Options, func(option Option) bool { return option.Name == name })
	if index < 0 {
		return Option{}, false
	}
	return Options[index], true
}

func Save(path string, config Config) error {
	top := maps.Clone(config.Global)
	if top == nil {
		top = map[string]json.RawMessage{}
	}
	owners := map[string]map[string]json.RawMessage{}
	for owner, values := range config.Owners {
		if len(values) > 0 {
			owners[owner] = values
		}
	}
	if len(owners) > 0 {
		encoded, err := json.Marshal(owners)
		if err != nil {
			return fmt.Errorf("encode config: %w", err)
		}
		top[ownersKey] = encoded
	}
	content, err := json.MarshalIndent(top, "", "  ")
	if err != nil {
		return fmt.Errorf("encode config: %w", err)
	}
	temporary := path + ".tmp"
	if err = os.WriteFile(temporary, append(content, '\n'), 0o600); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	if err = os.Rename(temporary, path); err != nil {
		_ = os.Remove(temporary)
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

func (c *Config) Set(name string, owner string, text string) error {
	option, known := Lookup(name)
	if !known {
		return unknownOption(name)
	}
	parsed, err := parse(option, text)
	if err != nil {
		return err
	}
	encoded, err := json.Marshal(parsed)
	if err != nil {
		return err
	}
	if owner == "" {
		c.Global[name] = encoded
		return nil
	}
	if c.Owners[owner] == nil {
		c.Owners[owner] = map[string]json.RawMessage{}
	}
	c.Owners[owner][name] = encoded
	return nil
}

func (c Config) Unknown() []string {
	var unknown []string
	for scope, values := range c.scopes() {
		for name := range values {
			if _, known := Lookup(name); !known {
				unknown = append(unknown, scope+name)
			}
		}
	}
	slices.Sort(unknown)
	return unknown
}

func (c *Config) Unset(name string, owner string) (bool, error) {
	if _, known := Lookup(name); !known {
		return false, unknownOption(name)
	}
	values := c.Global
	if owner != "" {
		values = c.Owners[owner]
	}
	if _, stored := values[name]; !stored {
		return false, nil
	}
	delete(values, name)
	return true, nil
}

func (c Config) Value(name string, owner string) (Value, error) {
	option, known := Lookup(name)
	if !known {
		return Value{}, unknownOption(name)
	}
	if raw, stored := c.Owners[owner][name]; owner != "" && stored {
		value, err := decode(option, raw)
		return Value{Name: name, Owner: owner, Source: SourceOwner, Value: value}, err
	}
	if raw, stored := c.Global[name]; stored {
		value, err := decode(option, raw)
		return Value{Name: name, Source: SourceGlobal, Value: value}, err
	}
	return Value{Name: name, Source: SourceDefault, Value: option.Default}, nil
}

func (c Config) scopes() map[string]map[string]json.RawMessage {
	scopes := map[string]map[string]json.RawMessage{"": c.Global}
	for owner, values := range c.Owners {
		scopes[ownersKey+"."+owner+"."] = values
	}
	return scopes
}

func decode(option Option, raw json.RawMessage) (any, error) {
	switch option.Kind {
	case KindBoolean:
		var value bool
		if err := json.Unmarshal(raw, &value); err != nil {
			return nil, fmt.Errorf("want a boolean, got %s", raw)
		}
		return value, nil
	case KindInteger:
		// null unmarshals into an int as 0, which would turn an option off unasked.
		var value int
		if err := json.Unmarshal(raw, &value); err != nil || value < 0 || string(raw) == "null" {
			return nil, fmt.Errorf("want a whole number of 0 or more, got %s", raw)
		}
		return value, nil
	case KindPath:
		var value string
		if err := json.Unmarshal(raw, &value); err != nil {
			return nil, fmt.Errorf("want a path, got %s", raw)
		}
		return expandPath(value)
	}
	return nil, fmt.Errorf("option %s has unknown kind %q", option.Name, option.Kind)
}

func expandPath(path string) (string, error) {
	if rest, ok := strings.CutPrefix(path, "~/"); ok {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, rest), nil
	}
	if !filepath.IsAbs(path) {
		return "", fmt.Errorf("want an absolute or ~/ path, got %q", path)
	}
	return path, nil
}

func parse(option Option, text string) (any, error) {
	switch option.Kind {
	case KindBoolean:
		value, err := strconv.ParseBool(text)
		if err != nil {
			return nil, fmt.Errorf("%s takes true or false, not %q", option.Name, text)
		}
		return value, nil
	case KindInteger:
		value, err := strconv.Atoi(text)
		if err != nil || value < 0 {
			return nil, fmt.Errorf("%s takes a whole number of 0 or more, not %q", option.Name, text)
		}
		return value, nil
	case KindPath:
		path, err := expandPath(text)
		if err != nil {
			return nil, fmt.Errorf("%s takes an absolute or ~/ path, not %q; ele config unset %s returns it to its default", option.Name, text, option.Name)
		}
		if err = readableFile(path); err != nil {
			return nil, fmt.Errorf("%s: %w", option.Name, err)
		}
		// Stored as typed, so a config.json holding ~/ stays valid on a machine with another home.
		return text, nil
	}
	return nil, fmt.Errorf("option %s has unknown kind %q", option.Name, option.Kind)
}

// Stat before Open: opening a FIFO blocks until a writer appears.
func readableFile(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("%s is not a regular file", path)
	}
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	return file.Close()
}

func unknownOption(name string) error {
	names := make([]string, 0, len(Options))
	for _, option := range Options {
		names = append(names, option.Name)
	}
	return fmt.Errorf("no option %q; options: %s", name, strings.Join(names, ", "))
}
