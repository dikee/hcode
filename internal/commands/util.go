package commands

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/dikee/hetzner-code/internal/run"
)

// expandUser expands a leading "~" the way Python's Path.expanduser
// does.
func expandUser(p string) (string, error) {
	if p == "~" || strings.HasPrefix(p, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, strings.TrimPrefix(p, "~")), nil
	}
	return p, nil
}

// confirm prompts prompt + " [y/N]: " and aborts with an *HcodeError if
// the answer isn't affirmative — mirrors click.confirm(..., abort=True).
func confirm(prompt string) error {
	fmt.Printf("%s [y/N]: ", prompt)
	reader := bufio.NewReader(os.Stdin)
	line, _ := reader.ReadString('\n')
	line = strings.ToLower(strings.TrimSpace(line))
	if line == "y" || line == "yes" {
		return nil
	}
	return run.Errorf("aborted!")
}
