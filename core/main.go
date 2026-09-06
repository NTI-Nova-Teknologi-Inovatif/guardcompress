// Command guardcompress: single-binary CLI Core Engine.
// Contract (STABLE):
//
//	guardcompress check --in <path> --out-dir <dir> [--config <json>] [--json]
//	exit 0 = clean, exit 2 = blocked, exit 1 = error
//	stdout (check --json) = report.json di baris terakhir
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

var Version = "v0.1.0"

type Report struct {
	Status    string         `json:"status"` // clean | blocked | error
	InPath    string         `json:"in_path"`
	OutPath   string         `json:"out_path,omitempty"`
	Detected  string         `json:"detected_mime"`
	OrigBytes int64          `json:"orig_bytes"`
	NewBytes  int64          `json:"new_bytes,omitempty"`
	TookMs    int64          `json:"took_ms"`
	Reason    string         `json:"reason,omitempty"`
	Details   map[string]any `json:"details,omitempty"`
}

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}
	switch os.Args[1] {
	case "check":
		runCheck()
	case "doctor":
		runDoctor()
	case "init":
		runInit()
	case "version", "--version", "-v":
		fmt.Println("guardcompress " + Version)
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n", os.Args[1])
		usage()
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, `guardcompress `+Version+`
usage:
  guardcompress check --in <path> --out-dir <dir> [--config <json>] [--json]
  guardcompress doctor                      # cek ffmpeg + env, output JSON
  guardcompress init --lang php|node|python|go  # cetak contoh integrasi
  guardcompress version`)
}

func runCheck() {
	fs := flag.NewFlagSet("check", flag.ExitOnError)
	inPath := fs.String("in", "", "input file path")
	outDir := fs.String("out-dir", "", "output directory")
	configStr := fs.String("config", "{}", "JSON config")
	_ = fs.Bool("json", false, "output report as JSON")
	_ = fs.Parse(os.Args[2:])

	start := time.Now()
	report := Report{InPath: *inPath, Details: map[string]any{}}
	// Struktur output di out-dir (gampang dicek manual):
	//   <nama-aman>.<ext> + report.json (selalu ditulis, blocked pun ada jejaknya)
	saveReport := func() {
		if *outDir == "" {
			return
		}
		b, _ := json.MarshalIndent(report, "", "  ")
		_ = os.WriteFile(filepath.Join(*outDir, "report.json"), b, 0o644)
	}
	fail := func(msg string, code int) {
		report.Status = "blocked"
		if code == 1 {
			report.Status = "error"
		}
		report.Reason = msg
		report.TookMs = time.Since(start).Milliseconds()
		saveReport()
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

	// AUDIT TOCTOU: file bisa diganti penyerang di antara scan & kompres
	// (sharing tmp). Batalkan bila ukuran berubah setelah scan.
	if fi2, err := os.Stat(*inPath); err != nil || fi2.Size() != gres.Size {
		fail("input changed after scan (possible race), abort", 1)
	}

	outPath := filepath.Join(*outDir, guard.OutputName(*inPath, gres.Mime, cfg))
	cres, err := compress.Run(*inPath, outPath, gres.Mime, cfg)
	if err != nil {
		fail("compress error: "+err.Error(), 1)
	}
	report.OutPath = outPath
	report.NewBytes = cres.NewBytes
	report.Details["compress"] = cres.Details

	report.Status = "clean"
	report.TookMs = time.Since(start).Milliseconds()
	saveReport()
	b, _ := json.Marshal(report)
	fmt.Println(string(b))
}

func runDoctor() {
	ff := compress.FindFFmpeg()
	info := map[string]any{
		"version":      Version,
		"ffmpeg":       ff,
		"ffmpeg_found": ff != "",
		"cache_dir":    compress.CacheDir(),
	}
	if ff != "" {
		info["ffmpeg_version"] = compress.ProbeFFmpegVersion(ff)
	} else {
		info["hint"] = "ffmpeg tidak ketemu, mode guard-only (copy). Jalankan installer wrapper / set GUARDCOMPRESS_FFMPEG."
	}
	b, _ := json.MarshalIndent(info, "", "  ")
	fmt.Println(string(b))
}

func runInit() {
	fs := flag.NewFlagSet("init", flag.ExitOnError)
	lang := fs.String("lang", "php", "php|node|python|go")
	_ = fs.Parse(os.Args[2:])
	fmt.Println(snippetFor(*lang))
}

func snippetFor(lang string) string {
	switch lang {
	case "node":
		return `// Node/Express + Multer
const gc = require('guardcompress');
const r = gc.process(req.file.path, { max_mb: 500, video_crf: 28 });
// simpan r.path ke S3, try/catch BLOCKED -> 422`
	case "python":
		return `# Django/Flask
from guardcompress import process, BlockedError
try:
    r = process(tmp_path, {"max_mb": 500, "video_crf": 28})
except BlockedError as e:
    return 422, str(e)`
	case "go":
		return `// Go net/http
import gc "github.com/guardcompress/guardcompress/wrappers/go"
res, err := gc.Process(tmpPath, map[string]any{"max_mb": 500})`
	default:
		return `<?php
// Laravel: tempel ke UploadController@store
use GuardCompress\GuardCompress;
try {
    $r = GuardCompress::process($request->file('video')->getRealPath(), ['max_mb'=>500,'video_crf'=>28]);
    $path = Storage::putFile('media', new File($r->path));
} catch (\GuardCompress\InfectedFileException $e) {
    return response()->json(['blocked'=>$e->getMessage()], 422);
}`
	}
}
