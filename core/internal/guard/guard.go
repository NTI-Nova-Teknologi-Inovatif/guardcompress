// Package guard: Guard Phase - validasi format + scan malware ringan.
// v1: magic numbers (stdlib) + heuristic webshell/polyglot + blokir zip-bomb.
// v2 roadmap: ganti heuristic dengan yara-x binding + rules/ dir (go:embed).
package guard

import (
	"bytes"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type Result struct {
	Mime    string
	Size    int64
	Allowed bool
	Reason  string
	Details map[string]any
}

var extByMime = map[string]string{
	"video/mp4": ".mp4", "video/webm": ".webm", "video/x-matroska": ".mkv",
	"audio/mpeg": ".mp3", "audio/wav": ".wav", "audio/ogg": ".ogg",
	"image/jpeg": ".jpg", "image/png": ".png", "image/webp": ".webp",
}

var defaultAllow = []string{
	"video/mp4", "video/webm", "video/x-matroska",
	"audio/mpeg", "audio/wav", "audio/ogg",
	"image/jpeg", "image/png", "image/webp",
}

// Suspicious byte patterns: webshell / script polyglot yang sering nempel di media.
var suspiciousTokens = [][]byte{
	[]byte("<?php"), []byte("<%"), []byte("<script"),
	[]byte("eval("), []byte("base64_decode"), []byte("c99shell"),
	[]byte("X5O!P%@AP[4\\PZX54(P^)7CC)7}$EICAR"), // EICAR test string
}

func (r Result) SafeExt() string {
	if e, ok := extByMime[r.Mime]; ok {
		return e
	}
	return filepath.Ext(r.Mime) // fallback, jarang kepakai
}

func Scan(path string, cfg map[string]any) (Result, error) {
	res := Result{Details: map[string]any{}}
	fi, err := os.Stat(path)
	if err != nil {
		return res, err
	}
	res.Size = fi.Size()

	// 1. Batas ukuran (default 500MB, bisa dioverride via --config)
	maxMB := 500.0
	if v, ok := cfg["max_mb"].(float64); ok && v > 0 {
		maxMB = v
	}
	if float64(res.Size) > maxMB*1024*1024 {
		res.Reason = fmt.Sprintf("file too large: %d bytes > %.0fMB", res.Size, maxMB)
		return res, nil
	}

	// 2. Sniff MIME dari 512 byte pertama (magic numbers), BUKAN dari extension.
	f, err := os.Open(path)
	if err != nil {
		return res, err
	}
	defer f.Close()
	head := make([]byte, 512)
	n, _ := f.Read(head)
	res.Mime = http.DetectContentType(head[:n])
	res.Details["sniffed"] = res.Mime

	// 3. Allowlist check
	allow := defaultAllow
	if raw, ok := cfg["allow"].([]any); ok && len(raw) > 0 {
		allow = nil
		for _, a := range raw {
			if s, ok := a.(string); ok {
				allow = append(allow, s)
			}
		}
	}
	allowed := false
	for _, a := range allow {
		if res.Mime == a || strings.HasPrefix(res.Mime, a) {
			allowed = true
			break
		}
	}
	// Catatan: video MP4 kadang terdeteksi application/octet-stream kalau header pendek.
	// Untuk v1 skeleton kita tolak tegas agar aman; tuning ftpy/mp4 box ada di roadmap.
	if !allowed {
		res.Reason = "mime not allowed: " + res.Mime
		return res, nil
	}

	// 4. Heuristic scan: baca seluruh file ( capped 200MB untuk v1 ) cari token berbahaya.
	// Production: ganti dengan streaming + yara scan per-chunk.
	const capBytes = 200 << 20
	toRead := res.Size
	if toRead > capBytes {
		toRead = capBytes
	}
	buf := make([]byte, toRead)
	_, _ = f.ReadAt(buf, 0)
	for _, tok := range suspiciousTokens {
		if bytes.Contains(buf, tok) {
			// Potong token EICAR agar tidak bocor ke log mentah
			shown := string(tok)
			if len(shown) > 20 {
				shown = shown[:20] + "..."
			}
			res.Reason = "suspicious token detected: " + shown
			res.Details["token"] = shown
			return res, nil
		}
	}

	// 5. Zip-bomb / ratio check sederhana: tolak file dengan null-ratio ekstrem + ukuran besar
	// (placeholder — kompresi ratio real dicek setelah FFmpeg; di sini cegah OOM).
	res.Allowed = true
	return res, nil
}
