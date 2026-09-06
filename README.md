# GuardCompress

Middleware keamanan + kompresi media yang embeddable untuk semua bahasa web.

**Cara kerja:** `Upload -> Guard (validasi MIME + scan) -> Compress (FFmpeg) -> Storage`

```
[PHP/Node/Python/Go] --exec--> [guardcompress binary (Go)] --call--> [ffmpeg static]
        ^                                  |
        |----------- report.json -----------+
```

## Struktur Monorepo

```
core/               # Core Engine Go -> single-binary CLI
  main.go           # CLI: check --in --out-dir --json
  internal/guard/   # Guard Phase: magic numbers + heuristic/YARA
  internal/compress/# Compress Phase: FFmpeg wrapper
  rules/            # YARA rules (embedded via go:embed)
wrappers/
  php/              # composer: guardcompress/php
  node/             # npm: guardcompress
  python/           # pip: guardcompress
  go/               # go sdk (os/exec wrapper)
docs/ARCHITECTURE.md
```

## Quick Start (Dev)

```powershell
# 1. Build core
make build
# 2. Test dengan file dummy
.\core\bin\guardcompress-windows-amd64.exe check --in sample.mp4 --out-dir ./tmp/out --json
# 3. Coba wrapper Node/Python
node wrappers/node/example.js
python wrappers/python/example.py
```

## Kontrak CLI (Abadi, jangan diubah sembarangan)

```
guardcompress check --in <path> --out-dir <dir> --config <json> --json
exit 0 = clean, exit 2 = blocked, exit 1 = error
stdout: report.json
```

Lihat `docs/ARCHITECTURE.md` untuk detail Data Flow & strategi FFmpeg zero-install.
