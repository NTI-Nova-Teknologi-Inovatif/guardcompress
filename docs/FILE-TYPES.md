# Jenis File — matriks lengkap GuardCompress

Cara baca: **Ekstensi** = yang boleh ditulis user di `allow_ext`;
**Magic** = sidik byte yang benar-benar diperiksa; **Output** = hasil
setelah kompres; ekstensi output SELALU dari MIME asli, bukan nama file.

## Gambar

| Ekstensi | Magic (byte awal) | Output | Catatan |
|---|---|---|---|
| `jpg`, `jpeg` | `FF D8 FF` | `.jpg` (quality terkontrol) | segmen COM/EXIF ikut dipindai |
| `png` | `89 50 4E 47 0D 0A 1A 0A` | `.png` (lossless max) | chunk teks (tEXt/zTXt/iTXt) dibuka + dipindai isinya |
| `webp` | `RIFF....WEBP` | `.webp` | — |
| `gif` | `GIF87a` / `GIF89a` | `.gif` (animasi utuh) | — |

## Video

| Ekstensi | Magic | Output | Catatan |
|---|---|---|---|
| `mp4` | `ftyp` (brand mp4/isom) | `.mp4` H.264 + faststart | — |
| `mov` | `ftyp` (brand qt) | `.mp4` | dikonversi agar ramah browser |
| `webm` | `1A 45 DF A3` (EBML) | `.webm` | — |
| `mkv` | `1A 45 DF A3` (EBML) | `.webm` | keluarga sama, keluar webm |
| `avi` | `RIFF....AVI` | `.mp4` | dikonversi |

## Audio

| Ekstensi | Magic | Output | Catatan |
|---|---|---|---|
| `mp3` | `ID3` / frame `FF Ex` | `.mp3` | — |
| `wav` | `RIFF....WAVE` | `.mp3` | besar → hemat |
| `ogg`, `oga` | `OggS` | `.ogg` | — |
| `m4a` | `ftyp` (brand M4A) | `.m4a` | tidak pernah dianggap video (brand dibedakan) |
| `flac` | `fLaC` | `.mp3` | besar → hemat |

## Selain itu = DITOLAK

`mime not allowed`. Termasuk: `.php .phtml .asp .aspx .jsp .exe .sh
.bat .ps1 .py .pl .cgi .htaccess` (nama mengandung ini = tolak duluan
via aturan nama), serta zip/rar/7z/pdf/doc (arsip & dokumen bukan media).

## Aturan nama file

- Ekstensi ganda eksekusi (`foto.jpg.php`, `x.mp4.exe`) = **tolak**,
  walau isinya bersih. Bahaya ada di namanya (server bisa eksekusi).
- Nama dibersihkan: `evil.mp4.php` → `evil_mp4_php.mp4`,
  `../../etc/passwd` → `passwd.bin`.
- Opsi penamaan: `output: "original"` (default) | `"uuid"` | `"teks kustom"`.

## Data tersembunyi: kapan lolos, kapan tolak?

- Teks/metadata jinak (EXIF, komentar, `tEXt` biasa) = **lolos**. Benar
  begitu: hampir semua foto asli membawa metadata.
- Pola serangan di mana saja (termasuk segmen COM JPEG, chunk teks PNG
  biasa **maupun terkompres** zTXt/iTXt) = **tolak**. Lihat
  `docs/THREATS.md` untuk daftar kasus yang terbukti tertangkap.
