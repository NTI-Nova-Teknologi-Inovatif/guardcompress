package guard

import (
	"crypto/rand"
	"encoding/hex"
	"path/filepath"
	"strings"
	"unicode"
)

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
	if isWindowsReserved(stem) {
		stem = "file_" + stem
	}
	return stem + OutExt(mime)
}

var windowsReserved = map[string]bool{
	"con": true, "prn": true, "aux": true, "nul": true,
	"com1": true, "com2": true, "com3": true, "com4": true, "com5": true,
	"com6": true, "com7": true, "com8": true, "com9": true,
	"lpt1": true, "lpt2": true, "lpt3": true, "lpt4": true, "lpt5": true,
	"lpt6": true, "lpt7": true, "lpt8": true, "lpt9": true,
}

func isWindowsReserved(stem string) bool {
	return windowsReserved[strings.ToLower(stem)]
}

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
