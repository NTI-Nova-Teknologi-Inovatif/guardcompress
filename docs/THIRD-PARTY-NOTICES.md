# Atribusi Pihak Ketiga

GuardCompress (kode kami) berlisensi **MIT** (lihat `LICENSE`).

## FFmpeg (binary, TIDAK dikomit ke repo)

- Build diunduh maintainer saat menyiapkan rilis, lalu diunggah ke GitHub Releases + `CHECKSUMS.txt`.
- Varian **GPL** (wajib — pipeline kami butuh encoder `libx264` yang
  memang GPL). Kode GuardCompress tetap MIT (binary terpisah, tanpa link).
- Kewajiban redistribute:
  1. Cantumkan atribusi + link sumber di halaman rilis.
  2. Sertakan teks lisensi GPL (`COPYING.GPLv3`) di arsip rilis.
  3. Sediakan link source ffmpeg yang sesuai versinya
     (lihat `docs/FFMPEG.md`).
- Sumber build independen: BtbN/FFmpeg-Builds (linux), GyanD/codexffmpeg
  (windows), evermeet.cx (macOS, build Intel → Rosetta 2 di Apple Silicon).

## ClamAV (opsional, tidak dibundel)

Hanya dipakai bila SUDAH terinstal di server user (`clamdscan`). Lisensi ikut paket ClamAV di mesin masing-masing (GPL). Kami tidak mendistribusikannya.

## Go toolchain + stdlib (build core)

Lisensi BSD-style (golang.org/LICENSE). Static link stdlib tidak menular ke lisensi kode kami.

## Dependensi lain

**Tidak ada.** Semua wrapper (Node/PHP/Python/Go) dan core Go hanya pakai stdlib — nol dependensi npm/composer/pip pihak ketiga.
