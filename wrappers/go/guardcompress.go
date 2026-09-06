// Package guardcompress: Go SDK tipis (os/exec wrapper).
package guardcompress

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

type Result struct {
	Path   string
	Report map[string]any
}

// Process memanggil binary core. Set GUARDCOMPRESS_BIN atau taruh binary di core/bin.
func Process(inPath string, opts map[string]any) (Result, error) {
	bin := os.Getenv("GUARDCOMPRESS_BIN")
	if bin == "" {
		return Result{}, fmt.Errorf("set GUARDCOMPRESS_BIN ke path binary guardcompress")
	}
	if opts == nil {
		opts = map[string]any{}
	}
	cfg, _ := json.Marshal(opts)
	outDir, err := os.MkdirTemp("", "gc-")
	if err != nil {
		return Result{}, err
	}
	// AUDIT: timeout wrapper (120s) harus > timeout ffmpeg core (100s)
	// agar core yang selalu menuai ffmpeg, bukan wrapper.
	timeout := 120 * time.Second
	if v, ok := opts["timeoutSec"]; ok {
		switch n := v.(type) {
		case int:
			if n > 0 {
				timeout = time.Duration(n) * time.Second
			}
		case float64:
			if n > 0 {
				timeout = time.Duration(n * float64(time.Second))
			}
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, bin, "check", "--in", inPath, "--out-dir", outDir, "--config", string(cfg), "--json")
	out, err := cmd.Output()
	var report map[string]any
	_ = json.Unmarshal(lastLine(out), &report)
	if err != nil {
		os.RemoveAll(outDir) // gagal/blocked: buang output
		if ee, ok := err.(*exec.ExitError); ok && ee.ExitCode() == 2 {
			return Result{}, fmt.Errorf("blocked: %v", report["reason"])
		}
		return Result{}, fmt.Errorf("guardcompress failed: %v (%s)", err, string(out))
	}
	p, _ := report["out_path"].(string)
	if p == "" {
		p = filepath.Join(outDir, "output")
	}
	return Result{Path: p, Report: report}, nil
}

func lastLine(b []byte) []byte {
	lines := 0
	for i := len(b) - 1; i >= 0; i-- {
		if b[i] == '\n' {
			lines++
			if lines == 1 && i == len(b)-1 {
				continue
			}
			return b[i+1:]
		}
	}
	return b
}
