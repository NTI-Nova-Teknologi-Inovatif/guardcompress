# KONFIGURASI — semua kenop untuk developer

Semua knob lewat SATU tempat: argumen `opts` di fungsi wrapper
(PHP/Node/Python/Go), diteruskan mentah sebagai `--config` JSON ke core.
Tak diisi = default aman (diam, tidak error).

## Keamanan & format

| Knob | Default | Arti |
|---|---|---|
| `allow_ext` | bawaan (jpg png webp gif mp4 mp3 ...) | extension yang diterima; ganti default bila diisi |
| `allow` | bawaan (MIME) | versi MIME; gabung dengan `allow_ext` bila keduanya diisi |
| `max_mb` | `500` | tolak file lebih besar sebelum dibaca |
| `output` | `"original"` | `"uuid"` acak / `"teks kustom"` |

## Anti-down (developer yang atur sesuai servernya)

| Knob | Default | Kapan diubah |
|---|---|---|
| `max_slots` | jumlah CPU (`0` = tanpa batas) | server kecil turunkan (cth `2`); server media naikkan/`0` |
| `ffmpeg_threads` | `2` (1–32) | VPS besar + queue: `4`; shared hosting: `1` |
| `timeoutSec` / `timeout_sec` | 100 core / 120 wrapper | video panjang + queue: `600` |
| `min_compress_kb` | `0` (selalu kompres) | situs avatar: `100` — file kecil langsung copy byte-identik (hemat CPU, tanpa artefak rekompresi) |
| `thumb_widths` | tak ada | `[300, 800]` atau `[{"w":300,"suffix":"-sm"}]` — turunan ukuran sekali jalan (tanpa upscale; gagal turunan tak gugurkan hasil utama) |
| `webp` | `false` | `true` — salinan `.webp` untuk output utama + tiap thumb |
| `jobs` (batch) | `1` | batch paralel: maks `recommended_jobs` (lihat `doctor`) |
| `quarantine_dir` | `""` (buang) | isi path hanya bila butuh forensik |
| `GUARDCOMPRESS_CACHE` (env) | `~/.cache/guardcompress` | container/serverless: arahkan ke volume writable |
| `GUARDCOMPRESS_BIN` / `GUARDCOMPRESS_FFMPEG` (env) | auto | path manual bila installer tak dipakai |

Contoh server kecil (2 CPU, shared):
```php
['max_mb' => 100, 'ffmpeg_threads' => 1, 'max_slots' => 2, 'timeoutSec' => 120]
```
Contoh server media (8 CPU + queue):
```php
['max_mb' => 500, 'ffmpeg_threads' => 4, 'max_slots' => 4, 'timeoutSec' => 600, 'jobs' => 4]
```

Cek kapasitas kapan saja: `guardcompress doctor` (`cpu_count`, `recommended_jobs`).
Sinyal & perilaku tiap kondisi: lihat `docs/CONTRACT.md`.
