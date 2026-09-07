// Package guardcompress: Go SDK tipis (os/exec wrapper).
package guardcompress

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"sync"
	"time"
)

type Result struct {
	Path   string
	Report map[string]any
}

// Preset per jenis (1 sistem di belakangnya, opts user menang bila menimpa).
func Image(inPath string, opts map[string]any) (Result, error) {
	return Process(inPath, withDefault(opts, "allow_ext",
		[]any{"jpg", "jpeg", "png", "webp", "gif"}))
}

func Video(inPath string, opts map[string]any) (Result, error) {
	return Process(inPath, withDefault(opts, "allow_ext",
		[]any{"mp4", "mov", "webm", "mkv", "avi"}))
}

func Audio(inPath string, opts map[string]any) (Result, error) {
	return Process(inPath, withDefault(opts, "allow_ext",
		[]any{"mp3", "wav", "ogg", "oga", "m4a", "flac"}))
}

func withDefault(opts map[string]any, key string, val any) map[string]any {
	out := map[string]any{}
	for k, v := range opts {
		out[k] = v
	}
	if _, ok := out[key]; !ok {
		out[key] = val
	}
	return out
}

// BatchItem: satu input dalam batch (Opts boleh nil = pakai bersama).
type BatchItem struct {
	Path string
	Opts map[string]any
}

// BatchResult: hasil per input. Err==nil artinya ok; ErrBlocked menandai ditolak.
type BatchResult struct {
	Path   string
	Report map[string]any
	Err    error
}

// IsBlocked: true bila file ditolak scanner (bukan error teknis).
func (b BatchResult) IsBlocked() bool {
	return IsBlocked(b.Err)
}

func isBusyReport(report map[string]any) bool {
	if report == nil {
		return false
	}
	d, ok := report["details"].(map[string]any)
	if !ok {
		return false
	}
	b, _ := d["busy"].(bool)
	return b
}

// IsBlocked: true bila file ditolak scanner (bukan error teknis).
func IsBlocked(err error) bool {
	return err != nil && len(err.Error()) >= 7 && err.Error()[:7] == "blocked"
}

// IsBusy: true bila server penuh (backpressure) -> retry nanti (HTTP 429).
func IsBusy(err error) bool {
	return err != nil && len(err.Error()) >= 5 && err.Error()[:5] == "busy:"
}

// Batch: multi-input beda jenis sekaligus. File ditolak terkumpul per item.
// Mode sekuensial (default): error teknis menghentikan langsung (fail-fast).
// Mode paralel (jobs>1): semua dijalankan, hasil terkumpul semua.
// Paralel bila opts["jobs"] > 1 (default 1 = sekuensial, hemat resource).
// Urutan hasil selalu sama dengan urutan input.
func Batch(items []BatchItem, opts map[string]any) []BatchResult {
	jobs := 1
	if v, ok := opts["jobs"]; ok {
		switch n := v.(type) {
		case int:
			if n > 1 {
				jobs = n
			}
		case float64:
			if n > 1 {
				jobs = int(n)
			}
		}
	}
	out := make([]BatchResult, len(items))
	if jobs == 1 {
		for i, it := range items {
			out[i] = runOne(it, opts)
			if out[i].Err != nil && !out[i].IsBlocked() {
				return out[:i+1] // error teknis: berhenti
			}
		}
		return out
	}
	sem := make(chan struct{}, jobs)
	var wg sync.WaitGroup
	for i, it := range items {
		wg.Add(1)
		go func(i int, it BatchItem) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			out[i] = runOne(it, opts)
		}(i, it)
	}
	wg.Wait()
	return out
}

func runOne(it BatchItem, opts map[string]any) BatchResult {
	merged := map[string]any{}
	for k, v := range opts {
		if k == "jobs" {
			continue
		}
		merged[k] = v
	}
	for k, v := range it.Opts {
		merged[k] = v
	}
	r, err := Process(it.Path, merged)
	return BatchResult{Path: r.Path, Report: r.Report, Err: err}
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
		// Sinyal busy (backpressure): server penuh -> retry, bukan vonis jahat.
		if isBusyReport(report) {
			return Result{}, fmt.Errorf("busy: %v", report["reason"])
		}
		return Result{}, fmt.Errorf("guardcompress failed: %v (%s)", err, string(out))
	}
	p, _ := report["out_path"].(string)
	// Gagal cepat di batas: fallback nama lama dihapus, error eksplisit.
	if p == "" {
		os.RemoveAll(outDir)
		return Result{}, fmt.Errorf("guardcompress: out_path hilang dari report")
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
