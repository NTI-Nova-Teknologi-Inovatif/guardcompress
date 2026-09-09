# GuardCompress for Node.js

[![License: MIT](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)

Keamanan + kompresi upload untuk Express/Fastify/Nest. Thin wrapper di atas
binary inti Go — tanpa dependensi npm.

## Apa itu ini?

- **Wrapper** = penerjemah: kode JS memanggil binary Go lewat `spawn`,
  lalu menerjemahkan `report.json` jadi return/throw yang enak dipakai.
- **Binary inti** = program Go (`guardcompress-*.exe`) yang berisi SEMUA
  logika (scan + kompres). Satu file, tanpa install lain.
- **FFmpeg** = mesin kompres video/gambar/audio, diunduh otomatis.

## Instalasi

> **Hanya via GitHub untuk saat ini** — belum ada di npm.

```bash
npm install NTI-Nova-Teknologi-Inovatif/guardcompress-js
# binary otomatis terunduh saat install (ffmpeg menyusul; sementara mode guard-only).
# Belum ada rilis? installer diam — mode guard-only, tanpa gagalkan install.
```

## Pakai

```js
const gc = require('guardcompress');

try {
  const { path } = gc.processFile(req.file.path, { max_mb: 500 });
  // pindahkan path ke storage, lalu:
  gc.cleanup();
} catch (e) {
  if (e.code === 'BLOCKED') return res.status(422).json({ blocked: e.message });
  if (e.code === 'BUSY') return res.status(429).json({ retry: true });
  throw e;
}
```

Shortcut per jenis: `gc.image(p, opts)`, `gc.video(p, opts)`, `gc.audio(p, opts)`.
Batch: `gc.batch({avatar: p1, klip: {path: p2, opts}})` (sekuensial) atau
`gc.batchAsync(items, { jobs: 4 })` (paralel, hasil urut = urutan input).

## Env

| Var | Arti |
|---|---|
| `GUARDCOMPRESS_BIN` | path binary manual |
| `GUARDCOMPRESS_VERSION` | versi binary (default `v0.1.0`) |
| `GUARDCOMPRESS_RELEASE_BASE` | base URL rilis (default GitHub Releases org) |
| `GUARDCOMPRESS_FFMPEG` | path ffmpeg manual |

Detail kontrak, config, dan keamanan: repo utama
[guardcompress](https://github.com/NTI-Nova-Teknologi-Inovatif/guardcompress)
(`docs/CONTRACT.md`, `docs/CONFIG.md`, `SECURITY.md`). Lisensi MIT.
