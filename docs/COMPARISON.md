# Perbandingan — GuardCompress vs pendekatan lain

| Pendekatan | Kelebihan | Kelemahan | Kapan pakai |
|---|---|---|---|
| **Validasi ekstensi saja** | murah, cepat | buta isi: `shell.php` → `foto.png` lolos | jangan pernah sendirian |
| **Cek MIME dari framework** (`getMimeType`) | gampang | hanya baca header; webshell tempelan & polyglot lolos |Upload non-publik + file tepercaya |
| **ClamAV saja** | basis signature luas | butuh daemon + update DB; tidak kompres; tidak sanitasi nama; berat di shared hosting | server besar yang sudah ada ClamAV (GuardCompress bisa hook `clamdscan` sebagai lapis kedua) |
| **Moderasi cloud** (API pihak ketiga) | akurat + zero-ops | biaya per file, latensi, data user keluar server, lock-in | konten UGC publik skala besar |
| **GuardCompress** | cek isi + scan + kompres + sanitasi dalam 1 panggilan lokal; nol daemon; nol dependensi; gratis (MIT) | fail-closed (false positive mungkin); bukan AV penuh; butuh binary per OS/arch | form upload web umum (avatar, lampiran, media) yang butuh aman + hemat storage |

Intinya: GuardCompress menggantikan **lapisan validasi upload**, bukan
antivirus sistem. Kombinasi ideal untuk situs menengah: GuardCompress di
titik upload + ClamAV di server (hook `clamav: "auto"`) + backup/verify
berkala.

## Petakan ke checklist OWASP (jujur)

| Kontrol OWASP File Upload Cheat Sheet | Status kami |
|---|---|
| Allowlist ekstensi + validasi tipe (magic) | ✅ `allow_ext` + sniff |
| Nama acak dari aplikasi (`uuid`) | ✅ `output: "uuid"` |
| Batas ukuran + panjang nama + karakter | ✅ `max_mb`, 80 char, sanitasi |
| Simpan di luar webroot / tanpa eksekusi | ⚠️ tugas deployer (kami beri nama aman + `.bin` fallback) |
| Antivirus bila ada | ✅ hook ClamAV + VirusTotal (opsional) |
| CDR / image rewriting | ✅ re-encode FFmpeg + sanitasi SVG |
| Tolak ekstensi ganda eksekusi | ✅ aturan nama |
| Nama reserved Windows (`CON`/`NUL`...) | ✅ dinetralkan (`file_CON`) |
| Tolak symlink / cek TOCTOU | ✅ |
| Batas pixel flood (ASVS V5.2.6) | ✅ `max_pixels` (default 100MP) |
| Batas dekompresi (ASVS V5.2.3) | ✅ cap 8MB zTXt/iTXt |
| Autentikasi uploader / rate limit | ⚠️ tugas aplikasi (kami sediakan sinyal `BUSY`/429) |
