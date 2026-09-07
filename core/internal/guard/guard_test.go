package guard

import (
	"os"
	"path/filepath"
	"runtime"
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

func TestBinaryCoincidenceAllowed(t *testing.T) {
	// Regression: byte "<%" yang muncul kebetulan di data biner (non-teks)
	// JANGAN diblokir. Konteks harus >70% printable agar dihitung webshell.
	png := []byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A}
	noise := make([]byte, 300)
	for i := range noise {
		noise[i] = byte(i * 7 % 256) // acak non-printable
	}
	blob := append(png, noise...)
	blob = append(blob, '<', '%')
	blob = append(blob, noise...)
	p := writeTmp(t, "binary.png", blob)
	r, _ := Scan(p, map[string]any{})
	if !r.Allowed {
		t.Fatalf("harusnya allowed (konteks biner), reason: %s", r.Reason)
	}
}

func TestEicarAlwaysBlocked(t *testing.T) {
	// EICAR diblokir bahkan di tengah data biner (tanpa konteks teks).
	png := []byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A}
	blob := append(png, 0x00, 0xFF, 0x01)
	blob = append(blob, []byte(`X5O!P%@AP[4\PZX54(P^)7CC)7}$EICAR-STANDARD-ANTIVIRUS-TEST-FILE!$H+H*`)...)
	blob = append(blob, 0x00, 0xFE)
	p := writeTmp(t, "eicar.png", blob)
	r, _ := Scan(p, map[string]any{})
	if r.Allowed {
		t.Fatal("harusnya BLOCKED (EICAR)")
	}
}

func TestSymlinkRejected(t *testing.T) {
	target := writeTmp(t, "real.png", []byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A, 0, 0})
	link := filepath.Join(t.TempDir(), "link.png")
	if err := os.Symlink(target, link); err != nil {
		t.Skip("symlink tak didukung di sini: " + err.Error())
	}
	r, err := Scan(link, map[string]any{})
	if err != nil {
		t.Fatal(err)
	}
	if r.Allowed {
		t.Fatal("harusnya BLOCKED (symlink)")
	}
}

func TestUppercaseVariantsBlocked(t *testing.T) {
	for _, payload := range []string{
		"EVAL(BASE64_DECODE(\"eA==\"));",
		"<ScRiPt>alert(1)</ScRiPt>",
		"<?php SHELL_EXEC(\"id\"); ?>",
		"<?php $h = popen(\"ls\",\"r\"); ?>",
	} {
		png := []byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A}
		p := writeTmp(t, "upper.png", append(png, payload...))
		r, _ := Scan(p, map[string]any{})
		if r.Allowed {
			t.Fatalf("harusnya BLOCKED: %s", payload)
		}
	}
}

func TestPayloadInMiddleOfLargeFile(t *testing.T) {
	// Regression blind-spot: payload di tengah file 40MB HARUS ketemu
	// (streaming scan, bukan head+tail).
	const size = 40 << 20
	blob := make([]byte, size)
	for i := range blob {
		blob[i] = byte(i * 13 % 251)
	}
	copy(blob[:8], []byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A})
	payload := []byte("......<?php system($_GET['x']); ?>......")
	copy(blob[20<<20:], payload)
	p := writeTmp(t, "big.png", blob)
	r, err := Scan(p, map[string]any{})
	if err != nil {
		t.Fatal(err)
	}
	if r.Allowed {
		t.Fatal("harusnya BLOCKED (payload di tengah file besar lolos!)")
	}
}

func TestADSRejected(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("ADS hanya ada di Windows")
	}
	base := writeTmp(t, "host.png", []byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A, 0, 0})
	ads := base + ":evil"
	if err := os.WriteFile(ads, []byte("x"), 0o644); err != nil {
		t.Skip("ADS tak didukung di volume ini: " + err.Error())
	}
	r, err := Scan(ads, map[string]any{})
	if err != nil {
		t.Fatal(err)
	}
	if r.Allowed {
		t.Fatal("harusnya BLOCKED (ADS)")
	}
}

func TestSniffFile(t *testing.T) {
	png := writeTmp(t, "a.png", append([]byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A}, 0, 0))
	if mime, _ := SniffFile(png); mime != "image/png" {
		t.Fatalf("dapat %q", mime)
	}
	mp4 := writeTmp(t, "a.mp4", append([]byte{0, 0, 0, 24, 'f', 't', 'y', 'p', 'i', 's', 'o', 'm'}, 0))
	if mime, d := SniffFile(mp4); mime != "video/mp4" {
		t.Fatalf("dapat %q (%v)", mime, d)
	}
	if mime, _ := SniffFile(t.TempDir() + "/tak-ada"); mime != "unknown" {
		t.Fatalf("file hilang harus unknown, dapat %q", mime)
	}
}

func TestTopType(t *testing.T) {
	if TopType("video/mp4") != "video" || TopType("image/png") != "image" ||
		TopType("application/octet-stream") != "application" || TopType("unknown") != "unknown" {
		t.Fatal("TopType salah")
	}
}

func TestAllowExt(t *testing.T) {
	png := writeTmp(t, "a.png", append([]byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A}, 0, 0))
	// extension familiar membuka izin tanpa tulis MIME
	r, err := Scan(png, map[string]any{"allow_ext": []any{"jpg", "png"}})
	if err != nil || !r.Allowed {
		t.Fatalf("harusnya allowed via allow_ext, err=%v r=%+v", err, r)
	}
	// allow sempit + allow_ext gabungan (union)
	r, _ = Scan(png, map[string]any{"allow": []any{"video/mp4"}, "allow_ext": []any{"png"}})
	if !r.Allowed {
		t.Fatal("harusnya allowed via union allow+allow_ext")
	}
	// extension ngawur = error developer (bukan blocked)
	_, err = Scan(png, map[string]any{"allow_ext": []any{"exe"}})
	if err == nil {
		t.Fatal("allow_ext ngawur harusnya error")
	}
}

func TestAliasWavOgg(t *testing.T) {
	// ejaan ganda MIME dianggap sama
	wav := writeTmp(t, "a.wav", append([]byte{'R', 'I', 'F', 'F', 0, 0, 0, 0, 'W', 'A', 'V', 'E'}, 0))
	r, _ := Scan(wav, map[string]any{"allow": []any{"audio/wav"}})
	if r.Mime != "audio/wave" {
		t.Fatalf("wav harus terdeteksi audio/wave, dapat %q", r.Mime)
	}
	if !r.Allowed {
		t.Fatalf("audio/wav harus mengizinkan audio/wave: %+v", r)
	}
}

func TestFtypBrandM4A(t *testing.T) {
	// M4A bukan video! brand mayor menentukan keluarga.
	m4a := writeTmp(t, "a.m4a", append([]byte{0, 0, 0, 28, 'f', 't', 'y', 'p', 'M', '4', 'A', ' ', 0}, 0))
	r, _ := Scan(m4a, map[string]any{"allow_ext": []any{"m4a"}})
	if r.Mime != "audio/mp4" {
		t.Fatalf("m4a harus audio/mp4, dapat %q", r.Mime)
	}
	if !r.Allowed {
		t.Fatalf("harusnya allowed: %+v", r)
	}
}

func TestOutExtTranscode(t *testing.T) {
	if OutExt("audio/wave") != ".mp3" || OutExt("audio/flac") != ".mp3" ||
		OutExt("video/mp4") != ".mp4" || OutExt("image/gif") != ".gif" {
		t.Fatal("OutExt salah")
	}
	if got := OutputName("/tmp/rekaman.WAV", "audio/wave", map[string]any{}); got != "rekaman.mp3" {
		t.Fatalf("wav harus jadi mp3: %q", got)
	}
}
