//go:build linux

package compress

import (
	"os/exec"
	"syscall"
)

// setDeathsig: ffmpeg menerima SIGKILL otomatis bila proses core mati.
// Mencegah orphan ffmpeg saat wrapper/core di-kill paksa.
func setDeathsig(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Pdeathsig: syscall.SIGKILL}
}
