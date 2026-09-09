# FAQ — GuardCompress

## Umum

**Apa itu GuardCompress?**
Middleware upload: setiap file diperiksa isinya (bukan ekstensinya),
dipindai pola berbahaya, lalu media dikompresi ulang via FFmpeg. Bersih =
disimpan, jahat = ditolak dengan alasan.

**Bahasa apa saja yang didukung?**
PHP (Laravel/WordPress), Node.js, Python (Django/Flask), Go. Logika hanya
ada di satu binary Go; wrapper hanya pemanggil tipis.

**Apakah menggantikan antivirus server?**
Tidak. Ini satu lapis pertahanan di titik upload. Tetap pakai AV,
firewall, dan hardening. Lihat `SECURITY.md`.

**Apakah data saya dikirim ke mana-mana?**
Tidak — kecuali kamu NYALAKAN hook VirusTotal (`virustotal_api_key`).
Itu pun yang keluar hanya SHA256 file (untuk tanya reputasi hash),
bukan isi file. Tanpa kunci API: 100% lokal, tanpa telemetri.

## Instalasi

**Binary/ffmpeg dari mana?**
Otomatis dari GitHub Releases saat instalasi wrapper (postinstall /
`install-binary.php` / `python -m guardcompress.install`), diverifikasi
SHA256 terhadap `CHECKSUMS.txt`. Detail: `docs/FFMPEG.md`.

**Offline / server tanpa internet?**
Unduh binary + ffmpeg dari halaman Releases di mesin lain, taruh
bersebelahan, set `GUARDCOMPRESS_BIN` + `GUARDCOMPRESS_FFMPEG`.

**Tanpa ffmpeg bisa jalan?**
Bisa — mode guard-only: cek keamanan tetap jalan, kompresi jadi copy +
catatan di report. Tanpa error.

## Perilaku

**File bersih tapi ditolak, kenapa?**
Fail-closed disengaja. Penyebab umum: ekstensi tidak ada di allowlist,
MIME tidak cocok ekstensi, ukuran > `max_mb`, atau konten mengandung pola
mencurigakan (misal lirik MP3 berisi kata `eval(`). Longgarkan via
`allow_ext`/`max_mb`, atau laporkan bila salah.

**Bedanya 422 vs 429?**
422 = file bermasalah (jangan retry file yang sama). 429 = server penuh
(`BUSY`, backpressure) — retry dengan jeda. Jangan tertukar.

**File besar / video panjang?**
Jangan di request HTTP langsung — lempar ke queue (`examples/`) dengan
`timeoutSec` besar (misal 600). Cek kapasitas: `guardcompress doctor`.

**Bagaimana memastikan file simpanan tidak diubah orang?**
`guardcompress verify --in <file> --expect-sha256 <sidik>` — hash beda =
berubah setelah lolos = blocked. Jalankan berkala via cron/queue.

**File PNG/JPG murni tapi ada data tersembunyi kok lolos?**
Tergantung datanya. Metadata/teks jinak (EXIF, komentar, `tEXt` biasa)
memang diloloskan — semua foto asli membawanya, memblokirnya berarti
memblokir internet. Yang ditolak: pola serangan di mana saja (webshell
di ekor file, segmen COM JPEG, chunk teks PNG biasa maupun terkompres
zTXt/iTXt, ekstensi ganda eksekusi). Daftar lengkap yang terbukti
tertangkap: `docs/THREATS.md`.

## Lisensi & rilis

**Boleh dipakai komersial?**
Boleh — MIT. Lihat `LICENSE` dan `THIRD-PARTY-NOTICES.md` (khusus
redistribusi binary FFmpeg: wajib varian LGPL + atribusi).

**Di mana lapor bug / celah?**
Bug biasa: issue GitHub. Celah keamanan: privat ke maintainer
(`SECURITY.md`) — jangan issue publik.
