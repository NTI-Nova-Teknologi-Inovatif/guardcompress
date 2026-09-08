package guard

import (
	"bytes"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

type Result struct {
	Mime    string
	Size    int64
	ModNano int64 // mtime untuk cek ulang TOCTOU di main
	Allowed bool
	Reason  string
	Details map[string]any
}

var extByMime = map[string]string{
	"video/mp4": ".mp4", "video/webm": ".webm", "video/x-matroska": ".mkv",
	"video/avi":  ".avi",
	"audio/mpeg": ".mp3", "audio/wav": ".wav", "audio/wave": ".wav",
	"audio/ogg": ".ogg", "application/ogg": ".ogg",
	"audio/mp4": ".m4a", "audio/flac": ".flac",
	"image/jpeg": ".jpg", "image/png": ".png", "image/webp": ".webp",
	"image/gif": ".gif",
}

var defaultAllow = []string{
	"video/mp4", "video/webm",
	"audio/mpeg", "audio/wav", "audio/wave", "audio/ogg", "application/ogg",
	"image/jpeg", "image/png", "image/webp", "image/gif",
}

var outExtByMime = map[string]string{
	"audio/wav": ".mp3", "audio/wave": ".mp3", "audio/flac": ".mp3",
}

func OutExt(mime string) string {
	if e, ok := outExtByMime[mime]; ok {
		return e
	}
	return extFor(mime)
}

var extToMime = map[string][]string{
	"jpg": {"image/jpeg"}, "jpeg": {"image/jpeg"},
	"png": {"image/png"}, "webp": {"image/webp"}, "gif": {"image/gif"},
	"mp4": {"video/mp4"}, "mov": {"video/mp4"}, // mov terdeteksi ftyp -> mp4
	"webm": {"video/webm"}, "mkv": {"video/webm"}, // mkv terdeteksi EBML -> webm
	"avi": {"video/avi"},
	"mp3": {"audio/mpeg"},
	"wav": {"audio/wave", "audio/wav"},
	"ogg": {"application/ogg", "audio/ogg"},
	"oga": {"application/ogg", "audio/ogg"},
	"m4a": {"audio/mp4"}, "flac": {"audio/flac"},
}

var mimeAlias = map[string]string{
	"audio/wav": "audio/wave", "audio/wave": "audio/wav",
	"audio/ogg": "application/ogg", "application/ogg": "audio/ogg",
}

func mimeAllowed(mime string, allow []string) bool {
	for _, a := range allow {
		if mime == a {
			return true
		}
		if al, ok := mimeAlias[mime]; ok && al == a {
			return true
		}
		if al, ok := mimeAlias[a]; ok && al == mime {
			return true
		}
	}
	return false
}

func resolveAllow(cfg map[string]any) ([]string, error) {
	var hasAllow bool
	var allow []string
	if raw, ok := cfg["allow"].([]any); ok && len(raw) > 0 {
		hasAllow = true
		for _, a := range raw {
			if s, ok := a.(string); ok {
				allow = append(allow, s)
			}
		}
	} else if raw2, ok := cfg["allow"].([]string); ok && len(raw2) > 0 {
		hasAllow = true
		allow = raw2
	}
	var hasExt bool
	var mapped []string
	if raw, ok := cfg["allow_ext"].([]any); ok && len(raw) > 0 {
		hasExt = true
		for _, e := range raw {
			s, ok := e.(string)
			if !ok {
				continue
			}
			m, ok := extToMime[strings.ToLower(strings.TrimPrefix(s, "."))]
			if !ok {
				return nil, fmt.Errorf("unknown allow_ext %q (valid: jpg jpeg png webp gif mp4 mov webm mkv avi mp3 wav ogg oga m4a flac)", s)
			}
			mapped = append(mapped, m...)
		}
	}
	switch {
	case hasAllow && hasExt:
		return append(allow, mapped...), nil
	case hasAllow:
		return allow, nil
	case hasExt:
		return mapped, nil
	default:
		return defaultAllow, nil
	}
}

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
	return ".bin"
}

func Scan(path string, cfg map[string]any) (Result, error) {
	res := Result{Details: map[string]any{}}
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
	res.ModNano = fi.ModTime().UnixNano()

	if runtime.GOOS == "windows" {
		rest := path
		if len(rest) > 2 && rest[1] == ':' {
			rest = rest[2:]
		}
		if strings.Contains(rest, ":") {
			res.Reason = "alternate data stream not allowed"
			return res, nil
		}
	}

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

	mime, sniffDetails := SniffFile(path)
	for k, v := range sniffDetails {
		res.Details[k] = v
	}
	res.Mime = mime
	res.Details["sniffed"] = mime

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

	if reason, token := streamScan(f, res.Size); reason != "" {
		res.Reason = reason
		if token != "" {
			res.Details["token"] = token
		}
		return res, nil
	}

	if v, ok := cfg["block_embedded_containers"].(bool); !ok || v {
		if found := scanContainers(f, res.Size); found != "" {
			res.Reason = found
			res.Details["container"] = found
			return res, nil
		}
	}

	if sig, used, err := scanClamAV(path, cfg); err != nil {
		return res, err
	} else if used {
		res.Details["clamav"] = sig
		if sig != "" {
			res.Reason = "clamav detected: " + sig
			return res, nil
		}
	}

	allow, err := resolveAllow(cfg)
	if err != nil {
		return res, err
	}
	if !mimeAllowed(res.Mime, allow) {
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

func SniffFile(path string) (string, map[string]any) {
	details := map[string]any{}
	f, err := os.Open(path)
	if err != nil {
		return "unknown", details
	}
	defer f.Close()
	head := make([]byte, 8192)
	n, _ := f.Read(head)
	head = head[:n]
	if len(head) == 0 {
		return "unknown", details
	}
	mime := http.DetectContentType(head[:min(n, 512)])
	if mime == "application/octet-stream" && len(head) > 12 && string(head[4:8]) == "ftyp" {
		brand := string(head[8:12])
		switch brand {
		case "M4A ", "M4B ":
			mime = "audio/mp4"
		case "isom", "iso2", "iso3", "iso4", "iso5", "iso6",
			"mp41", "mp42", "M4V ", "M4P ", "MSNV", "avc1", "qt  ":
			mime = "video/mp4"
			details["ftyp_fix"] = true
		}
	}
	if mime == "application/octet-stream" && len(head) > 4 && string(head[:4]) == "fLaC" {
		mime = "audio/flac"
	}
	if mime == "application/octet-stream" && len(head) > 4 &&
		head[0] == 0x1A && head[1] == 0x45 && head[2] == 0xDF && head[3] == 0xA3 {
		mime = "video/webm"
	}
	return mime, details
}

func TopType(mime string) string {
	if i := strings.Index(mime, "/"); i > 0 {
		return mime[:i]
	}
	return mime
}

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
	if bytes.IndexByte(tok, 0) >= 0 {
		return true
	}
	return len(tok) >= 4
}

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
