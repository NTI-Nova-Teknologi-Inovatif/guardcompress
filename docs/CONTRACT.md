# KONTRAK SINYAL — untuk developer pemakai

Prinsip: **sinyal tolak selalu eksplisit &Typed, fitur opsional selalu diam.**

## 1. Sinyal penolakan (dari isolasi)

| Kondisi | CLI exit | Wrapper |
|---|---|---|
| Bersih | `0` | return hasil normal |
| File berbahaya / berubah setelah lolos | `2` | **exception khusus**: PHP `InfectedFileException`, Node `e.code==='BLOCKED'`, Python `BlockedError`, Go `err "blocked: ..."` |
| Server penuh (backpressure) | `1` + `details.busy` | **retry**: PHP `BusyException`, Node `e.code==='BUSY'`, Python `BusyError`, Go `IsBusy(err)` → balas HTTP **429**, bukan 422 |
| Error teknis (file hilang, ffmpeg gagal, config rusak) | `1` | exception umum (`GuardException` / `Error` / `RuntimeError` / `error`) |

Contoh tangkap per bahasa ada di `core/main.go` (`init --lang=...`) dan `examples/`.
Selalu tangkap DUA-duanya: blocked → 422 ke user, error → 500 + alert.

## 2. Field report yang DIJAMIN ada vs OPSIONAL

Dijamin (boleh diandalkan di kode web):
`status`, `in_path`, `detected_mime`, `orig_bytes`, `reason` (saat gagal),
`out_path` + `new_bytes` (saat clean — wrapper melempar error jelas bila hilang).

Opsional (baca bila dipakai, abaikan bila tidak — tidak pernah error):
`sha256` (ada saat clean), `thumbs` (daftar turunan bila diminta),
`details.guard/compress/out_mime`, `took_ms`.

## 3. Fitur opsional: tidak dipakai = diam total

| Fitur | Cara pakai | Bila tidak dipakai |
|---|---|---|
| `allow` | `{"allow": [...]}` | allowlist bawaan, diam |
| `output` | `"original" / "uuid" / "nama"` | `"original"`, diam |
| `max_mb`, `video_crf`, `image_quality`, ... | angka | default aman, diam |
| `timeoutSec` | detik | 100 core / 120 wrapper, diam |
| `quarantine_dir` | path | file jahat langsung dibuang, diam |
| `verify --expect-sha256` | hex | cek hash dilewati, diam |
| ffmpeg tidak ada | — | mode guard-only (copy) + catatan di report, **tanpa error** |

Pengecualian satu-satunya yang BERSUARA: binary core tidak ketemu
(`GUARDCOMPRESS_BIN` / installer) — ini fail-closed yang disengaja agar
aplikasi tidak jalan tanpa perlindungan secara diam-diam.

## 4. Format yang didukung + config web

Di web, developer cukup atur **extension familiar** — tools menerima semua
file yang bisa diperkecil:

```php
GuardCompress::process($file, ['allow_ext' => ['jpg','png','gif','mp4','mp3']]);
```

| Extension web | Terdeteksi (magic) | Output |
|---|---|---|
| `jpg jpeg png webp gif` | image/* | diperkecil (gif tetap animasi) |
| `mp4 mov webm mkv avi` | video/* (mov/mkv via ftyp/EBML) | H.264 + faststart |
| `mp3 wav ogg oga m4a flac` | audio/* (brand ftyp M4A, magic fLaC) | mp3 / ogg / m4a hemat |
| lain (default) | — | **ditolak** (`mime not allowed`) |

Catatan: `wav/flac` besar otomatis jadi `mp3`; `mkv` keluar sebagai `.webm`
(keluarga EBML sama, ramah browser); `M4A` tidak pernah dianggap video
(brand ftyp dibedakan). Bila `allow` atau `allow_ext` diisi, ia MENGGANTI
default (keduanya diisi = gabungan); extension tak dikenal = error developer
yang jelas, bukan blocked.

Preset siap pakai (satu sistem di belakangnya, argumen user menang):
`GuardCompress::image($p)` / `::video($p)` / `::audio($p)` (PHP),
`gc.image/video/audio` (Node), `image()/video()/audio()` (Python & Go).

Batch multi-input beda jenis sekaligus (form ada field PNG + video):
```php
$h = GuardCompress::batch(['avatar' => $p1, 'klip' => ['path' => $p2, 'opts' => ['allow_ext' => ['mp4']]]]);
// $h['avatar'] = ['ok'=>true,'result'=>...], $h['klip'] = ['ok'=>false,'blocked'=>true,...]
```
(Node: `gc.batch({...})`, Python: `batch({...})`, Go: `Batch([]BatchItem{...})`.)
File ditolak terkumpul per item (tidak melempar); error teknis tetap
melempar langsung. Aturan per item bisa beda (preset/opts sendiri) atau
digabung dalam satu panggilan.

Paralelisme dua tingkat:
- **Antar-user (request bersamaan): selalu paralel.** Tiap request = proses
  CLI sendiri tanpa lock/antrean di tools kita. Batasnya hanya CPU untuk
  ffmpeg — video berat disarankan lewat queue (`examples/`).
- **Dalam satu batch: sekuensial default, paralel bila diminta** via
  `jobs`: PHP `batchParallel($items, ['jobs' => 4])`, Node
  `batchAsync(items, {jobs})` (default CPU, maks 4), Python/Go
  `batch(..., {"jobs": N})`. Urutan hasil selalu = urutan input.
  Terbukti: 2 video jobs:2 = 2.8s vs sekuensial 3.8s (skala ikut inti CPU).
