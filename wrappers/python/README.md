# GuardCompress for Python

[![License: MIT](https://img.shields.io/badge/License-MIT-green.svg)](../../LICENSE)
[![PyPI](https://img.shields.io/pypi/v/guardcompress)](https://pypi.org/project/guardcompress/)

Keamanan + kompresi upload untuk Django/Flask/FastAPI. Thin wrapper di atas
binary inti Go — tanpa dependensi pip.

```bash
pip install guardcompress
python -m guardcompress.install   # unduh binary + ffmpeg (sekali saja)
```

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
