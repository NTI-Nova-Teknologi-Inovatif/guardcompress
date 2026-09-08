//go:build !linux

package compress

import "os/exec"

func setDeathsig(cmd *exec.Cmd) {}
