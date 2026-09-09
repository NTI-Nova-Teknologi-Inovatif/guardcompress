# 🛡️ GuardCompress

[![License: MIT](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)
[![Release](https://img.shields.io/github/v/release/NTI-Nova-Teknologi-Inovatif/guardcompress)](https://github.com/NTI-Nova-Teknologi-Inovatif/guardcompress/releases)
[![CI](https://github.com/NTI-Nova-Teknologi-Inovatif/guardcompress/actions/workflows/ci.yml/badge.svg)](https://github.com/NTI-Nova-Teknologi-Inovatif/guardcompress/actions/workflows/ci.yml)
[![Go](https://img.shields.io/badge/Go-1.22+-00ADD8?logo=go)](core/)

**Upload file tanpa cemas.** Setiap file diperiksa isinya (bukan ekstensinya),
dipindai webshell, lalu media dikompresi ulang — dalam **satu panggilan**,
langsung di kode web kamu. Tanpa daemon. Tanpa Docker. Tanpa dependensi.

```php
$r = GuardCompress::process($upload, ['max_mb' => 500]);  // bersih? simpan. jahat? 422.
```

## ⚡ Coba 30 detik

```powershell
# 1. Ambil binary (Windows; Linux/macOS/darwin ada di Releases)
Invoke-WebRequest https://github.com/NTI-Nova-Teknologi-Inovatif/guardcompress/releases/download/v0.1.2/guardcompress-windows-amd64.exe -OutFile gc.exe
# 2. Cek file
.\gc.exe check --in foto.png --out-dir ./out --json
# exit 0 = bersih | exit 2 = diblokir | exit 1 = error
```

## ✨ Kenapa GuardCompress?

| Serangan nyata | GuardCompress |
|---|---|
| `shell.php` diganti nama `foto.png` | ❌ Tolak — isi dibaca via magic numbers, bukan nama |
| Webshell ditempel di ekor gambar | ❌ Tolak — pindai token (`<?php`, `eval(`, `c99shell`, ...) |
| Shell di komentar JPEG / chunk PNG terkompres | ❌ Tolak — segmen COM + zTXt/iTXt dibuka & dipindai |
| `foto.jpg.php` (nama menipu) | ❌ Tolak — aturan ekstensi eksekusi |
| SVG ber-script (opt-in) | 🧹 Sanitasi — buang script/event-handler, bentuk utuh |
| Foto 8MB untuk avatar 100px | ✅ Kompres otomatis via FFmpeg |
| `../../etc/passwd` sebagai nama file | ✅ Disanitasi jadi `passwd.bin` |

Daftar lengkap yang terbukti tertangkap: [`docs/THREATS.md`](docs/THREATS.md).

## 📁 File yang didukung (15 jenis)

| Gambar | Video | Audio |
|---|---|---|
| `jpg` `jpeg` `png` `webp` `gif` | `mp4` `mov` `webm` `mkv` `avi` | `mp3` `wav` `ogg` `oga` `m4a` `flac` |

Selain itu = ditolak (`mime not allowed`). Matriks lengkap (magic numbers,
output, konversi): [`docs/FILE-TYPES.md`](docs/FILE-TYPES.md).

## 📦 Instalasi

> **Status: hanya GitHub** (binary v0.1.2 ✅ · FFmpeg menyusul · npm/Packagist/PyPI belum).

| Bahasa | Perintah |
|---|---|
| PHP | composer via VCS [`guardcompress-php`](https://github.com/NTI-Nova-Teknologi-Inovatif/guardcompress-php) + `php bin/install-binary.php` |
| Node | `npm install NTI-Nova-Teknologi-Inovatif/guardcompress-js` |
| Python | `pip install git+https://github.com/NTI-Nova-Teknologi-Inovatif/guardcompress-python.git` + `python -m guardcompress.install` |
| Go | `go get github.com/NTI-Nova-Teknologi-Inovatif/guardcompress-go` |

Binary inti terunduh + verifikasi SHA otomatis. Detail per bahasa ada di
README masing-masing repo.

## 🔌 Pakai (4 bahasa, pola sama)

```php
// Laravel — tangkap 2 exception, selesai
try {
    $r = GuardCompress::process($request->file('video')->getRealPath(), ['max_mb' => 500]);
    Storage::putFile('media', new File($r->path));
} catch (\GuardCompress\InfectedFileException $e) { return response()->json(['blocked' => $e->getMessage()], 422); }
```

```js
// Express — tangkap 2 kode, selesai
try {
  const { path } = gc.processFile(req.file.path, { max_mb: 500 });
} catch (e) {
  if (e.code === 'BLOCKED') return res.status(422).json({ blocked: e.message });
  if (e.code === 'BUSY') return res.status(429).json({ retry: true });
  throw e;
}
```

```python
# Django/Flask
try:
    r = process(tmp_path, {"max_mb": 500})
except BlockedError as e:
    return 422, str(e)
```

```go
// net/http
res, err := gc.Process(tmpPath, map[string]any{"max_mb": 500})
if gc.IsBlocked(err) { /* 422 */ }
```

Contoh antrean (Laravel job / BullMQ / Celery): [`examples/`](examples/).

## 🧠 Cara kerja

```
Upload ─▶ Guard ──────────────▶ Compress ──────▶ Web
           │ cek format asli    │ FFmpeg         simpan + sha256
           │ pindai webshell    │ (guard-only bila tak ada)
           │ sanitasi nama
           └─ jahat? TOLAK (422) + alasan
```

- **Fail-closed**: ragu sedikit = tolak. False positive diterima, lolos tidak.
- **Isolasi**: folder tmp unik 0700 per proses, tanpa shell, tanpa jaringan, auto-cleanup.
- **Terus diawasi**: `verify --expect-sha256` memastikan simpanan tak berubah.
- **Lapis kedua (opsional)**: hook ClamAV lokal + cek reputasi hash VirusTotal.
- Arsitektur penuh: [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md).

## 📚 Dokumen

| Dokumen | Isi |
|---|---|
| [`docs/FILE-TYPES.md`](docs/FILE-TYPES.md) | Matriks jenis file: ekstensi → magic → output |
| [`docs/THREATS.md`](docs/THREATS.md) | Ancaman yang terbukti tertangkap (dan yang tidak) |
| [`docs/CONFIG.md`](docs/CONFIG.md) | Semua kenop: batas, thread, timeout, thumbs, karantina |
| [`docs/CONTRACT.md`](docs/CONTRACT.md) | Kontrak sinyal per bahasa (jangan diubah sembarangan) |
| [`docs/FAQ.md`](docs/FAQ.md) · [`docs/GLOSSARY.md`](docs/GLOSSARY.md) · [`docs/COMPARISON.md`](docs/COMPARISON.md) | Tanya-jawab, istilah, perbandingan |
| [`docs/FFMPEG.md`](docs/FFMPEG.md) · [`docs/RELEASING.md`](docs/RELEASING.md) · [`docs/RELEASE.md`](docs/RELEASE.md) | Supply chain, panduan rilis, struktur repo |

## 🗂️ Repo

Ngoding di monorepo ini; 4 repo bahasa tersinkron otomatis:

| Repo | Isi |
|---|---|
| [`guardcompress`](https://github.com/NTI-Nova-Teknologi-Inovatif/guardcompress) | full (kamu di sini) |
| [`guardcompress-js`](https://github.com/NTI-Nova-Teknologi-Inovatif/guardcompress-js) → npm | [`guardcompress-php`](https://github.com/NTI-Nova-Teknologi-Inovatif/guardcompress-php) → Packagist |
| [`guardcompress-python`](https://github.com/NTI-Nova-Teknologi-Inovatif/guardcompress-python) → PyPI | [`guardcompress-go`](https://github.com/NTI-Nova-Teknologi-Inovatif/guardcompress-go) → go get |

## 🔒 Keamanan & lisensi

- Satu lapis pertahanan — **bukan** pengganti AV enterprise/pentest. Batas jujur di [`docs/THREATS.md`](docs/THREATS.md).
- Lapor celah **privat** (jangan issue publik): [`.github/SECURITY.md`](.github/SECURITY.md).
- 100% lokal, tanpa telemetri (kecuali hook VirusTotal yang kamu nyalakan sendiri).
- Kode **MIT** ([LICENSE](LICENSE)) · FFmpeg LGPL ([docs/THIRD-PARTY-NOTICES.md](docs/THIRD-PARTY-NOTICES.md)) · Kontribusi: [.github/CONTRIBUTING.md](.github/CONTRIBUTING.md).

## 💻 Dev lokal

```powershell
go build -o core/bin/guardcompress-windows-amd64.exe ./core
.\core\bin\guardcompress-windows-amd64.exe check --in foto.png --out-dir ./tmp/out --json
.\core\bin\guardcompress-windows-amd64.exe doctor
```
Demo web: `web/` (`GUARDCOMPRESS_BIN=... node server.js` → http://localhost:8080).
