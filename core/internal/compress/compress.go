// Package compress: Compress Phase - bungkus FFmpeg static.
// Strategi zero-install: cari ffmpeg dengan urutan:
//  1. env GUARDCOMPRESS_FFMPEG (di-set wrapper, hasil lazy-download + SHA verify)
//  2. ./ffmpeg(.exe) di sebelah binary core
//  3. ffmpeg di PATH (fallback kalau admin memang sudah install)
//
// Kalau ffmpeg tidak ketemu: fallback copy file (guard-only mode) agar tidak gagal total.
package compress

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

type Result struct {
	NewBytes int64
	Details  map[string]any
}

func FindFFmpeg() string {
	if p := os.Getenv("GUARDCOMPRESS_FFMPEG"); p != "" {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	exe, _ := os.Executable()
	for _, c := range []string{filepath.Join(filepath.Dir(exe), "ffmpeg"), filepath.Join(filepath.Dir(exe), "ffmpeg.exe")} {
		if _, err := os.Stat(c); err == nil {
			return c
		}
	}
	if p, err := exec.LookPath("ffmpeg"); err == nil {
		return p
	}
	return ""
}

func Run(inPath, outPath, mime string, cfg map[string]any) (Result, error) {
	res := Result{Details: map[string]any{}}
	ff := FindFFmpeg()
	if ff == "" {
		// Guard-only fallback: copy tanpa kompresi
		if err := copyFile(inPath, outPath); err != nil {
			return res, err
		}
		fi, _ := os.Stat(outPath)
		res.NewBytes = fi.Size()
		res.Details["mode"] = "copy (ffmpeg not found)"
		return res, nil
	}

	crf := "28"
	if v, ok := cfg["video_crf"]; ok {
		crf = fmt.Sprintf("%v", v)
	}
	abitrate := "96k"
	if v, ok := cfg["audio_bitrate"]; ok {
		abitrate = fmt.Sprintf("%v", v)
	}

	var args []string
	switch {
	case len(mime) >= 5 && mime[:5] == "video":
		args = []string{"-y", "-i", inPath,
			"-vcodec", "libx264", "-crf", crf, "-preset", "veryfast",
			"-movflags", "+faststart", "-pix_fmt", "yuv420p",
			"-acodec", "aac", "-b:a", abitrate,
			outPath}
	case len(mime) >= 5 && mime[:5] == "audio":
		args = []string{"-y", "-i", inPath, "-codec:a", "libmp3lame", "-b:a", abitrate, outPath}
	case mime == "image/jpeg" || mime == "image/png" || mime == "image/webp":
		maxDim := "1920"
		if v, ok := cfg["image_max_dim"]; ok {
			maxDim = fmt.Sprintf("%v", v)
		}
		q := "82"
		if v, ok := cfg["image_quality"]; ok {
			q = fmt.Sprintf("%v", v)
		}
		// Kecilkan dimensi bila lebih besar dari maxDim, pertahankan aspek.
		// JPEG/WebP: quality terkontrol. PNG: kompresi max (lossless).
		args = []string{"-y", "-i", inPath,
			"-vf", "scale=w='min(" + maxDim + ",iw)':h='-2'",
			"-q:v", q,
			outPath}
		if mime == "image/png" {
			args = []string{"-y", "-i", inPath,
				"-vf", "scale=w='min(" + maxDim + ",iw)':h='-2'",
				"-compression_level", "9",
				outPath}
		}
	default: // mime tak dikenal (tak lolos guard normal): copy aman
		if err := copyFile(inPath, outPath); err != nil {
			return res, err
		}
		fi, _ := os.Stat(outPath)
		res.NewBytes = fi.Size()
		res.Details["mode"] = "copy (non-av)"
		res.Details["ffmpeg"] = ff
		return res, nil
	}
	ctx, cancel := ctxTimeout(cfg)
	defer cancel()
	cmd := exec.CommandContext(ctx, ff, args...)
	out, err := cmd.CombinedOutput()
	res.Details["ffmpeg"] = ff
	res.Details["ffmpeg_log_tail"] = tail(string(out), 2000)
	if err != nil {
		return res, fmt.Errorf("ffmpeg failed: %v", err)
	}
	fi, err := os.Stat(outPath)
	if err != nil {
		return res, err
	}
	// AUDIT: output 0 byte = hasil korup, jangan pernah dianggap sukses.
	if fi.Size() == 0 {
		os.Remove(outPath)
		return res, fmt.Errorf("ffmpeg produced empty output")
	}
	res.NewBytes = fi.Size()
	res.Details["mode"] = "ffmpeg"
	return res, nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}

func tail(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[len(s)-n:]
}

// ctxTimeout: batas waktu ffmpeg agar file jahat/korup yang bikin hang
// tidak menggantung worker selamanya. Default 100s (harus < timeout wrapper
// 120s agar core yang selalu menuai ffmpeg, bukan wrapper).
// Override via cfg "timeout_sec" / "timeoutSec" (file besar + queue job).
func ctxTimeout(cfg map[string]any) (context.Context, context.CancelFunc) {
	secs := 100.0
	for _, k := range []string{"timeout_sec", "timeoutSec"} {
		switch v := cfg[k].(type) {
		case float64:
			if v > 0 {
				secs = v
			}
		case int:
			if v > 0 {
				secs = float64(v)
			}
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(secs*float64(time.Second)))
	return ctx, cancel
}

// CacheDir: lokasi lazy-download ffmpeg static (~/.cache/guardcompress).
func CacheDir() string {
	if h, err := os.UserHomeDir(); err == nil && h != "" {
		return filepath.Join(h, ".cache", "guardcompress")
	}
	return filepath.Join(os.TempDir(), "guardcompress-cache")
}

// ProbeFFmpegVersion: baris pertama `ffmpeg -version`, "" bila gagal.
func ProbeFFmpegVersion(ff string) string {
	cmd := exec.Command(ff, "-version")
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	line := string(out)
	for i, c := range line {
		if c == '\n' {
			return line[:i]
		}
	}
	if len(line) > 120 {
		return line[:120]
	}
	return line
}
