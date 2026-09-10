# FFmpeg Supply Chain — GuardCompress

Developer **tidak install ffmpeg sendiri**. Kita yang sediakan + pasang otomatis.

## Aset rilis (per version, di GitHub Releases)

```
guardcompress-linux-amd64 / -arm64 / -windows-amd64.exe / -darwin-arm64
ffmpeg-linux-amd64 / -arm64 / -windows-amd64.exe / -darwin-arm64
CHECKSUMS.txt   # sha256 untuk SEMUA file di atas
```

## Cara maintainer menyiapkan ffmpeg static (sekali per update ffmpeg)

Sumber upstream terpercaya (varian **GPL** — pipeline butuh `libx264`):
- Linux amd64/arm64: `https://github.com/BtbN/FFmpeg-Builds/releases` (file `ffmpeg-n*-latest-linux{64,arm64}-gpl-*.tar.xz`, ambil `bin/ffmpeg`)
- Windows: `https://github.com/GyanD/codexffmpeg/releases` (file `*-essentials_build.zip`, ambil `bin/ffmpeg.exe`)
- macOS arm64: `https://evermeet.cx/ffmpeg/getrelease/zip` (build Intel → jalan via Rosetta 2)

Langkah:
1. Download upstream, extract `ffmpeg` / `ffmpeg.exe` saja.
2. Rename jadi `ffmpeg-<os>-<arch>(.exe)`, misal `ffmpeg-windows-amd64.exe`.
3. Masukkan ke draft release + hitung ulang `CHECKSUMS.txt`:
   ```bash
   sha256sum guardcompress-* ffmpeg-* > CHECKSUMS.txt
   ```
4. Publish release. Installer wrapper (php/node/python) otomatis ambil + verifikasi.

## Urutan pencarian ffmpeg oleh Core (`compress.FindFFmpeg`)

1. env `GUARDCOMPRESS_FFMPEG` (di-set installer ke hasil download cache)
2. `ffmpeg(.exe)` di sebelah binary core
3. `ffmpeg` di PATH (milik admin — bonus, bukan syarat)

Tidak ketemu semua = **mode guard-only** (cek keamanan jalan, kompres jadi copy + peringatan di `doctor`).

## Keamanan supply chain

- Semua installer verifikasi SHA256 terhadap `CHECKSUMS.txt` dari release yang sama.
- File yang hash-nya cocok dilewati (idempotent, hemat bandwidth).
- Checksum gagal = file dibuang + installer lapor, tidak pernah dieksekusi.
