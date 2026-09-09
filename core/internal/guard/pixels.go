package guard

import (
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"os"
)

const defaultMaxPixels = 100_000_000

func maxPixels(cfg map[string]any) int64 {
	v, ok := cfg["max_pixels"]
	if !ok {
		return defaultMaxPixels
	}
	switch n := v.(type) {
	case int:
		return int64(n)
	case int64:
		return n
	case float64:
		return int64(n)
	}
	return defaultMaxPixels
}

func checkPixels(f *os.File, size int64, mime string, cfg map[string]any) string {
	max := maxPixels(cfg)
	if max <= 0 {
		return ""
	}
	var w, h int64
	var ok bool
	switch mime {
	case "image/jpeg", "image/png", "image/gif":
		c, _, err := image.DecodeConfig(io.NewSectionReader(f, 0, size))
		if err != nil {
			return ""
		}
		w, h, ok = int64(c.Width), int64(c.Height), true
	case "image/webp":
		head := make([]byte, 64)
		n, _ := f.ReadAt(head, 0)
		w, h, ok = webpDims(head[:n])
	}
	if !ok || w <= 0 || h <= 0 {
		return ""
	}
	if w*h > max {
		return fmt.Sprintf("image dimensions too large (%dx%d)", w, h)
	}
	return ""
}

func webpDims(head []byte) (int64, int64, bool) {
	if len(head) < 30 || string(head[:4]) != "RIFF" || string(head[8:12]) != "WEBP" {
		return 0, 0, false
	}
	switch string(head[12:16]) {
	case "VP8X":
		if len(head) < 30 {
			return 0, 0, false
		}
		w := int64(head[24]) | int64(head[25])<<8 | int64(head[26])<<16
		h := int64(head[27]) | int64(head[28])<<8 | int64(head[29])<<16
		return w + 1, h + 1, true
	case "VP8 ":
		if len(head) < 30 {
			return 0, 0, false
		}
		w := int64(head[26]) | int64(head[27]&0x3F)<<8
		h := int64(head[28]) | int64(head[29]&0x3F)<<8
		return w + 1, h + 1, true
	case "VP8L":
		if len(head) < 25 {
			return 0, 0, false
		}
		b := head[21:25]
		w := int64(b[0]) | int64(b[1]&0x3F)<<8
		h := int64(b[1]>>6) | int64(b[2])<<2 | int64(b[3]&0x0F)<<10
		return w + 1, h + 1, true
	}
	return 0, 0, false
}
