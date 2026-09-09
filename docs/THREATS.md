# Model Ancaman — apa yang ditangkap GuardCompress

Setiap baris di bawah **terbukti tertangkap** lewat uji nyata
(CLI + regression test `core/internal/guard/guard_test.go`).

## Webshell dalam gambar

| Kasus | Contoh | Status |
|---|---|---|
| PHP di ekor file | `foto.png` + `<?php system($_GET["c"]); ?>` | tolak (token) |
| Short tag di segmen COM JPEG | `<?=`$_GET[x]`?>` dalam komentar EXIF/COM | tolak (tag+payload) |
| ASP di ekor file | `<%eval(request("x"))%>` | tolak (tag+payload) |
| Shell di chunk teks PNG | `tEXt` berisi `<?php ...` | tolak (token) |
| Shell di chunk teks PNG **terkompres** | `zTXt`/`iTXt` berisi webshell (zlib) | tolak (dekompres+pindai) |
| EVAL huruf-besar | `EVAL(BASE64_DECODE("eA=="));` | tolak (case-insensitive) |
| Obfuskasi | `gzinflate(str_rot13(...))`, `create_function` | tolak |
| Marker shell legendaris | `c99shell`, `r57shell` | tolak |
| Perintah OS | `cmd.exe /c ...`, `/bin/sh -c ...`, `shell_exec`, `passthru`, `popen`, `proc_open` | tolak |
| UTF-16 | `<?php` dalam UTF-16LE/BE | tolak |
| Test standar | string EICAR | tolak |

## XSS tersimpan

| Kasus | Contoh | Status |
|---|---|---|
| Script tag campur huruf | `<ScRiPt>alert(1)</ScRiPt>` | tolak (case-insensitive) |
| SVG berisi `<script>` | `<svg><script>alert(1)</script></svg>` | tolak |
| SVG berisi event handler | `<rect onload="x()">` | **sanitasi** (atribut dibuang, bentuk utuh) |

## Penipuan nama & format

| Kasus | Contoh | Status |
|---|---|---|
| Ekstensi ganda eksekusi | `foto.jpg.php`, `x.mp4.exe` | tolak (aturan nama) |
| Ekstensi vs isi beda | `shell.php` rename `foto.png` (isi PHP) | tolak (MIME tak dikenal/tak diizinkan) |
| Arsip tertanam | ZIP di dalam file media | tolak (container scan, default on) |
| Path traversal | `../../etc/passwd` sebagai nama | disanitasi → `passwd.bin` |
| Symlink | input berupa symlink | tolak |
| Nama reserved Windows | `CON.jpg`, `NUL.png`, `COM1.mp4` | dinetralkan (`file_CON.jpg`) |
| Pixel flood | PNG 262144×262144 | tolak (`max_pixels`, default 100MP) |
| File berubah setelah scan (TOCTOU) | ukuran/mtime berubah di tengah jalan | batal + error |

## Yang SENGAJA diloloskan (bukan ancaman)

| Kasus | Alasan |
|---|---|
| Metadata/teks jinak (EXIF, `tEXt` biasa) | semua foto asli membawanya |
| Deklarasi `<?xml ...?>` | bukan payload (tanpa keyword bahaya) |
| Kebetulan biner (`<%` 2 byte di data acak) | run terlalu pendek + tanpa payload |

## Yang TIDAK ditangani

- Steganografi murni (data disembunyikan tanpa pola serangan) — bukan
  serangan dengan sendirinya; tidak ada pemindai upload yang menanganinya.
- Malware zero-day tanpa pola dikenal — mitigasi: `verify` berkala +
  hook ClamAV (`clamav: "auto"`) sebagai lapis kedua.
- Eksekusi di server korban akibat salah konfigurasi server itu sendiri
  (misal `AddHandler php` untuk `.jpg`) — perbaiki konfigurasi server.
