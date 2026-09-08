# Changelog — GuardCompress

Format mengikuti [Keep a Changelog](https://keepachangelog.com/id/1.0.0/).

## [Unreleased]

## [0.1.0] — 2026-09-08 (rencana rilis pertama)

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
