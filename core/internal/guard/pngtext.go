package guard

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"io"
	"os"
)

const (
	pngTextMaxChunk  = 16 << 20
	pngTextMaxExpand = 8 << 20
	pngTextMaxChunks = 500
)

func matchTokens(buf []byte) (string, string) {
	lowered := asciiLower(buf)
	for _, tok := range suspiciousTokens {
		hay := buf
		if tok.fold {
			hay = lowered
		}
		if foundAt(hay, buf, tok.pat) {
			shown := string(tok.pat)
			if len(shown) > 24 {
				shown = shown[:24] + "..."
			}
			return "suspicious token detected: " + shown, shown
		}
	}
	return "", ""
}

func scanPNGText(f *os.File, size int64) (string, string) {
	sig := make([]byte, 8)
	if n, err := f.ReadAt(sig, 0); err != nil || n < 8 ||
		!bytes.Equal(sig, []byte("\x89PNG\r\n\x1a\n")) {
		return "", ""
	}
	off := int64(8)
	for n := 0; n < pngTextMaxChunks && off+8 <= size; n++ {
		hdr := make([]byte, 8)
		if _, err := f.ReadAt(hdr, off); err != nil {
			return "", ""
		}
		dlen := int64(binary.BigEndian.Uint32(hdr[:4]))
		typ := string(hdr[4:8])
		if dlen < 0 || dlen > pngTextMaxChunk || off+8+dlen+4 > size {
			return "", ""
		}
		if typ == "zTXt" || typ == "iTXt" {
			data := make([]byte, dlen)
			if _, err := f.ReadAt(data, off+8); err != nil {
				return "", ""
			}
			if raw := pngTextPayload(typ, data); raw != nil {
				if reason, token := matchTokens(raw); reason != "" {
					return reason, token
				}
			}
		}
		off += 8 + dlen + 4
		if typ == "IEND" {
			break
		}
	}
	return "", ""
}

func pngTextPayload(typ string, data []byte) []byte {
	if typ == "zTXt" {
		i := bytes.IndexByte(data, 0)
		if i < 0 || i+2 >= len(data) || data[i+1] != 0 {
			return nil
		}
		return zlibExpand(data[i+2:])
	}
	i := bytes.IndexByte(data, 0)
	if i < 0 || i+3 >= len(data) {
		return nil
	}
	if data[i+1] != 1 || data[i+2] != 0 {
		return nil
	}
	rest := data[i+3:]
	if j := bytes.IndexByte(rest, 0); j < 0 {
		return nil
	} else {
		rest = rest[j+1:]
	}
	if j := bytes.IndexByte(rest, 0); j < 0 {
		return nil
	} else {
		rest = rest[j+1:]
	}
	return zlibExpand(rest)
}

func zlibExpand(comp []byte) []byte {
	zr, err := zlib.NewReader(bytes.NewReader(comp))
	if err != nil {
		return nil
	}
	defer zr.Close()
	raw, err := io.ReadAll(io.LimitReader(zr, pngTextMaxExpand+1))
	if err != nil || len(raw) == 0 || len(raw) > pngTextMaxExpand {
		return nil
	}
	return raw
}
