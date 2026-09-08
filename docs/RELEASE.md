# Struktur Rilis GuardCompress

## Repo

| Repo | Isi | Publish |
|---|---|---|
| `NeuNovaTech/guardcompress` | full monorepo (core + semua wrapper + docs) | kode sumber |
| `NeuNovaTech/guardcompress-js` | `wrappers/node/` saja | npm (`guardcompress`) |
| `NeuNovaTech/guardcompress-php` | `wrappers/php/` saja | Packagist (`guardcompress/php`) |
| `NeuNovaTech/guardcompress-python` | `wrappers/python/` saja | PyPI (`guardcompress`) |
| `NeuNovaTech/guardcompress-go` | `wrappers/go/` saja | `go get github.com/guardcompress/guardcompress/wrappers/go` |

## Alur rilis

1. Tag `vX.Y.Z` di repo full → GitHub Actions menjalankan `release.yml`.
2. `release.yml` mem-build 4 binary Go + membuat GitHub Release dengan artifact.
3. **Subtree split** (otomatis, bawah) mendorong masing-masing folder wrapper ke repo bahasanya via PAT.
4. Register versi di npm / Packagist / PyPI (manual sekali, atau gunakan GitHub Action publish masing-masing).

## Ketentuan hukum

- Kode GuardCompress: **MIT** (file `LICENSE`).
- FFmpeg: binary redistribusi harus varian **LGPL** saja + atribusi + `COPYING.LGPLv2.1` + link source. Lihat `THIRD-PARTY-NOTICES.md` dan `docs/FFMPEG.md`.
- ClamAV: tidak dibundel, lisensi ikut instalasi user.
- Go stdlib: BSD-style, tidak menular.
- Tidak ada dependensi npm/composer/pip pihak ketiga — semua wrapper stdlib-only.
- Kontribusi dirilis di bawah MIT (lihat `CONTRIBUTING.md`).
- Celah keamanan: lihat `SECURITY.md` (responsible disclosure, safe harbor).
