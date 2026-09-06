# ARCHITECTURE — GuardCompress

## 1. Prinsip
- **Embeddable:** dipanggil via `exec/spawn/subprocess`, tanpa daemon/Docker.
- **Polyglot:** 1 Core (Go) + N Thin Wrapper. Logika hanya di Core.
- **Zero-install:** FFmpeg static lazy-download, YARA embedded, fallback guard-only.
- **Kontrak stabil:** `check --in --out-dir --config --json`, exit `0/2/1`.

## 2. Data Flow
```
Upload tmp -> wrapper proses():
  exec guardcompress check
    -> guard.Scan(): stat, max_mb, http.DetectContentType (512B), allowlist,
                     heuristic token (EICAR/webshell), return Allowed+Reason
    -> compress.Run(): findFFmpeg() [ENV > sidecar > PATH], spawn ffmpeg
                       video: libx264 crf+preset+faststart, audio: mp3 96k
                       fallback: copy bila ffmpeg absen
  <- stdout report.json -> wrapper raise/return -> app simpan ke S3/DB
```

## 2b. Struktur output (gampang dicek manual)
```
<out-dir>/
  video_liburan_anak.mp4   # nama ikut file asli, disanitasi; ext SELALU dari MIME asli
  report.json              # laporan pretty-print (ada juga saat blocked, tanpa file media)
```
Opsi penamaan via `--config {"output": ...}`:
`original` (default) | `uuid` (acak 16 hex) | `"teks kustom"` (disanitasi).
Contoh: `evil.mp4.php` -> `evil_mp4_php.mp4`, `../../etc/passwd` -> `passwd.bin`.

## 3. Dependensi
| Komponen | Strategi v1 | Roadmap v2 |
|---|---|---|
| FFmpeg | static build per OS/arch, lazy-download ke `~/.cache/guardcompress`, SHA verify, `GUARDCOMPRESS_FFMPEG` | campur `go:embed` untuk 1 platform populer |
| Malware DB | heuristic + `rules/*.yar` placeholder | `yara-x` Go binding + `go:embed rules/` + auto-update via rilis binary |
| ClamAV | opsional hook `clamdscan` bila ada | - |

## 4. Matrix rilis
`linux-amd64, linux-arm64, windows-amd64, darwin-arm64` via `.github/workflows/release.yml`.

## 5. Keamanan
- Jangan percaya extension; pakai magic numbers.
- `escapeshellarg` / arg array (tanpa shell) di semua wrapper.
- Timeout berlapis: ffmpeg core 100s < wrapper 120s (core selalu yang menuai ffmpeg).
  File besar naikkan via `timeoutSec` + pindah ke queue (lihat `examples/`).
- Hapus tmp `in/out` setelah selesai; jangan log isi file.
- AUDIT: report JSON berisi path server (`in_path`, `ffmpeg`) — jangan kirim
  mentah ke browser, kirim ringkasannya saja (`stored_at`, ukuran).
- AUDIT: symlink input ditolak; ukuran input dicek ulang setelah scan (TOCTOU).
