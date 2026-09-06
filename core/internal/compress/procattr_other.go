//go:build !linux

package compress

import "os/exec"

// setDeathsig: no-op di luar Linux (Windows/macOS mengandalkan
// context timeout core; orphan saat wrapper-kill didokumentasikan).
func setDeathsig(cmd *exec.Cmd) {}
