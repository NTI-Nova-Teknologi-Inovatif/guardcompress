package compress

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// TestVideoCompressReal: bukti kompres beneran, bukan copy.
// Butuh ffmpeg di PATH / GUARDCOMPRESS_FFMPEG. Skip bila absen (misal CI ringan).
func TestVideoCompressReal(t *testing.T) {
	ff := FindFFmpeg()
	if ff == "" {
		t.Skip("ffmpeg tidak ketemu, skip (mode guard-only)")
	}
	tmp := t.TempDir()
	src := filepath.Join(tmp, "src.mp4")
	// Sample 3 detik 1280x720 testsrc + sine audio
	gen := exec.Command(ff, "-y",
		"-f", "lavfi", "-i", "testsrc=duration=3:size=1280x720:rate=30",
		"-f", "lavfi", "-i", "sine=frequency=440:duration=3",
		"-c:v", "libx264", "-pix_fmt", "yuv420p", "-c:a", "aac", src)
	if out, err := gen.CombinedOutput(); err != nil {
		t.Fatalf("gagal bikin sample: %v\n%s", err, out)
	}
	fi, _ := os.Stat(src)
	t.Logf("sample: %d bytes", fi.Size())

	out := filepath.Join(tmp, "out.mp4")
	res, err := Run(src, out, "video/mp4", map[string]any{"video_crf": 32})
	if err != nil {
		t.Fatalf("Run gagal: %v", err)
	}
	if res.Details["mode"] != "ffmpeg" {
		t.Fatalf("harusnya mode=ffmpeg, dapat %v", res.Details["mode"])
	}
	if res.NewBytes == 0 {
		t.Fatal("output 0 byte")
	}
	t.Logf("compressed: %d -> %d bytes", fi.Size(), res.NewBytes)
}

// TestImageCompressReal: JPEG besar di-downscale via ffmpeg.
func TestImageCompressReal(t *testing.T) {
	ff := FindFFmpeg()
	if ff == "" {
		t.Skip("ffmpeg tidak ketemu, skip")
	}
	tmp := t.TempDir()
	src := filepath.Join(tmp, "big.jpg")
	gen := exec.Command(ff, "-y",
		"-f", "lavfi", "-i", "testsrc=duration=1:size=3000x2000:rate=1",
		"-frames:v", "1", src)
	if out, err := gen.CombinedOutput(); err != nil {
		t.Fatalf("gagal bikin sample image: %v\n%s", err, out)
	}
	out := filepath.Join(tmp, "out.jpg")
	res, err := Run(src, out, "image/jpeg", map[string]any{"image_max_dim": 1920})
	if err != nil {
		t.Fatalf("Run gagal: %v", err)
	}
	if res.Details["mode"] != "ffmpeg" {
		t.Fatalf("harusnya mode=ffmpeg, dapat %v", res.Details["mode"])
	}
	t.Logf("image compressed OK: %d bytes", res.NewBytes)
}

// TestGifCompressReal: filter palet 1-pass harus valid (pernah salah sintaks).
func TestGifCompressReal(t *testing.T) {
	if FindFFmpeg() == "" {
		t.Skip("butuh ffmpeg")
	}
	ff := FindFFmpeg()
	tmp := t.TempDir()
	src := filepath.Join(tmp, "a.gif")
	gen := exec.Command(ff, "-y", "-v", "error",
		"-f", "lavfi", "-i", "testsrc=duration=1:size=320x240:rate=10",
		"-frames:v", "10", src)
	if out, err := gen.CombinedOutput(); err != nil {
		t.Fatalf("gagal bikin sample gif: %v\n%s", err, out)
	}
	out := filepath.Join(tmp, "o.gif")
	res, err := Run(src, out, "image/gif", map[string]any{})
	if err != nil {
		t.Fatalf("Run gif gagal: %v", err)
	}
	if res.Details["mode"] != "ffmpeg" {
		t.Fatalf("harusnya mode=ffmpeg, dapat %v", res.Details["mode"])
	}
	// output harus valid gif
	head := make([]byte, 6)
	f, _ := os.Open(out)
	_, _ = f.Read(head)
	f.Close()
	if string(head) != "GIF89a" && string(head) != "GIF87a" {
		t.Fatalf("output bukan gif valid: %q", head)
	}
}

func TestInvalidConfigRejected(t *testing.T) {
	// AUDIT: nilai config liar harus ditolak tegas sebelum ffmpeg dipanggil.
	// Paksa mode copy (ffmpeg absen mustahil di sini) — validasi jalan duluan
	// hanya untuk branch ffmpeg; guard-only copy tidak butuh crf. Maka uji
	// butuh ffmpeg: skip bila absen.
	if FindFFmpeg() == "" {
		t.Skip("butuh ffmpeg untuk capai validasi angka")
	}
	tmp := t.TempDir()
	src := tmp + "/s.mp4"
	if err := os.WriteFile(src, []byte{0, 0, 0, 24, 'f', 't', 'y', 'p'}, 0o644); err != nil {
		t.Fatal(err)
	}
	for label, cfg := range map[string]map[string]any{
		"crf-negatif":  {"video_crf": -5},
		"crf-lewat":    {"video_crf": 99},
		"crf-teks":     {"video_crf": "x"},
		"bitrate-aneh": {"audio_bitrate": "96k;rm -rf /"},
	} {
		if _, err := Run(src, tmp+"/o.mp4", "video/mp4", cfg); err == nil {
			t.Fatalf("%s harusnya ditolak", label)
		} else {
			t.Logf("%s ditolak: %v", label, err)
		}
	}
	if _, err := Run(src, tmp+"/o.jpg", "image/jpeg",
		map[string]any{"image_max_dim": 0}); err == nil {
		t.Fatal("image_max_dim=0 harusnya ditolak")
	}
	if _, err := Run(src, tmp+"/o.jpg", "image/jpeg",
		map[string]any{"image_quality": 500}); err == nil {
		t.Fatal("image_quality=500 harusnya ditolak")
	}
}
