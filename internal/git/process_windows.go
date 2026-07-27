//go:build windows

package git

import (
	"os/exec"
	"time"
)

func configureGitCommand(cmd *exec.Cmd) {
	cmd.WaitDelay = 250 * time.Millisecond
}
