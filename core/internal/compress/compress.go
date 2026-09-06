// Package compress: Compress Phase - bungkus FFmpeg static.
// Strategi zero-install: cari ffmpeg dengan urutan:
//  1. env GUARDCOMPRESS_FFMPEG (di-set wrapper, hasil lazy-download + SHA verify)
//  2. ./ffmpeg(.exe) di sebelah binary core
//  3. ffmpeg di PATH (fallback kalau admin memang sudah install)
// Kalau ffmpeg tidak ketemu: fallback copy file (guard-only mode) agar tidak gagal total.
package compress

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
)

type Result struct {
	NewBytes int64
	Details  map[string]any
}

func findFFmpeg() string {
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
	ff := findFFmpeg()
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
	default: // image & lain: copy saja di v1
		if err := copyFile(inPath, outPath); err != nil {
			return res, err
		}
		fi, _ := os.Stat(outPath)
		res.NewBytes = fi.Size()
		res.Details["mode"] = "copy (non-av)"
		res.Details["ffmpeg"] = ff
		return res, nil
	}

	cmd := exec.Command(ff, args...)
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
