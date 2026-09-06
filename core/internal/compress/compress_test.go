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
