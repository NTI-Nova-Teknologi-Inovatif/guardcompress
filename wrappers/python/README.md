# GuardCompress for Python

[![License: MIT](https://img.shields.io/badge/License-MIT-green.svg)](../../LICENSE)

Keamanan + kompresi upload untuk Django/Flask/FastAPI. Thin wrapper di atas
binary inti Go — tanpa dependensi pip.

## Apa itu ini?

- **Wrapper** = fungsi `process()` yang memanggil binary Go lewat
  `subprocess`, lalu menerjemahkan hasilnya jadi dict atau exception
  (`BlockedError` = 422, `BusyError` = 429).
- **Binary inti** = program Go yang berisi SEMUA logika (scan + kompres).
- **FFmpeg** = mesin kompres, diunduh otomatis oleh installer.

## Instalasi (via GitHub — PyPI segera)

```bash
pip install git+https://github.com/NTI-Nova-Teknologi-Inovatif/guardcompress-python.git
python -m guardcompress.install   # unduh binary + ffmpeg (sekali saja)
```

Nanti setelah publish: `pip install guardcompress`.

## Pakai

```python
from guardcompress import process, BlockedError, BusyError, cleanup

try:
    r = process(tmp_path, {"max_mb": 500})
    # pindahkan r["path"] ke storage, lalu:
    cleanup(r["path"])
except BlockedError as e:
    return 422, str(e)
except BusyError:
    return 429, "server penuh, coba lagi"
```

Shortcut: `image()`, `video()`, `audio()`. Batch beda jenis sekaligus:
`batch({"avatar": p1, "klip": {"path": p2, "opts": {...}}})`.

Env: `GUARDCOMPRESS_BIN`, `GUARDCOMPRESS_FFMPEG`, `GUARDCOMPRESS_CACHE`.

Detail kontrak, config, dan keamanan: repo utama
[guardcompress](https://github.com/NTI-Nova-Teknologi-Inovatif/guardcompress)
(`docs/CONTRACT.md`, `docs/CONFIG.md`, `SECURITY.md`). Lisensi MIT.
