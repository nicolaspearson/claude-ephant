package tail

import (
	"bufio"
	"fmt"
	"os"
)

func Lines(path string, count int) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	if len(lines) <= count {
		return lines, nil
	}
	return lines[len(lines)-count+1:], nil
}
