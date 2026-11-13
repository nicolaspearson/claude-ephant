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
	"github.com/nicolaspearson/claude-ephant/internal/atomicfile"
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
	held, ok, err := watermarkOf(watermarks, analysis)
	if err != nil || (ok && !timestamp.After(held)) {
		return err
	}
	if watermarks[analysis], err = json.Marshal(timestamp.UTC()); err != nil {
		return err
	}
	content, err := json.MarshalIndent(watermarks, "", "  ")
	if err != nil {
		return err
	}
	return atomicfile.Write(filepath.Join(directory, watermarksFile), append(content, '\n'))
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
	watermark, _, err := watermarkOf(watermarks, analysis)
	return watermark, err
}

// loadWatermarks keeps each value raw, so a key a later format adds is neither read as a time nor dropped.
func loadWatermarks(directory string) (map[string]json.RawMessage, error) {
	path := filepath.Join(directory, watermarksFile)
	content, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return map[string]json.RawMessage{}, nil
	}
	if err != nil {
		return nil, err
	}
	watermarks := map[string]json.RawMessage{}
	if err = json.Unmarshal(content, &watermarks); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return watermarks, nil
}

func watermarkOf(watermarks map[string]json.RawMessage, analysis string) (time.Time, bool, error) {
	raw, ok := watermarks[analysis]
	if !ok {
		return time.Time{}, false, nil
	}
	var watermark time.Time
	if err := json.Unmarshal(raw, &watermark); err != nil {
		return time.Time{}, false, fmt.Errorf("%s %s: %w", watermarksFile, analysis, err)
	}
	return watermark, true, nil
}
