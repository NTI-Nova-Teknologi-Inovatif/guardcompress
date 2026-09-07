# Atribusi Pihak Ketiga

GuardCompress (kode kami) berlisensi **MIT** (lihat `LICENSE`).

## FFmpeg (binary, TIDAK dikomit ke repo)

- Build diunduh maintainer saat menyiapkan rilis, lalu diunggah ke GitHub Releases + `CHECKSUMS.txt`.
- WAJIB varian **LGPL** (misal `*-essentials_build.zip`). Jangan varian `full_build`/`gpl` kecuali siap mematuhi GPL penuh.
- Kewajiban redistribute:
  1. Cantumkan atribusi + link sumber di halaman rilis.
  2. Sertakan teks lisensi LGPL (`COPYING.LGPLv2.1`) di arsip rilis.
  3. Sediakan link source ffmpeg sesuai versinya (lihat `docs/FFMPEG.md`).
- Sumber build independen: GyanD/codexffmpeg, BtbN/FFmpeg-Builds, johnvansickle.com, evermeet.cx.

## ClamAV (opsional, tidak dibundel)

Hanya dipakai bila SUDAH terinstal di server user (`clamdscan`). Lisensi ikut paket ClamAV di mesin masing-masing (GPL). Kami tidak mendistribusikannya.

## Go toolchain + stdlib (build core)

Lisensi BSD-style (golang.org/LICENSE). Static link stdlib tidak menular ke lisensi kode kami.

## Dependensi lain

**Tidak ada.** Semua wrapper (Node/PHP/Python/Go) dan core Go hanya pakai stdlib — nol dependensi npm/composer/pip pihak ketiga.
