# KONTRAK SINYAL — untuk developer pemakai

Prinsip: **sinyal tolak selalu eksplisit &Typed, fitur opsional selalu diam.**

## 1. Sinyal penolakan (dari isolasi)

| Kondisi | CLI exit | Wrapper |
|---|---|---|
| Bersih | `0` | return hasil normal |
| File berbahaya / berubah setelah lolos | `2` | **exception khusus**: PHP `InfectedFileException`, Node `e.code==='BLOCKED'`, Python `BlockedError`, Go `err "blocked: ..."` |
| Error teknis (file hilang, ffmpeg gagal, config rusak) | `1` | exception umum (`GuardException` / `Error` / `RuntimeError` / `error`) |

Contoh tangkap per bahasa ada di `core/main.go` (`init --lang=...`) dan `examples/`.
Selalu tangkap DUA-duanya: blocked → 422 ke user, error → 500 + alert.

## 2. Field report yang DIJAMIN ada vs OPSIONAL

Dijamin (boleh diandalkan di kode web):
`status`, `in_path`, `detected_mime`, `orig_bytes`, `reason` (saat gagal),
`out_path` + `new_bytes` (saat clean — wrapper melempar error jelas bila hilang).

Opsional (baca bila dipakai, abaikan bila tidak — tidak pernah error):
`sha256` (ada saat clean), `details.guard/compress/out_mime`, `took_ms`.

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
