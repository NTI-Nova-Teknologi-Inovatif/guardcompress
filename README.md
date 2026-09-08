# GuardCompress

[![License: MIT](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)
[![Release](https://img.shields.io/github/v/release/NTI-Nova-Teknologi-Inovatif/guardcompress)](https://github.com/NTI-Nova-Teknologi-Inovatif/guardcompress/releases)

Middleware keamanan + kompresi media yang nempel langsung di kode web.
Buat developer, bukan end-user. Satu binary Go, tanpa daemon, tanpa Docker.

## Masalahnya

Form upload adalah pintu masuk favorit penyerang: `shell.php` yang diganti
nama jadi `foto.png`, webshell yang ditempel di ekor gambar valid, file
zip/HTML yang menyamar jadi media. Validasi ekstensi tidak cukup —
GuardCompress memeriksa **isi asli file** (magic numbers), memindai pola
berbahaya, lalu mengompresi ulang media lewat FFmpeg sehingga file yang
disimpan adalah hasil render ulang yang bersih.

**Alur:** `Upload -> Guard (cek format asli + scan) -> Compress (FFmpeg) -> balik ke web`

```
[PHP/Node/Python/Go] --exec--> [guardcompress binary (Go)] --call--> [ffmpeg static]
        ^                                  |
        |----------- report.json -----------+
```

## Fitur

- **Deteksi format asli** — magic numbers, bukan ekstensi. `evil.mp4.php` ketahuan.
- **Pindai webshell & skrip** — tag `<?php`/`<%`/`<script>`, `eval`, `base64_decode`,
  `shell_exec`, perintah `cmd.exe`/`/bin/sh`, marker `c99shell`, string EICAR.
- **Fail-closed** — ragu sedikit = tolak. Lebih baik false positive daripada lolos.
- **Sanitasi nama file** — `../../etc/passwd` jadi `passwd.bin`, ekstensi selalu dari MIME asli.
- **Kompresi ulang** — gambar/video/audio diperkecil via FFmpeg static (lazy-download + verifikasi SHA).
- **Isolasi** — kerja di folder tmp unik (0700), tanpa shell, tanpa jaringan, tmp dibersihkan otomatis.
- **Verifikasi simpanan** — `verify` memastikan file di storage tidak diubah setelah lolos.
- **Karantina opsional + hook ClamAV** bila `clamdscan` tersedia di server.
- **Nol dependensi** — core Go stdlib-only; semua wrapper stdlib-only.

## Instalasi

```bash
composer require guardcompress/php        # PHP (Laravel/WordPress)
npm install guardcompress                 # Node.js
pip install guardcompress                 # Python
go get github.com/guardcompress/guardcompress/wrappers/go   # Go
```

Binary inti + FFmpeg diunduh otomatis saat instalasi (postinstall) dari
GitHub Releases — cocok dengan OS/arch mesin. Atau override manual:

```bash
php bin/install-binary.php v0.1.0   # PHP, sekali saja
```

Env: `GUARDCOMPRESS_BIN` (path binary), `GUARDCOMPRESS_FFMPEG` (path ffmpeg).

## Contoh pakai

```php
// Laravel
use GuardCompress\GuardCompress;
try {
    $r = GuardCompress::process($request->file('video')->getRealPath(), ['max_mb' => 500]);
    Storage::putFile('media', new File($r->path));
} catch (\GuardCompress\InfectedFileException $e) {
    return response()->json(['blocked' => $e->getMessage()], 422);
}
```

```js
// Node.js / Express
const gc = require('guardcompress');
try {
  const { path } = gc.processFile(req.file.path, { max_mb: 500 });
  // simpan path ke storage
} catch (e) {
  if (e.code === 'BLOCKED') return res.status(422).json({ blocked: e.message });
  if (e.code === 'BUSY') return res.status(429).json({ retry: true });
  throw e;
}
```

```python
# Django / Flask
from guardcompress import process, BlockedError
try:
    r = process(tmp_path, {"max_mb": 500})
except BlockedError as e:
    return 422, str(e)
```

```go
// Go net/http
import gc "github.com/guardcompress/guardcompress/wrappers/go"
res, err := gc.Process(tmpPath, map[string]any{"max_mb": 500})
```

Contoh antrean (Laravel job / BullMQ / Celery) ada di `examples/`.

## Hasil (verdicts)

| Exit | Artinya | Aksi app |
|---|---|---|
| `0` | bersih, file + `report.json` di out-dir | simpan + catat sha256 |
| `2` | **diblokir** + alasan | tolak (HTTP 422), opsional karantina |
| `1` | error / server penuh (`busy`) | coba lagi (HTTP 429 + retry) |

`report.json` berisi path server — jangan kirim mentah ke browser,
kirim ringkasannya saja.

## Isi repo

```
core/               # inti Go, jadi satu binary CLI
  main.go           # perintah: check | verify | doctor | init
  internal/guard/   # cek format + scan
  internal/compress/# bungkus ffmpeg
  rules/            # aturan YARA (dipakai penuh di v2)
wrappers/
  php/              # composer: guardcompress/php
  node/             # npm: guardcompress
  python/           # pip: guardcompress
  go/               # SDK go
docs/               # ARCHITECTURE, CONFIG, CONTRACT, FFMPEG, RELEASE
examples/           # contoh queue Laravel / BullMQ / Celery
web/                # demo upload (contoh, bukan produksi)
```

## Repo (multi-repo, mono-sumber)

| Repo | Isi | Publish |
|---|---|---|
| `NTI-Nova-Teknologi-Inovatif/guardcompress` | full monorepo | kode sumber |
| `NTI-Nova-Teknologi-Inovatif/guardcompress-js` | `wrappers/node/` | npm `guardcompress` |
| `NTI-Nova-Teknologi-Inovatif/guardcompress-php` | `wrappers/php/` | Packagist `guardcompress/php` |
| `NTI-Nova-Teknologi-Inovatif/guardcompress-python` | `wrappers/python/` | PyPI `guardcompress` |
| `NTI-Nova-Teknologi-Inovatif/guardcompress-go` | `wrappers/go/` | `go get .../wrappers/go` |

Ngoding di repo full; wrapper dibagi ke repo bahasa lewat subtree split
otomatis (`.github/workflows/subtree.yml`).

## Keamanan & batasan (jujur)

- Satu lapis pertahanan berlapis — **bukan** pengganti antivirus enterprise,
  firewall, pentest, atau hardening server.
- Fail-closed: file aneh tapi jinak bisa ikut tertolak (misal MP3 yang
  liriknya mengandung kata `eval(`). Itu disengaja.
- Tidak ada scanner yang janji 100% — termasuk kami. Batas yang diakui ada
  di `docs/ARCHITECTURE.md` §5–§6.
- Lapor celah privat, bukan issue publik — lihat `SECURITY.md`.
- Berjalan 100% lokal: tanpa telemetri, tanpa upload sampel ke pihak ketiga.

## Lisensi & hukum

- Kode: **MIT** (`LICENSE`).
- FFmpeg binary: redistribusi wajib LGPL + atribusi + `COPYING.LGPLv2.1`
  (lihat `THIRD-PARTY-NOTICES.md`, `docs/FFMPEG.md`).
- ClamAV: tidak dibundel, ikut lisensi instalasi user.
- Nol dependensi npm/composer/pip pihak ketiga — semua stdlib.
- Panduan rilis & kebijakan: `docs/RELEASE.md`, `SECURITY.md`,
  `CONTRIBUTING.md`, `CODE_OF_CONDUCT.md`.

## Coba-coba (dev)

```powershell
# 1. Build core
go build -o core/bin/guardcompress-windows-amd64.exe ./core
# 2. Cek file
.\core\bin\guardcompress-windows-amd64.exe check --in foto.png --out-dir ./tmp/out --json
# 3. Cek kapasitas mesin
.\core\bin\guardcompress-windows-amd64.exe doctor
```

Linux/macOS tinggal ganti nama binary-nya (`guardcompress-linux-amd64` dst,
lihat `.github/workflows/release.yml`).

## Kontrak CLI

```
guardcompress check --in <path> --out-dir <dir> [--config <json>] [--json]
exit 0 = bersih, exit 2 = diblokir, exit 1 = error
stdout: report.json
```

Jangan ubah kontrak ini sembarangan — semua wrapper ngandalin formatnya.
Detail ada di `docs/ARCHITECTURE.md`, daftar config di `docs/CONFIG.md`.
