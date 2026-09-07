# GuardCompress

Middleware keamanan + kompresi media yang nempel langsung di kode web.
Buat developer, bukan end-user.

**Alur:** `Upload -> Guard (cek format asli + scan) -> Compress (FFmpeg) -> balik ke web`

```
[PHP/Node/Python/Go] --exec--> [guardcompress binary (Go)] --call--> [ffmpeg static]
        ^                                  |
        |----------- report.json -----------+
```

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
docs/               # ARCHITECTURE, CONFIG, CONTRACT, FFMPEG
examples/           # contoh queue Laravel / BullMQ / Celery
```

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
