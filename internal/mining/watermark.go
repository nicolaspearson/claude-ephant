package mining

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/nicolaspearson/claude-ephant/internal/archive"
)

const watermarksFile = "watermarks.json"

func Advance(directory string, analysis string, timestamp time.Time) error {
	if err := checkAnalysis(analysis); err != nil {
		return err
	}
	unlock, err := archive.Lock(directory)
	if err != nil {
		return err
	}
	defer unlock()
	watermarks, err := loadWatermarks(directory)
	if err != nil {
		return err
	}
	if held, ok := watermarks[analysis]; ok && !timestamp.After(held) {
		return nil
	}
	watermarks[analysis] = timestamp.UTC()
	content, err := json.MarshalIndent(watermarks, "", "  ")
	if err != nil {
		return err
	}
	return writeAtomically(filepath.Join(directory, watermarksFile), append(content, '\n'))
}

// Watermark returns the zero time for an analysis never mined.
func Watermark(directory string, analysis string) (time.Time, error) {
	if err := checkAnalysis(analysis); err != nil {
		return time.Time{}, err
	}
	watermarks, err := loadWatermarks(directory)
	if err != nil {
		return time.Time{}, err
	}
	return watermarks[analysis], nil
}

func loadWatermarks(directory string) (map[string]time.Time, error) {
	path := filepath.Join(directory, watermarksFile)
	content, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return map[string]time.Time{}, nil
	}
	if err != nil {
		return nil, err
	}
	watermarks := map[string]time.Time{}
	if err = json.Unmarshal(content, &watermarks); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return watermarks, nil
}
