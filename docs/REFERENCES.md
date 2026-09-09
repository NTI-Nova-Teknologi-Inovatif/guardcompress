# Referensi — standar & tool publik yang dipakai GuardCompress

Keputusan desain kami merujuk sumber-sumber ini (bukan klaim kosong).

## Standar

- **OWASP File Upload Cheat Sheet** (`cheatsheetseries.owasp.org`) —
  allowlist ekstensi, validasi tipe file (bukan header), nama acak,
  batas ukuran, simpan di luar webroot, antivirus bila ada, CDR
  (Content Disarm & Reconstruct) untuk tipe yang cocok. Pemetaan
  kontrol-ke-fitur ada di `docs/COMPARISON.md`.
- **OWASP ASVS 5.0 §V5.2 (File Upload and Content)** — `max_mb` (V5.2.1),
  magic bytes + image re-writing (V5.2.2), batas dekompresi arsip (V5.2.3,
  kami: cap 8MB zTXt/iTXt), tolak symlink (V5.2.5), tolak pixel flood
  (V5.2.6, kami: `max_pixels`).
- **SANS — 8 Basic Rules for Secure File Uploads (2025)** — nama acak,
  simpan di luar document root, ekstensi tak bermakna, rebuild file
  (JPEG→GIF→JPEG ≈ re-encode FFmpeg kami), batas jumlah/ukuran.
- **OWASP Unrestricted File Upload** — nama reserved Windows
  (`CON`/`NUL`/`COM1`...), traversal, null byte, karakter khusus:
  semuanya disanitasi/dinetralkan di `naming.go`.

## Tool publik

- **ClamAV** (`clamav/clamav`) — hook opsional `clamdscan`/`clamscan`
  (`clamav: "auto"`), plus contoh cek silang Docker `examples/clamav.yml`.
- **VirusTotal API v3** — cek reputasi hash SHA256 (isi file tak pernah
  keluar). Opsional via `virustotal_api_key`.
- **DOMPurify (Cure53, 17k+ stars)** — rujukan desain sanitizer SVG kami.
  Mereka butuh jsdom di server; kami menulis sanitizer Go stdlib sendiri
  (allowlist elemen/atribut, buang `on*`/`javascript:`/DOCTYPE/komentar)
  agar tetap nol-dependensi. Perbandingan jujur ada di `docs/COMPARISON.md`.
- **FFmpeg** — mesin CDR/re-encode (build LGPL independen, lihat
  `docs/FFMPEG.md`).
