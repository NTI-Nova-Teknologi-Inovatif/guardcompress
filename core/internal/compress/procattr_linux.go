//go:build linux

package compress

import (
	"os/exec"
	"syscall"
)

func setDeathsig(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Pdeathsig: syscall.SIGKILL}
}
