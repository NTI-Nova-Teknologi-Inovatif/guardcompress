package guard

import (
	"archive/zip"
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// zipAsli: arsip ZIP valid (dibuat stdlib) ditempel di ekor PNG.
func TestEmbeddedZipBlocked(t *testing.T) {
	var zb bytes.Buffer
	w := zip.NewWriter(&zb)
	fw, _ := w.Create("hello.txt")
	_, _ = fw.Write([]byte("halo"))
	w.Close()
	png := append([]byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A}, 0, 0)
	p := writeTmp(t, "zip.png", append(png, zb.Bytes()...))
	r, err := Scan(p, map[string]any{})
	if err != nil {
		t.Fatal(err)
	}
	if r.Allowed {
		t.Fatal("harusnya BLOCKED (zip tersembunyi)")
	}
}

// "PK" kebetulan di data acak: JANGAN diblokir (validasi struktur).
func TestRandomPKAllowed(t *testing.T) {
	blob := append([]byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A}, bytes.Repeat([]byte("PK"), 500)...)
	p := writeTmp(t, "pk.png", blob)
	r, _ := Scan(p, map[string]any{})
	if !r.Allowed {
		t.Fatalf("harusnya allowed (PK tanpa struktur): %s", r.Reason)
	}
}

// PE minimal tervalidasi (MZ + e_lfanew + PE sig).
func TestEmbeddedPEBlocked(t *testing.T) {
	pe := make([]byte, 0x60)
	pe[0], pe[1] = 'M', 'Z'
	pe[0x3C] = 0x40 // e_lfanew
	copy(pe[0x40:], []byte{'P', 'E', 0, 0})
	png := append([]byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A}, 0, 0)
	p := writeTmp(t, "pe.png", append(png, pe...))
	r, _ := Scan(p, map[string]any{})
	if r.Allowed {
		t.Fatal("harusnya BLOCKED (PE tersembunyi)")
	}
}

// "MZ" tanpa struktur PE: JANGAN diblokir.
func TestRandomMZAllowed(t *testing.T) {
	blob := append([]byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A}, bytes.Repeat([]byte("MZ"), 500)...)
	p := writeTmp(t, "mz.png", blob)
	r, _ := Scan(p, map[string]any{})
	if !r.Allowed {
		t.Fatalf("harusnya allowed (MZ tanpa struktur): %s", r.Reason)
	}
}

func TestRAR7zELFBlocked(t *testing.T) {
	png := []byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A, 0, 0}
	for name, magic := range map[string][]byte{
		"rar": {'R', 'a', 'r', '!', 0x1a, 0x07, 0x00, 0, 0, 0},
		"7z":  {'7', 'z', 0xbc, 0xaf, 0x27, 0x1c, 0, 0},
		"elf": {0x7f, 'E', 'L', 'F', 1, 1, 0, 0},
	} {
		p := writeTmp(t, name+".png", append(append([]byte{}, png...), magic...))
		r, _ := Scan(p, map[string]any{})
		if r.Allowed {
			t.Fatalf("harusnya BLOCKED (%s tersembunyi)", name)
		}
	}
}

// Opt-out container audit untuk app yang memang butuh (resiko sendiri).
func TestContainerOptOut(t *testing.T) {
	var zb bytes.Buffer
	w := zip.NewWriter(&zb)
	fw, _ := w.Create("a.txt")
	_, _ = fw.Write([]byte("x"))
	w.Close()
	png := append([]byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A}, 0, 0)
	p := writeTmp(t, "zip2.png", append(png, zb.Bytes()...))
	r, _ := Scan(p, map[string]any{"block_embedded_containers": false})
	if !r.Allowed {
		t.Fatalf("opt-out harus lolos: %s", r.Reason)
	}
}

func TestClamAVParser(t *testing.T) {
	sig := parseClamSig("/tmp/x: Eicar-Test-Signature FOUND\n")
	if sig != "Eicar-Test-Signature" {
		t.Fatalf("dapat %q", sig)
	}
}

func TestClamAVSkippedWhenAbsent(t *testing.T) {
	if _, err := os.Stat(filepath.Join(t.TempDir(), "x")); err == nil {
		t.Fatal("unreachable")
	}
	// Tanpa clamav terinstal: auto = diam (used=false, tanpa error).
	sig, used, err := scanClamAV("dummy", map[string]any{})
	if err != nil || used || sig != "" {
		// Bila mesin kebetulan punya clamav, terima hasil apa pun kecuali error aneh.
		if _, lookErr := exec.LookPath("clamdscan"); lookErr != nil {
			if _, lookErr2 := exec.LookPath("clamscan"); lookErr2 != nil {
				t.Fatalf("harusnya skip diam: %v %v %q", used, err, sig)
			}
		}
	}
}
