package plan

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

type Stored struct {
	Items []StoredItem `json:"items"`
	Path  string       `json:"path"`
}

type StoredItem struct {
	Done bool   `json:"done"`
	Name string `json:"name"`
}

func ReadStored(directory string, id string) (Stored, bool, error) {
	path := filepath.Join(directory, id+".json")
	content, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return Stored{}, false, nil
	}
	if err != nil {
		return Stored{}, false, err
	}
	var stored Stored
	if err = json.Unmarshal(content, &stored); err != nil {
		return Stored{}, false, fmt.Errorf("%s: %w", path, err)
	}
	return stored, true, nil
}

// WriteStored keeps only named items: an unnamed one can be neither a blocker nor a clash.
func WriteStored(directory string, id string, path string, parsed Plan) error {
	stored := Stored{Items: []StoredItem{}, Path: path}
	for _, action := range parsed.Actions {
		if action.Name != "" {
			stored.Items = append(stored.Items, StoredItem{Done: action.Done, Name: action.Name})
		}
	}
	content, err := json.MarshalIndent(stored, "", "  ")
	if err != nil {
		return err
	}
	if err = os.MkdirAll(directory, 0o755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(directory, id+".json"), append(content, '\n'), 0o644)
}
