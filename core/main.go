// Command guardcompress: single-binary CLI Core Engine.
// Contract (STABLE):
//   guardcompress check --in <path> --out-dir <dir> [--config <json>] [--json]
//   exit 0 = clean, exit 2 = blocked, exit 1 = error
//   stdout = report.json
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/guardcompress/guardcompress/core/internal/compress"
	"github.com/guardcompress/guardcompress/core/internal/guard"
)

type Report struct {
	Status   string         `json:"status"` // clean | blocked
	InPath   string         `json:"in_path"`
	OutPath  string         `json:"out_path,omitempty"`
	Detected string         `json:"detected_mime"`
	OrigBytes int64         `json:"orig_bytes"`
	NewBytes  int64         `json:"new_bytes,omitempty"`
	TookMs   int64          `json:"took_ms"`
	Reason   string         `json:"reason,omitempty"`
	Details  map[string]any `json:"details,omitempty"`
}

func main() {
	if len(os.Args) < 2 || os.Args[1] != "check" {
		fmt.Fprintln(os.Stderr, "usage: guardcompress check --in <path> --out-dir <dir> [--config <json>] [--json]")
		os.Exit(1)
	}

	fs := flag.NewFlagSet("check", flag.ExitOnError)
	inPath := fs.String("in", "", "input file path")
	outDir := fs.String("out-dir", "", "output directory")
	configStr := fs.String("config", "{}", "JSON config: {\"max_mb\":500,\"allow\":[],\"video_crf\":28}")
	_ = fs.Bool("json", false, "output report as JSON to stdout")
	_ = fs.Parse(os.Args[2:])

	start := time.Now()
	report := Report{InPath: *inPath, Details: map[string]any{}}

	fail := func(msg string, code int) {
		report.Status = "blocked"
		if code == 1 {
			report.Status = "error"
		}
		report.Reason = msg
		report.TookMs = time.Since(start).Milliseconds()
		b, _ := json.Marshal(report)
		fmt.Println(string(b))
		os.Exit(code)
	}

	if *inPath == "" || *outDir == "" {
		fail("missing --in or --out-dir", 1)
	}
	var cfg map[string]any
	if err := json.Unmarshal([]byte(*configStr), &cfg); err != nil {
		fail("invalid --config JSON: "+err.Error(), 1)
	}
	if err := os.MkdirAll(*outDir, 0o755); err != nil {
		fail("cannot create out-dir: "+err.Error(), 1)
	}

	// ---- GUARD PHASE ----
	gres, err := guard.Scan(*inPath, cfg)
	if err != nil {
		fail("guard error: "+err.Error(), 1)
	}
	report.Detected = gres.Mime
	report.OrigBytes = gres.Size
	report.Details["guard"] = gres.Details
	if !gres.Allowed {
		fail("blocked: "+gres.Reason, 2)
	}

	// ---- COMPRESS PHASE ----
	outPath := filepath.Join(*outDir, "output"+gres.SafeExt())
	cres, err := compress.Run(*inPath, outPath, gres.Mime, cfg)
	if err != nil {
		fail("compress error: "+err.Error(), 1)
	}
	report.OutPath = outPath
	report.NewBytes = cres.NewBytes
	report.Details["compress"] = cres.Details

	report.Status = "clean"
	report.TookMs = time.Since(start).Milliseconds()
	b, _ := json.Marshal(report)
	fmt.Println(string(b))
}
