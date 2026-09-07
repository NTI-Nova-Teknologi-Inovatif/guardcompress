package guard

import (
	"crypto/rand"
	"encoding/hex"
	"path/filepath"
	"strings"
	"unicode"
)

// OutputName: nama file output yang aman & fleksibel.
//
//	mode "original" (default): ikut nama file asli, disanitasi.
//	  "video liburan anak.mp4" -> "video_liburan_anak.mp4"
//	  "evil.mp4.php"           -> "evil_mp4_php.mp4" (ext selalu dari MIME asli!)
//	mode "uuid":   nama acak 16 hex char ( misal "a3f9c1...mp4").
//	mode lain:     dianggap stem kustom, ikut disanitasi.
//
// Extension SELALU dari hasil sniff MIME (SafeExt), bukan dari nama input,
// agar nama tidak bisa dipakai menyelundupkan ".php".
func OutputName(inPath, mime string, cfg map[string]any) string {
	mode, _ := cfg["output"].(string)
	var stem string
	switch mode {
	case "uuid":
		var b [8]byte
		if _, err := rand.Read(b[:]); err == nil {
			stem = hex.EncodeToString(b[:])
			break
		}
		stem = "file"
	case "", "original":
		base := filepath.Base(inPath)
		if i := strings.LastIndex(base, "."); i > 0 {
			base = base[:i]
		}
		stem = Sanitize(base)
	default:
		stem = Sanitize(mode)
	}
	if stem == "" {
		stem = "file"
	}
	return stem + OutExt(mime)
}

// Sanitize: hanya huruf, angka, "-", "_". Selainnya jadi "_".
// Unicode letters dilipat ke "_" agar aman di semua FS & URL.
func Sanitize(s string) string {
	var b strings.Builder
	prevUnder := false
	count := 0
	for _, r := range s {
		var c rune
		switch {
		case r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9':
			c = r
		case r == '-' || r == '_' || unicode.IsSpace(r):
			c = '_'
		default:
			c = '_'
		}
		if c == '_' {
			if prevUnder {
				continue
			}
			prevUnder = true
		} else {
			prevUnder = false
		}
		b.WriteRune(c)
		count++
		if count >= 80 {
			break
		}
	}
	return strings.Trim(b.String(), "_.")
}

func extFor(mime string) string {
	if e, ok := extByMime[mime]; ok {
		return e
	}
	return ".bin"
}
