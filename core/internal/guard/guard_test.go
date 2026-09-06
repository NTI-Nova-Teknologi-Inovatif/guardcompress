package guard

import (
	"os"
	"path/filepath"
	"testing"
)

func writeTmp(t *testing.T, name string, data []byte) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(p, data, 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestBlockWebshellPolyglot(t *testing.T) {
	// PNG header valid + payload PHP nempel
	png := []byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A}
	png = append(png, []byte("...<?php eval(base64_decode('x'));")...)
	p := writeTmp(t, "evil.png", png)
	r, err := Scan(p, map[string]any{})
	if err != nil {
		t.Fatal(err)
	}
	if r.Allowed {
		t.Fatal("harusnya BLOCKED karena token php")
	}
}

func TestBlockDoubleExtension(t *testing.T) {
	png := []byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A, 0, 0}
	p := writeTmp(t, "foto.png.php", png)
	r, _ := Scan(p, map[string]any{})
	if r.Allowed {
		t.Fatal("harusnya BLOCKED karena double extension .php")
	}
}

func TestAllowCleanPNG(t *testing.T) {
	png := []byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A}
	png = append(png, make([]byte, 100)...)
	p := writeTmp(t, "clean.png", png)
	r, err := Scan(p, map[string]any{})
	if err != nil {
		t.Fatal(err)
	}
	if !r.Allowed || r.Mime != "image/png" {
		t.Fatalf("harusnya clean png, dapat %+v", r)
	}
}

func TestDetectFTYPasMP4(t *testing.T) {
	// Minimal ftyp box: ....ftypisom
	head := []byte{0, 0, 0, 24, 'f', 't', 'y', 'p', 'i', 's', 'o', 'm', 0, 0, 0, 0}
	head = append(head, make([]byte, 100)...)
	p := writeTmp(t, "clip.mp4", head)
	r, _ := Scan(p, map[string]any{})
	if r.Mime != "video/mp4" {
		t.Fatalf("harusnya video/mp4, dapat %s (%s)", r.Mime, r.Reason)
	}
	if !r.Allowed {
		t.Fatalf("harusnya allowed, reason: %s", r.Reason)
	}
}

func TestRejectTooLarge(t *testing.T) {
	p := writeTmp(t, "big.mp4", []byte{0, 0, 0, 24, 'f', 't', 'y', 'p'})
	r, _ := Scan(p, map[string]any{"max_mb": float64(0.000001)})
	if r.Allowed {
		t.Fatal("harusnya BLOCKED karena over max_mb")
	}
}
