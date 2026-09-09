# Changelog — GuardCompress

Format mengikuti [Keep a Changelog](https://keepachangelog.com/id/1.0.0/).

## [Unreleased]

### Ditambahkan
- Sanitizer SVG opt-in (`allow_ext: ["svg"]`): allowlist elemen/atribut,
  buang script/event-handler/DOCTYPE/komentar; XML malformed ditolak.
- Batas pixel flood `max_pixels` (default 100MP) — jpeg/png/gif/webp.
- Netralisasi nama reserved Windows (`CON`/`NUL`/`COM1`... → `file_CON`).
- Referensi standar: `docs/REFERENCES.md` + petakan OWASP di COMPARISON.

### Diubah
- Wrapper dipecah jadi struktur SDK multi-file (node/php/python/go),
  API tetap sama. Exception PHP satu class satu file (PSR-4 penuh).

## [0.1.1] — 2026-09-09

### Ditambahkan
- Webshell short-tag + payload (tutup lolos shell segmen COM JPEG).
- Pindai isi chunk teks PNG terkompres (zTXt/iTXt).
- Hook VirusTotal opsional (reputasi hash).
- Wrapper SDK multi-file; demo web pertahankan nama asli di tmp.

### Diperbaiki
- CI rilis: izin publish + build dari folder core.
- Subtree split: 1 secret `SPLIT_PAT` (matrix) + full history.
- Struktur dokumen: kebijakan ke `.github/`, notices ke `docs/`.

## [0.1.0] — 2026-09-09 (rilis pertama, GitHub saja)

Binary inti 4 platform + CHECKSUMS di GitHub Releases. Registry npm /
Packagist / PyPI belum. FFmpeg belum dibundel (mode guard-only).

### Ditambahkan
- Core Go: `check | verify | doctor | init`, exit `0/2/1` + `report.json`.
- Guard: sniff magic numbers, tolak spoofing ekstensi, pindai webshell/
  skrip/EICAR, sanitasi nama, karantina opsional, hook ClamAV.
- Compress: FFmpeg static lazy-download + verifikasi SHA, preset
  image/video/audio, thumbs, salinan webp.
- Wrapper PHP (Composer), Node (npm), Python (PyPI), Go — stdlib-only.
- Contoh queue: Laravel job, BullMQ worker, Celery task.
- Demo web Node stdlib (`web/`).
- Dokumen: ARCHITECTURE, CONFIG, CONTRACT, FFMPEG, RELEASE, FAQ,
  COMPARISON, SECURITY, CONTRIBUTING, THIRD-PARTY-NOTICES.
- CI: build 4 platform + subtree split ke 4 repo bahasa.
