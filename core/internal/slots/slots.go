// Package slots: semafor file lintas-proses (tanpa daemon, tanpa lock server).
// Melindungi server dari N proses guardcompress yang jalan bareng
// (100 user upload bersamaan): slot penuh -> tolak cepat "busy" (HTTP 429),
// bukan terima semua lalu server tumbang.
//
// Aman dipakai lintas bahasa/OS karena hanya butuh file + O_EXCL.
// Batas lunak (race sempit bisa lolos 1-2 slot) — cukup untuk admission,
// bukan batas keamanan keras.
package slots

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ErrBusy dikembalikan bila slot penuh.
var ErrBusy = errors.New("server busy, retry later")

// StaleAfter: lock lebih tua dari ini dianggap yatim (proses crash) dan dibersihkan.
const StaleAfter = 15 * time.Minute

// Acquire merebut 1 slot di dir (dibuat 0700 bila belum ada).
// max <= 0 artinya tanpa batas. Return fungsi release (wajib dipanggil!).
func Acquire(dir string, max int) (func(), error) {
	if max <= 0 {
		return func() {}, nil
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	cleanStale(dir)
	mine, err := claim(dir)
	if err != nil {
		return nil, err
	}
	// Hitung ulang SETELAH klaim: bila ternyata over kuota (race),
	// lepaskan dan mundur (backpressure konvergen).
	if countFresh(dir) > max {
		os.Remove(mine)
		return nil, ErrBusy
	}
	return func() { os.Remove(mine) }, nil
}

func slotName() string {
	var b [6]byte
	_, _ = rand.Read(b[:])
	return "slot-" + hex.EncodeToString(b[:])
}

func claim(dir string) (string, error) {
	for range 10 {
		p := filepath.Join(dir, slotName())
		f, err := os.OpenFile(p, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
		if err == nil {
			_, _ = f.WriteString("pid")
			f.Close()
			return p, nil
		}
	}
	return "", errors.New("cannot claim slot")
}

func countFresh(dir string) int {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0
	}
	n := 0
	for _, e := range entries {
		if !strings.HasPrefix(e.Name(), "slot-") {
			continue
		}
		n++
	}
	return n
}

func cleanStale(dir string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	now := time.Now()
	for _, e := range entries {
		if !strings.HasPrefix(e.Name(), "slot-") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		if now.Sub(info.ModTime()) > StaleAfter {
			os.Remove(filepath.Join(dir, e.Name()))
		}
	}
}
