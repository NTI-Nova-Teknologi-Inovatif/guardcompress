# Atribusi Pihak Ketiga

GuardCompress (kode kami) berlisensi **MIT** (lihat `LICENSE`).

## FFmpeg (didistribusikan ulang sebagai static binary)

- Sumber: build independen (GyanD/codexffmpeg, BtbN/FFmpeg-Builds,
  johnvansickle.com, evermeet.cx) — BUKAN kode kami.
- WAJIB pakai varian build **LGPL** untuk rilis resmi (misal
  `*-essentials_build.zip`), JANGAN varian `full_build`/`gpl`
  kecuali siap mematuhi GPL penuh.
- Kewajiban saat redistribute build LGPL:
  1. Cantumkan atribusi + link sumber di halaman rilis.
  2. Sertakan teks lisensi LGPL (`COPYING.LGPLv2.1`) di arsip rilis.
  3. Sediakan link ke source ffmpeg yang sesuai versinya
     (lihat `docs/FFMPEG.md`).
- Binary ffmpeg TIDAK di-commit ke repo ini — diunduh maintainer
  saat menyiapkan rilis, lalu diunggah ke GitHub Releases + `CHECKSUMS.txt`.

## ClamAV (opsional, tidak dibundel)

Dipakai hanya bila SUDAH terinstal di server user (hook `clamdscan`).
Lisensi ikut paket ClamAV di mesin masing-masing (GPL). Kami tidak
mendistribusikannya.

## Go toolchain + stdlib (build core)

Lisensi BSD-style (golang.org/LICENSE). Static link stdlib tidak
menular ke lisensi kode kami.

## YARA rules bawaan (`core/rules/`)

Ditulis maintainer GuardCompress, ikut lisensi MIT repo ini.
