// Package guard: Guard Phase - validasi format + scan malware ringan.
// v1: magic numbers (stdlib) + ftyp MP4 fix + heuristic webshell/polyglot.
// v2 roadmap: yara-x binding + rules/ via go:embed.
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

// Token berbahaya: webshell / script polyglot yang sering ditempel di media.
var suspiciousTokens = [][]byte{
	[]byte("<?php"), []byte("<?="), []byte("<%"), []byte("<script"),
	[]byte("eval("), []byte("base64_decode"), []byte("c99shell"), []byte("r57shell"),
	[]byte("X5O!P%@AP[4\\PZX54(P^)7CC)7}$EICAR"),
}

func (r Result) SafeExt() string {
	if e, ok := extByMime[r.Mime]; ok {
		return e
	}
	// fallback dari nama file asal bila mime tak dikenal (tak dipakai saat blocked)
	return ".bin"
}

func Scan(path string, cfg map[string]any) (Result, error) {
	res := Result{Details: map[string]any{}}
	fi, err := os.Stat(path)
	if err != nil {
		return res, err
	}
	res.Size = fi.Size()

	maxMB := 500.0
	if v, ok := cfg["max_mb"].(float64); ok && v > 0 {
		maxMB = v
	} else if v2, ok := cfg["max_mb"].(int); ok && v2 > 0 {
		maxMB = float64(v2)
	}
	if float64(res.Size) > maxMB*1024*1024 {
		res.Reason = fmt.Sprintf("file too large: %d bytes > %.0fMB", res.Size, maxMB)
		return res, nil
	}
	if res.Size == 0 {
		res.Reason = "empty file"
		return res, nil
	}

	f, err := os.Open(path)
	if err != nil {
		return res, err
	}
	defer f.Close()

	// Baca header lebih besar (8KB) agar deteksi ftyp MP4 akurat.
	head := make([]byte, 8192)
	n, _ := f.Read(head)
	head = head[:n]
	mime := http.DetectContentType(head[:min(n, 512)])

	// Fix: MP4 kadang terdeteksi application/octet-stream oleh stdlib.
	// Cek box 'ftyp' di offset 4.
	if (mime == "application/octet-stream" || mime == "video/mp4") && len(head) > 12 {
		if string(head[4:8]) == "ftyp" {
			mime = "video/mp4"
			res.Details["ftyp_fix"] = true
		}
	}
	// Fix: WebM = EBML header 0x1A45DFA3
	if mime == "application/octet-stream" && len(head) > 4 &&
		head[0] == 0x1A && head[1] == 0x45 && head[2] == 0xDF && head[3] == 0xA3 {
		mime = "video/webm"
	}
	res.Mime = mime
	res.Details["sniffed"] = mime

	// Peringatan double-extension: evil.mp4.php
	lower := strings.ToLower(filepath.Base(path))
	if strings.Contains(lower, ".php") || strings.Contains(lower, ".phtml") ||
		strings.Contains(lower, ".asp") || strings.Contains(lower, ".jsp") {
		res.Reason = "suspicious filename (executable extension): " + lower
		res.Details["filename"] = lower
		return res, nil
	}

	// Heuristic scan DULU (sebelum allowlist) agar alasan blokir presisi.
	// Cap 32MB untuk hemat memory; file besar discan per head+tail.
	const capBytes = 32 << 20
	var buf []byte
	if res.Size <= capBytes {
		buf = make([]byte, res.Size)
		_, _ = f.ReadAt(buf, 0)
	} else {
		buf = make([]byte, capBytes)
		_, _ = f.ReadAt(buf[:16<<20], 0) // 16MB head
		tail := make([]byte, 16<<20)
		_, _ = f.ReadAt(tail, res.Size-int64(len(tail)))
		copy(buf[16<<20:], tail)
		res.Details["scan"] = "head+tail 32MB"
	}
	for _, tok := range suspiciousTokens {
		if foundAt(buf, tok) {
			shown := string(tok)
			if len(shown) > 24 {
				shown = shown[:24] + "..."
			}
			res.Reason = "suspicious token detected: " + shown
			res.Details["token"] = shown
			return res, nil
		}
	}

	allow := defaultAllow
	if raw, ok := cfg["allow"].([]any); ok && len(raw) > 0 {
		allow = nil
		for _, a := range raw {
			if s, ok := a.(string); ok {
				allow = append(allow, s)
			}
		}
	} else if raw2, ok := cfg["allow"].([]string); ok && len(raw2) > 0 {
		allow = raw2
	}
	allowed := false
	for _, a := range allow {
		if res.Mime == a {
			allowed = true
			break
		}
	}
	if !allowed {
		res.Reason = "mime not allowed: " + res.Mime
		return res, nil
	}

	res.Allowed = true
	return res, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// foundAt: cari token di buf, tapi hanya hitung bila konteks sekitarnya
// terlihat seperti teks (kode script itu teks). Data biner video terkompresi
// bisa mengandung "<%" / "eval(" secara kebetulan di antara byte acak.
// EICAR (68 char) cukup unik -> cocok langsung tanpa cek konteks.
func foundAt(buf, tok []byte) bool {
	if bytes.Equal(tok, []byte("X5O!P%@AP[4\\PZX54(P^)7CC)7}$EICAR")) {
		return bytes.Contains(buf, tok)
	}
	start := 0
	for {
		i := bytes.Index(buf[start:], tok)
		if i < 0 {
			return false
		}
		at := start + i
		if isTextWindow(buf, at, len(tok)) {
			return true
		}
		start = at + 1
		if start >= len(buf) {
			return false
		}
	}
}

// isTextWindow: >70% byte di jendela ±64 sekitar temuan harus printable ASCII.
func isTextWindow(buf []byte, at, tokLen int) bool {
	const W = 64
	s := at - W
	if s < 0 {
		s = 0
	}
	e := at + tokLen + W
	if e > len(buf) {
		e = len(buf)
	}
	win := buf[s:e]
	if len(win) == 0 {
		return false
	}
	printable := 0
	for _, b := range win {
		if b == 9 || b == 10 || b == 13 || (b >= 32 && b < 127) {
			printable++
		}
	}
	return float64(printable)/float64(len(win)) > 0.7
}
