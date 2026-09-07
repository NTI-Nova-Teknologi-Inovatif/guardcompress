// Command guardcompress: single-binary CLI Core Engine.
// Contract (STABLE):
//
//	guardcompress check --in <path> --out-dir <dir> [--config <json>] [--json]
//	exit 0 = clean, exit 2 = blocked, exit 1 = error
//	stdout (check --json) = report.json di baris terakhir
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"time"

	"github.com/guardcompress/guardcompress/core/internal/compress"
	"github.com/guardcompress/guardcompress/core/internal/guard"
	"github.com/guardcompress/guardcompress/core/internal/slots"
)

var Version = "v0.1.0"

type Report struct {
	Status    string           `json:"status"` // clean | blocked | error
	InPath    string           `json:"in_path"`
	OutPath   string           `json:"out_path,omitempty"`
	Detected  string           `json:"detected_mime"`
	OrigBytes int64            `json:"orig_bytes"`
	NewBytes  int64            `json:"new_bytes,omitempty"`
	Thumbs    []compress.Thumb `json:"thumbs,omitempty"`
	SHA256    string           `json:"sha256,omitempty"` // sidik output (simpan di DB!)
	TookMs    int64            `json:"took_ms"`
	Reason    string           `json:"reason,omitempty"`
	Details   map[string]any   `json:"details,omitempty"`
}

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}
	switch os.Args[1] {
	case "check":
		runCheck()
	case "verify":
		runVerify()
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
    # ingest: isolasi -> scan -> kompres -> verifikasi output + sidik sha256
  guardcompress verify --in <path> [--config <json>] [--expect-sha256 <hex>] [--json]
    # audit simpanan: deteksi perubahan file SETELAH lolos (cron/queue berkala)
  guardcompress doctor                      # cek ffmpeg + env, output JSON
  guardcompress init --lang php|node|python|go  # cetak contoh integrasi
  guardcompress version`)
}

// fileSHA256: sidik streaming (memory konstan walau file 500MB).
func fileSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
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
	// Slot admission dilepas di SEMUA jalur keluar (fail memakai os.Exit,
	// jadi release eksplisit — bukan defer).
	var releaseSlot func()
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
		if releaseSlot != nil {
			releaseSlot()
		}
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

	// BACKPRESSURE: rebut slot lintas-proses SEBELUM kerja berat.
	// Penuh -> tolak cepat busy (HTTP 429), bukan terima lalu server tumbang.
	// Default = jumlah CPU; 0 = tanpa batas (server khusus media).
	maxSlots := runtime.NumCPU()
	if v, ok := cfg["max_slots"]; ok {
		switch n := v.(type) {
		case float64:
			maxSlots = int(n)
		case int:
			maxSlots = n
		}
	}
	rel, err := slots.Acquire(filepath.Join(compress.CacheDir(), "slots"), maxSlots)
	if err != nil {
		report.Details["busy"] = true
		fail("server busy, retry later", 1)
	}
	releaseSlot = rel

	gres, err := guard.Scan(*inPath, cfg)
	if err != nil {
		fail("guard error: "+err.Error(), 1)
	}
	report.Detected = gres.Mime
	report.OrigBytes = gres.Size
	report.Details["guard"] = gres.Details
	if !gres.Allowed {
		// Isi reason dulu agar salinan forensik di karantina lengkap.
		report.Reason = "blocked: " + gres.Reason
		quarantine(*inPath, &report, cfg)
		fail("blocked: "+gres.Reason, 2)
	}

	// AUDIT TOCTOU: file bisa diganti penyerang di antara scan & kompres
	// (sharing tmp). Batalkan bila ukuran ATAU waktu-ubah berubah setelah scan.
	if fi2, err := os.Stat(*inPath); err != nil || fi2.Size() != gres.Size ||
		fi2.ModTime().UnixNano() != gres.ModNano {
		fail("input changed after scan (possible race), abort", 1)
	}

	outPath := filepath.Join(*outDir, guard.OutputName(*inPath, gres.Mime, cfg))
	// AUDIT: tolak bila output = file input itu sendiri (CLI user bisa
	// mengarah --out-dir ke folder input; ffmpeg -y akan menghancurkan input).
	if same, _ := samePathFile(*inPath, outPath); same {
		fail("refusing: output path equals input (use a different --out-dir)", 1)
	}
	cres, err := compress.Run(*inPath, outPath, gres.Mime, cfg)
	if err != nil {
		fail("compress error: "+err.Error(), 1)
	}
	report.OutPath = outPath
	report.NewBytes = cres.NewBytes
	report.Thumbs = cres.Thumbs
	report.Details["compress"] = cres.Details

	// DETEKSI PERUBAHAN FORMAT: sniff ulang HASIL kompres, keluarganya harus
	// sama dengan input yang lolos (video->video dst). ffmpeg yang "berubah
	// pikiran" / file yang ditukar di tengah jalan langsung digagalkan.
	outMime, _ := guard.SniffFile(outPath)
	report.Details["out_mime"] = outMime
	if guard.TopType(outMime) != guard.TopType(gres.Mime) {
		os.Remove(outPath)
		fail("output format changed after compress (in "+gres.Mime+", out "+outMime+")", 1)
	}
	// Sidik output: simpan di DB; verify ulang kapan saja via `verify`.
	if sum, err := fileSHA256(outPath); err != nil {
		os.Remove(outPath)
		fail("cannot fingerprint output: "+err.Error(), 1)
	} else {
		report.SHA256 = sum
	}

	report.Status = "clean"
	report.TookMs = time.Since(start).Milliseconds()
	saveReport()
	if releaseSlot != nil {
		releaseSlot()
	}
	b, _ := json.Marshal(report)
	fmt.Println(string(b))
}

// quarantine: salin file yang DIBLOKIR ke dir forensik + laporannya.
// Default MATI ("") = langsung buang (aman). Nyalakan hanya bila butuh audit:
// --config {"quarantine_dir": "/var/lib/gc-quarantine"} (root-only, 0700!).
func quarantine(inPath string, report *Report, cfg map[string]any) {
	qd, _ := cfg["quarantine_dir"].(string)
	if qd == "" {
		return
	}
	if err := os.MkdirAll(qd, 0o700); err != nil {
		return
	}
	stem := guard.Sanitize(filepath.Base(inPath)) + "-q-" + strconv.FormatInt(time.Now().UnixNano(), 10)
	dst := filepath.Join(qd, stem)
	_ = copyFileLocal(inPath, dst)
	b, _ := json.MarshalIndent(report, "", "  ")
	_ = os.WriteFile(dst+".report.json", b, 0o600)
}

func copyFileLocal(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}

// runVerify: audit file SIMPANAN kapan saja (cron/queue berkala).
// Mendeteksi perubahan SETELAH lolos: hash beda dari sidik saat ingest,
// atau pola jahat baru (rules query) yang dulu belum dikenal.
func runVerify() {
	fs := flag.NewFlagSet("verify", flag.ExitOnError)
	inPath := fs.String("in", "", "stored file path to audit")
	configStr := fs.String("config", "{}", "JSON config")
	expectSHA := fs.String("expect-sha256", "", "fingerprint dari report ingest")
	_ = fs.Bool("json", false, "output report as JSON")
	_ = fs.Parse(os.Args[2:])

	start := time.Now()
	report := Report{InPath: *inPath, Details: map[string]any{}}
	var releaseSlot func()
	done := func(status, reason string, code int) {
		if releaseSlot != nil {
			releaseSlot()
		}
		report.Status = status
		report.Reason = reason
		report.TookMs = time.Since(start).Milliseconds()
		b, _ := json.Marshal(report)
		fmt.Println(string(b))
		os.Exit(code)
	}
	if *inPath == "" {
		done("error", "missing --in", 1)
	}
	var cfg map[string]any
	if err := json.Unmarshal([]byte(*configStr), &cfg); err != nil {
		done("error", "invalid --config JSON: "+err.Error(), 1)
	}
	// Verify ikut antre slot yang sama (audit massal cron tak boleh menumbangkan server).
	maxSlots := runtime.NumCPU()
	if v, ok := cfg["max_slots"]; ok {
		switch n := v.(type) {
		case float64:
			maxSlots = int(n)
		case int:
			maxSlots = n
		}
	}
	rel, err := slots.Acquire(filepath.Join(compress.CacheDir(), "slots"), maxSlots)
	if err != nil {
		report.Details["busy"] = true
		done("error", "server busy, retry later", 1)
	}
	releaseSlot = rel
	// 1. Sidik sekarang vs sidik ingest -> ketahuan bila file diganti/diubah.
	if *expectSHA != "" {
		sum, err := fileSHA256(*inPath)
		if err != nil {
			done("error", "cannot read stored file: "+err.Error(), 1)
		}
		report.SHA256 = sum
		if sum != *expectSHA {
			report.Details["expected_sha256"] = *expectSHA
			done("blocked", "stored file CHANGED since ingest (hash mismatch)", 2)
		}
	}
	// 2. Scan ulang dengan rules saat ini (tangkap pola baru).
	gres, err := guard.Scan(*inPath, cfg)
	if err != nil {
		done("error", "guard error: "+err.Error(), 1)
	}
	report.Detected = gres.Mime
	report.OrigBytes = gres.Size
	report.Details["guard"] = gres.Details
	if !gres.Allowed {
		done("blocked", "re-scan blocked: "+gres.Reason, 2)
	}
	done("clean", "", 0)
}

func runDoctor() {
	ff := compress.FindFFmpeg()
	nCPU := runtime.NumCPU()
	rec := nCPU / 2
	if rec < 1 {
		rec = 1
	}
	info := map[string]any{
		"version":                Version,
		"ffmpeg":                 ff,
		"ffmpeg_found":           ff != "",
		"cache_dir":              compress.CacheDir(),
		"cpu_count":              nCPU,
		"recommended_jobs":       rec, // ANTI-DOWN: jobs paralel + worker queue jangan lewat ini
		"ffmpeg_threads_default": 2,
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

// samePathFile: true bila outPath menunjuk file yang sama dengan inPath
// (banding path absolut + SameFile bila target sudah ada).
func samePathFile(inPath, outPath string) (bool, error) {
	ai, err := filepath.Abs(inPath)
	if err != nil {
		return false, err
	}
	ao, err := filepath.Abs(outPath)
	if err != nil {
		return false, err
	}
	if ai == ao {
		return true, nil
	}
	fi, err1 := os.Stat(ai)
	fo, err2 := os.Stat(ao)
	if err1 == nil && err2 == nil && os.SameFile(fi, fo) {
		return true, nil
	}
	return false, nil
}
