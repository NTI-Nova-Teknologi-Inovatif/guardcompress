# GuardCompress for Go

[![License: MIT](https://img.shields.io/badge/License-MIT-green.svg)](../../LICENSE)

Keamanan + kompresi upload untuk `net/http`, Gin, Echo. SDK Go di atas
binary inti — tanpa dependensi modul.

```bash
go get github.com/guardcompress/guardcompress/wrappers/go
```

```go
import gc "github.com/guardcompress/guardcompress/wrappers/go"

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

Binary inti dicari di `GUARDCOMPRESS_BIN`, lalu `~/.cache/guardcompress`.

Detail kontrak, config, dan keamanan: repo utama
[guardcompress](https://github.com/NTI-Nova-Teknologi-Inovatif/guardcompress)
(`docs/CONTRACT.md`, `docs/CONFIG.md`, `SECURITY.md`). Lisensi MIT.
