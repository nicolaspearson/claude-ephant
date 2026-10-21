package checkpoint

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ReviewedFile must match the Bash freeze's key: it writes `.started` beside this path.
func ReviewedFile(ctx context.Context, directory string) (string, error) {
	root, err := Root(ctx, directory)
	if err != nil {
		return "", err
	}
	reviewed := os.Getenv("CHECKPOINT_REVIEWED_DIR")
	if reviewed == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		reviewed = filepath.Join(home, ".claude", "cache", "checkpoint-reviewed")
	}
	if err := os.MkdirAll(reviewed, 0o700); err != nil {
		return "", err
	}
	sum := sha1.Sum([]byte(root))
	return filepath.Join(reviewed, hex.EncodeToString(sum[:])), nil
}

func Root(ctx context.Context, directory string) (string, error) {
	root, err := git(ctx, directory, "rev-parse", "--show-toplevel")
	if err != nil {
		return "", err
	}
	return strings.TrimRight(root, "\n"), nil
}

func git(ctx context.Context, directory string, args ...string) (string, error) {
	var stdout, stderr bytes.Buffer
	command := exec.CommandContext(ctx, "git", append([]string{"-C", directory}, args...)...)
	command.Stdout, command.Stderr = &stdout, &stderr
	if err := command.Run(); err != nil {
		return "", fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(stderr.String()))
	}
	return stdout.String(), nil
}
