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
