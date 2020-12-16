package git

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func Head(dir string) (string, error) {
	//git rev-parse HEAD
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = dir
	cmd.Env = os.Environ()

	res, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("unable to call git: %w", err)
	}

	return strings.TrimSpace(string(res)), nil
}
