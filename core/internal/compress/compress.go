// Package compress: fase kompres — bungkus ffmpeg static.
// Cari ffmpeg dengan urutan:
//  1. env GUARDCOMPRESS_FFMPEG (diisi installer dari hasil download + cek SHA)
//  2. ffmpeg(.exe) sebelah binary core
//  3. ffmpeg di PATH (kalau admin emang udah install)
//
// Kalau nggak ketemu semua: fallback copy file (mode guard-only) biar nggak gagal total.
package compress

import (
	"context"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type Result struct {
	NewBytes int64
	Details  map[string]any
	Thumbs   []Thumb
}

// Thumb: turunan ukuran/format dari gambar utama.
type Thumb struct {
	Path   string `json:"path"`
	Width  int    `json:"width"`
	Bytes  int64  `json:"bytes"`
	Format string `json:"format"`
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

	// File di bawah ambang nggak usah bayar ongkos spawn ffmpeg.
	// Default 0 = selalu kompres (perilaku lama).
	// Rekomendasi situs avatar: 100 (file <100KB langsung copy).
	if kb := minCompressKB(cfg); kb > 0 {
		if fi, err := os.Stat(inPath); err == nil && fi.Size() < int64(kb)*1024 {
			if err := copyFile(inPath, outPath); err != nil {
				return res, err
			}
			fi2, _ := os.Stat(outPath)
			res.NewBytes = fi2.Size()
			res.Details["mode"] = "copy (under min_compress_kb)"
			res.Details["ffmpeg"] = ff
			return res, nil
		}
	}

	crf := "28"
	if v, ok := cfg["video_crf"]; ok {
		crf = fmt.Sprintf("%v", v)
	}
	// Validasi angka config biar errornya jelas di awal, bukan error
	// ffmpeg yang misterius. Injeksi shell mustahil (argv tanpa shell),
	// tapi nilai ngawur tetap ditolak.
	if n, err := strconv.Atoi(crf); err != nil || n < 0 || n > 51 {
		return res, fmt.Errorf("invalid video_crf (0-51): %v", cfg["video_crf"])
	}
	abitrate := "96k"
	if v, ok := cfg["audio_bitrate"]; ok {
		abitrate = fmt.Sprintf("%v", v)
	}
	if ok, _ := regexp.MatchString(`^[0-9]+k$`, abitrate); !ok {
		return res, fmt.Errorf("invalid audio_bitrate (cth 96k): %v", cfg["audio_bitrate"])
	}
	// ANTI-DOWN: batasi thread CPU per job ffmpeg. Default 2 (aman di VPS
	// kecil/shared; 1 upload tak bisa menelan semua core). Naikkan
	// (cth 4-8) hanya di server khusus media + queue terbatas.
	threads := "2"
	if v, ok := cfg["ffmpeg_threads"]; ok {
		threads = fmt.Sprintf("%v", v)
	}
	if n, err := strconv.Atoi(threads); err != nil || n < 1 || n > 32 {
		return res, fmt.Errorf("invalid ffmpeg_threads (1-32): %v", cfg["ffmpeg_threads"])
	}

	var args []string
	// "-threads" global ditaruh di depan agar berlaku untuk semua filter+codec.
	tflag := []string{"-y", "-threads", threads}
	switch {
	case len(mime) >= 5 && mime[:5] == "video":
		args = append(tflag, "-i", inPath,
			"-vcodec", "libx264", "-crf", crf, "-preset", "veryfast",
			"-movflags", "+faststart", "-pix_fmt", "yuv420p",
			"-acodec", "aac", "-b:a", abitrate,
			outPath)
	case len(mime) >= 5 && mime[:5] == "audio" || mime == "application/ogg":
		// Codec per format input; output ext diatur guard.OutExt
		// (wav/flac besar -> mp3 hemat; ogg -> ogg; m4a -> m4a).
		acodec := "libmp3lame"
		if mime == "application/ogg" || mime == "audio/ogg" {
			acodec = "libvorbis"
		} else if mime == "audio/mp4" {
			acodec = "aac"
		}
		args = append(tflag, "-i", inPath, "-codec:a", acodec, "-b:a", abitrate, outPath)
	case mime == "image/jpeg" || mime == "image/png" || mime == "image/webp" || mime == "image/gif":
		maxDim := "1920"
		if v, ok := cfg["image_max_dim"]; ok {
			maxDim = fmt.Sprintf("%v", v)
		}
		if n, err := strconv.Atoi(maxDim); err != nil || n < 64 || n > 8192 {
			return res, fmt.Errorf("invalid image_max_dim (64-8192): %v", cfg["image_max_dim"])
		}
		q := "82"
		if v, ok := cfg["image_quality"]; ok {
			q = fmt.Sprintf("%v", v)
		}
		if n, err := strconv.Atoi(q); err != nil || n < 1 || n > 100 {
			return res, fmt.Errorf("invalid image_quality (1-100): %v", cfg["image_quality"])
		}
		// Kecilkan dimensi bila lebih besar dari maxDim, pertahankan aspek.
		// JPEG/WebP: quality terkontrol. PNG: kompresi max (lossless).
		// GIF: palet optimal 1-pass (tetap animasi).
		args = append(append(tflag, "-i", inPath), imageCodecArgs(mime, maxDim, q)...)
		args = append(args, outPath)
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
	// Di Linux, ffmpeg ikut mati kalau core mati mendadak.
	// (lihat procattr_linux.go; no-op di OS lain)
	setDeathsig(cmd)
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
	// Output 0 byte = hasil korup, jangan dianggap sukses.
	if fi.Size() == 0 {
		os.Remove(outPath)
		return res, fmt.Errorf("ffmpeg produced empty output")
	}
	res.NewBytes = fi.Size()
	res.Details["mode"] = "ffmpeg"
	// Turunan gambar (thumbs + webp) dibuat SETELAH output utama valid.
	// Gagal di turunan tak menggugurkan hasil utama (dicatat, lanjut).
	if isImageMime(mime) {
		makeDerivatives(ctx, ff, inPath, outPath, mime, cfg, &res)
	}
	return res, nil
}

// thumbSpec: satu turunan ukuran. "300" -> suffix "-300w"; map
// {"w":300,"suffix":"-sm"} -> suffix kustom.
type thumbSpec struct {
	w      int
	suffix string
}

// parseThumbs: baca cfg "thumb_widths" ([300, 800] atau [{"w":300,"suffix":"-sm"}]).
func parseThumbs(cfg map[string]any) []thumbSpec {
	raw, ok := cfg["thumb_widths"].([]any)
	if !ok {
		return nil
	}
	var out []thumbSpec
	for _, v := range raw {
		switch t := v.(type) {
		case float64:
			if t >= 16 && t <= 8192 {
				w := int(t)
				out = append(out, thumbSpec{w, "-" + strconv.Itoa(w) + "w"})
			}
		case int:
			if t >= 16 && t <= 8192 {
				out = append(out, thumbSpec{t, "-" + strconv.Itoa(t) + "w"})
			}
		case map[string]any:
			w := 0
			switch n := t["w"].(type) {
			case float64:
				w = int(n)
			case int:
				w = n
			}
			if w < 16 || w > 8192 {
				continue
			}
			sfx, _ := t["suffix"].(string)
			if sfx == "" {
				sfx = "-" + strconv.Itoa(w) + "w"
			}
			// sanitasi suffix: huruf/angka/dash saja
			sfx = regexp.MustCompile(`[^A-Za-z0-9-]`).ReplaceAllString(sfx, "")
			if sfx == "" {
				sfx = "-" + strconv.Itoa(w) + "w"
			}
			out = append(out, thumbSpec{w, sfx})
		}
	}
	return out
}

// probeWidth: lebar gambar via stdlib (jpeg/png/gif). 0 bila tak dikenal (webp).
func probeWidth(path string) int {
	f, err := os.Open(path)
	if err != nil {
		return 0
	}
	defer f.Close()
	c, _, err := image.DecodeConfig(f)
	if err != nil {
		return 0
	}
	return c.Width
}

// makeDerivatives: thumbs multi-ukuran + salinan webp di outDir yang sama.
// Gagal di turunan tak menggugurkan hasil utama (dicatat di thumb_errors).
func makeDerivatives(ctx context.Context, ff, inPath, outPath, mime string, cfg map[string]any, res *Result) {
	specs := parseThumbs(cfg)
	wantWebp, _ := cfg["webp"].(bool)
	if len(specs) == 0 && !wantWebp {
		return
	}
	q := "82"
	if v, ok := cfg["image_quality"]; ok {
		q = fmt.Sprintf("%v", v)
	}
	dir := filepath.Dir(outPath)
	stem := strings.TrimSuffix(filepath.Base(outPath), filepath.Ext(outPath))
	srcW := probeWidth(inPath)

	runFF := func(args []string, dst string) error {
		cmd := exec.CommandContext(ctx, ff, args...)
		setDeathsig(cmd)
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("%v: %s", err, tail(string(out), 500))
		}
		fi, err := os.Stat(dst)
		if err != nil || fi.Size() == 0 {
			return fmt.Errorf("output turunan kosong/hilang: %s", dst)
		}
		return nil
	}
	var made []string // file turunan format-asli (untuk dikonversi webp)
	for _, s := range specs {
		if srcW > 0 && srcW <= s.w {
			continue // jangan upscale: buang-buang CPU
		}
		dst := filepath.Join(dir, stem+s.suffix+filepath.Ext(outPath))
		cargs := append([]string{"-y", "-threads", threadsOf(cfg), "-i", inPath},
			imageCodecArgs(mime, strconv.Itoa(s.w), q)...)
		cargs = append(cargs, dst)
		if err := runFF(cargs, dst); err != nil {
			noteThumbErr(res, dst, err)
			continue
		}
		fi, _ := os.Stat(dst)
		res.Thumbs = append(res.Thumbs, Thumb{Path: dst, Width: s.w, Bytes: fi.Size(), Format: strings.TrimPrefix(filepath.Ext(dst), ".")})
		made = append(made, dst)
	}
	if wantWebp {
		// Konversi dari file JADI (lebih cepat & konsisten daripada dari input).
		targets := append([]string{outPath}, made...)
		for _, src := range targets {
			dst := strings.TrimSuffix(src, filepath.Ext(src)) + ".webp"
			if dst == src {
				continue
			}
			wargs := []string{"-y", "-threads", threadsOf(cfg), "-i", src,
				"-c:v", "libwebp", "-quality", q, dst}
			if err := runFF(wargs, dst); err != nil {
				noteThumbErr(res, dst, err)
				continue
			}
			fi, _ := os.Stat(dst)
			res.Thumbs = append(res.Thumbs, Thumb{Path: dst, Width: 0, Bytes: fi.Size(), Format: "webp"})
		}
	}
}

func noteThumbErr(res *Result, dst string, err error) {
	var list []string
	if v, ok := res.Details["thumb_errors"].([]string); ok {
		list = v
	}
	res.Details["thumb_errors"] = append(list, dst+": "+err.Error())
}

// threadsOf: baca ulang batas thread (dipakai tiap pemanggilan ffmpeg turunan).
func threadsOf(cfg map[string]any) string {
	if v, ok := cfg["ffmpeg_threads"]; ok {
		if s := fmt.Sprintf("%v", v); s != "" {
			if n, err := strconv.Atoi(s); err == nil && n >= 1 && n <= 32 {
				return s
			}
		}
	}
	return "2"
}

// isImageMime: keluarga gambar yang didukung turunan.
func isImageMime(mime string) bool {
	switch mime {
	case "image/jpeg", "image/png", "image/webp", "image/gif":
		return true
	}
	return false
}

// imageCodecArgs: argumen ffmpeg untuk keluarga codec gambar (1 code path
// dipakai output utama + thumbs agar konsisten).
// String filter GIF jangan diubah-ubah: sudah pas dengan yang lolos uji
// (h tanpa kutip sebelum :flags).
func imageCodecArgs(mime, dim, q string) []string {
	switch mime {
	case "image/png":
		return []string{"-vf", "scale=w='min(" + dim + ",iw)':h='-2'",
			"-compression_level", "9"}
	case "image/gif":
		return []string{"-vf", "scale=w='min(" + dim + ",iw)':h=-2:flags=lanczos,split[s0][s1];[s0]palettegen=max_colors=256[p];[s1][p]paletteuse"}
	default:
		return []string{"-vf", "scale=w='min(" + dim + ",iw)':h='-2'",
			"-q:v", q}
	}
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

// minCompressKB: ambang lewati-kompres (0 = selalu kompres).
func minCompressKB(cfg map[string]any) int {
	for _, k := range []string{"min_compress_kb", "minCompressKb"} {
		switch v := cfg[k].(type) {
		case float64:
			if v > 0 {
				return int(v)
			}
		case int:
			if v > 0 {
				return v
			}
		}
	}
	return 0
}

// CacheDir: lokasi lazy-download ffmpeg static + slot admission.
// Bisa dipindah via env GUARDCOMPRESS_CACHE (container/serverless/home read-only).
func CacheDir() string {
	if c := os.Getenv("GUARDCOMPRESS_CACHE"); c != "" {
		return c
	}
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
