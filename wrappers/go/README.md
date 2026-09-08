# GuardCompress for Go

[![License: MIT](https://img.shields.io/badge/License-MIT-green.svg)](../../LICENSE)

Keamanan + kompresi upload untuk `net/http`, Gin, Echo. SDK Go di atas
binary inti — tanpa dependensi modul.

## Apa itu ini?

- **Wrapper** = fungsi `gc.Process()` yang memanggil binary inti lewat
  `exec`, lalu menerjemahkan hasilnya jadi `Result` atau error
  (`gc.IsBlocked(err)` = 422, `gc.IsBusy(err)` = 429).
- **Binary inti** = program Go (`guardcompress-*`) yang berisi SEMUA
  logika (scan + kompres). Unduh dari halaman Releases, taruh di PATH
  atau set `GUARDCOMPRESS_BIN`.

## Instalasi

```bash
go get github.com/NTI-Nova-Teknologi-Inovatif/guardcompress-go
```

Lalu sediakan binary inti (lihat Releases repo utama) — atau set
`GUARDCOMPRESS_BIN` ke path binary yang sudah ada.

## Pakai

```go
import gc "github.com/NTI-Nova-Teknologi-Inovatif/guardcompress-go"

res, err := gc.Process(tmpPath, map[string]any{"max_mb": 500})
if gc.IsBlocked(err) {
    // HTTP 422
}
if gc.IsBusy(err) {
    // HTTP 429 + retry
}
// pindahkan res.Path ke storage, lalu os.RemoveAll(tmp)
```

Shortcut: `gc.Image()`, `gc.Video()`, `gc.Audio()`.
Batch: `gc.Batch([]gc.BatchItem{{Name: "avatar", Path: p1}}, opts)` —
hasil urut = urutan input, item terblokir tidak menggugurkan yang lain.

Detail kontrak, config, dan keamanan: repo utama
[guardcompress](https://github.com/NTI-Nova-Teknologi-Inovatif/guardcompress)
(`docs/CONTRACT.md`, `docs/CONFIG.md`, `SECURITY.md`). Lisensi MIT.
