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
// fold=true untuk pola yang case-insensitive di engine aslinya
// (fungsi PHP & tag HTML tidak peduli huruf besar/kecil).
// Varian UTF-16-LE/BE dari "<?php" ikut dipindai (webshell unicode).
type token struct {
	pat  []byte
	fold bool
}

var suspiciousTokens = []token{
	{[]byte("<?php"), false}, // tag PHP wajib lowercase di engine
	{[]byte("<?="), false},
	{[]byte("<%"), false},
	{[]byte("<script"), true}, // HTML case-insensitive
	{[]byte("eval("), true},   // konstruksi PHP case-insensitive
	{[]byte("assert("), true},
	{[]byte("base64_decode"), true}, // fungsi PHP case-insensitive
	{[]byte("str_rot13"), true},
	{[]byte("gzinflate"), true},
	{[]byte("create_function"), true},
	{[]byte("shell_exec"), true},
	{[]byte("passthru"), true},
	{[]byte("popen("), true},
	{[]byte("proc_open("), true},
	{[]byte("c99shell"), false},
	{[]byte("r57shell"), false},
	{[]byte("cmd.exe"), true},
	{[]byte("/bin/sh"), false},
	{[]byte{'<', 0, '?', 0, 'p', 0, 'h', 0, 'p', 0}, false}, // "<?php" UTF-16LE
	{[]byte{0, '<', 0, '?', 0, 'p', 0, 'h', 0, 'p'}, false}, // "<?php" UTF-16BE
	{[]byte("X5O!P%@AP[4\\PZX54(P^)7CC)7}$EICAR"), false},
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
	// AUDIT: tolak symlink agar pemindaian tidak bisa diarahkan baca file
	// sembarang milik server (defense-in-depth; path normalnya tmp acak).
	if li, err := os.Lstat(path); err != nil {
		return res, err
	} else if li.Mode()&os.ModeSymlink != 0 {
		res.Reason = "symlink not allowed"
		res.Details["filename"] = filepath.Base(path)
		return res, nil
	}
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

	// Peringatan double-extension: evil.mp4.php (nama file tmp normalnya acak
	// tanpa extension, tapi tetap tolak pola executable sebagai jaring kedua).
	lower := strings.ToLower(filepath.Base(path))
	for _, bad := range []string{".php", ".phtml", ".phar", ".asp", ".aspx",
		".jsp", ".jspx", ".cgi", ".pl", ".py", ".sh", ".exe", ".com", ".bat",
		".ps1", ".htaccess"} {
		if strings.Contains(lower, bad) {
			res.Reason = "suspicious filename (executable extension): " + lower
			res.Details["filename"] = lower
			return res, nil
		}
	}

	// Heuristic scan DULU (sebelum allowlist) agar alasan blokir presisi.
	// AUDIT: streaming per-chunk 1MB + overlap (bukan head+tail) agar payload
	// yang disembunyikan di TENGAH file besar tetap ketemu, memory konstan.
	if reason, token := streamScan(f, res.Size); reason != "" {
		res.Reason = reason
		if token != "" {
			res.Details["token"] = token
		}
		return res, nil
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

// streamScan: pindai seluruh file per-chunk 1MB dengan overlap 4KB.
// Overlap besar agar laju teks yang terpotong batas chunk tetap utuh
// terlihat oleh inTextRun dari kedua sisi.
func streamScan(f *os.File, size int64) (string, string) {
	const chunkSize = 1 << 20
	const overlap = 4096
	var prev []byte
	for offset := int64(0); offset < size; {
		toRead := int64(chunkSize)
		if offset+toRead > size {
			toRead = size - offset
		}
		chunk := make([]byte, toRead)
		n, err := f.ReadAt(chunk, offset)
		if n > 0 {
			chunk = chunk[:n]
		} else {
			break
		}
		window := make([]byte, 0, len(prev)+len(chunk))
		window = append(window, prev...)
		window = append(window, chunk...)
		// Varian lowercase sekali per window untuk token fold (ASCII-only
		// agar offset tetap 1:1 dengan buffer asli).
		lowered := asciiLower(window)
		for _, tok := range suspiciousTokens {
			hay := window
			if tok.fold {
				hay = lowered
			}
			if foundAt(hay, window, tok.pat) {
				shown := string(tok.pat)
				if len(shown) > 24 {
					shown = shown[:24] + "..."
				}
				return "suspicious token detected: " + shown, shown
			}
		}
		if len(window) > overlap {
			prev = append(prev[:0], window[len(window)-overlap:]...)
		} else {
			prev = append(prev[:0], window...)
		}
		offset += int64(n)
		if err != nil {
			break
		}
	}
	return "", ""
}

// foundAt: cari pat di hay (hay boleh versi lowercase dari orig).
// Token pendek hanya dihitung bila duduk di laju teks printable >=24
// (dicek pada ORIG agar tidak terpengaruh lowering).
// Token panjang & unik (EICAR, UTF-16) cocok langsung tanpa cek konteks.
func foundAt(hay, orig, pat []byte) bool {
	if isDirectToken(pat) {
		return bytes.Contains(hay, pat)
	}
	start := 0
	for {
		i := bytes.Index(hay[start:], pat)
		if i < 0 {
			return false
		}
		at := start + i
		if inTextRun(orig, at, len(pat)) {
			return true
		}
		start = at + 1
		if start >= len(hay) {
			return false
		}
	}
}

func isDirectToken(tok []byte) bool {
	if bytes.HasPrefix(tok, []byte("X5O!P%@AP")) {
		return true
	}
	// UTF-16 variants mengandung NUL -> tak pernah lolos inTextRun, direct saja.
	return bytes.IndexByte(tok, 0) >= 0
}

// inTextRun: panjang laju printable maksimal yang memuat token >= 24?
func inTextRun(buf []byte, at, tokLen int) bool {
	const minRun = 24
	l := at
	for l > 0 && isPrintable(buf[l-1]) {
		l--
	}
	r := at + tokLen
	for r < len(buf) && isPrintable(buf[r]) {
		r++
	}
	return r-l >= minRun
}

func isPrintable(b byte) bool {
	return b == 9 || b == 10 || b == 13 || (b >= 32 && b < 127)
}

// asciiLower: lowercase ASCII-only (A-Z -> a-z), byte lain utuh.
// Panjang & offset dijamin 1:1 dengan input (aman untuk pemetaan temuan).
func asciiLower(b []byte) []byte {
	out := make([]byte, len(b))
	for i, c := range b {
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		out[i] = c
	}
	return out
}
