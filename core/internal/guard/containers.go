package guard

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
)

var (
	sigZIPLocal   = []byte("PK\x03\x04")
	sigZIPEOCD    = []byte("PK\x05\x06")
	sigZIPCentral = []byte("PK\x01\x02")
	sigRAR        = []byte("Rar!\x1a\x07")
	sig7z         = []byte("7z\xbc\xaf\x27\x1c")
	sigMZ         = []byte("MZ")
	sigELF        = []byte{0x7f, 'E', 'L', 'F'}
)

func findContainer(window []byte, baseOff int64) (string, int64) {
	if at := bytes.Index(window, sigRAR); at >= 0 {
		if at+10 <= len(window) && (window[at+9] == 0 || window[at+9] == 1) {
			return "rar", baseOff + int64(at)
		}
	}
	if at := bytes.Index(window, sig7z); at >= 0 {
		return "7z", baseOff + int64(at)
	}
	if kind, at := validPE(window); kind != "" {
		return kind, baseOff + int64(at)
	}
	if at := bytes.Index(window, sigELF); at >= 0 {
		if at+6 < len(window) && (window[at+4] == 1 || window[at+4] == 2) && window[at+5] == 1 {
			return "elf", baseOff + int64(at)
		}
	}
	return "", -1
}

func collectEOCD(f *os.File, size int64) []int64 {
	const chunkSize = 1 << 20
	var out []int64
	var prev []byte
	for offset := int64(0); offset < size && len(out) < 16; {
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
		window := append(prev, chunk...)
		base := offset - int64(len(prev))
		start := 0
		for len(out) < 16 {
			i := bytes.Index(window[start:], sigZIPEOCD)
			if i < 0 {
				break
			}
			out = append(out, base+int64(start+i))
			start += i + 1
		}
		if len(window) > 22 {
			prev = append(prev[:0], window[len(window)-22:]...)
		} else {
			prev = append(prev[:0], window...)
		}
		offset += int64(n)
		if err != nil {
			break
		}
	}
	return out
}

func validZIPAt(f *os.File, size, eocd int64) bool {
	if eocd < 0 || eocd+22 > size {
		return false
	}
	rec := make([]byte, 22)
	if _, err := f.ReadAt(rec, eocd); err != nil {
		return false
	}
	entries := int(binary.LittleEndian.Uint16(rec[10:12]))
	cdSize := int64(binary.LittleEndian.Uint32(rec[12:16]))
	if entries == 0 || cdSize <= 0 || cdSize > size {
		return false
	}
	back := eocd
	if back > 1<<20 {
		back = 1 << 20
	}
	probe := make([]byte, back)
	if _, err := f.ReadAt(probe, eocd-back); err != nil {
		return false
	}
	ci := bytes.LastIndex(probe, sigZIPCentral)
	if ci < 0 {
		return false
	}
	return bytes.Index(probe[:ci], sigZIPLocal) >= 0
}

func validPE(window []byte) (string, int) {
	start := 0
	for {
		i := bytes.Index(window[start:], sigMZ)
		if i < 0 {
			return "", -1
		}
		at := start + i
		if at+0x40 <= len(window) {
			lfanew := int(binary.LittleEndian.Uint32(window[at+0x3C : at+0x40]))
			if lfanew >= 0 && at+lfanew+6 <= len(window) &&
				window[at+lfanew] == 'P' && window[at+lfanew+1] == 'E' &&
				window[at+lfanew+2] == 0 && window[at+lfanew+3] == 0 {
				return "pe", at
			}
		}
		start = at + 2
		if start >= len(window) {
			return "", -1
		}
	}
}

func scanContainers(f *os.File, size int64) string {
	for _, eocd := range collectEOCD(f, size) {
		if validZIPAt(f, size, eocd) {
			return fmt.Sprintf("embedded zip container at offset %d", eocd)
		}
	}
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
		base := offset - int64(len(prev))
		if kind, at := findContainer(window, base); kind != "" {
			return fmt.Sprintf("embedded %s container at offset %d", kind, at)
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
	return ""
}
