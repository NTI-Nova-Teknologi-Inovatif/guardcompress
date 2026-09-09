# ARCHITECTURE — GuardCompress

## 1. Prinsip
- **Embeddable:** dipanggil via `exec/spawn/subprocess`, tanpa daemon/Docker.
- **Polyglot:** 1 Core (Go) + N Thin Wrapper. Logika hanya di Core.
- **Zero-install:** FFmpeg static lazy-download, YARA embedded, fallback guard-only.
- **Kontrak stabil:** `check --in --out-dir --config --json`, exit `0/2/1`.

## 2. Data Flow (isolasi + deteksi berkelanjutan)
```
Upload tmp (terisolasi, tak pernah langsung ke storage)
  -> exec guardcompress check
    -> guard.Scan() di file isolasi
    -> nama reserved Windows? -> netralkan (file_CON)
    -> DITOLAK? -> karantina opsional (quarantine_dir, default: buang) + exit 2
    -> SVG? -> sanitasi allowlist (script/on*/DOCTYPE dibuang) -> lanjut
    -> compress.Run() -> sniff ULANG output (keluarga format harus sama)
    -> sidik sha256 output -> report.json + file bersih
  -> app simpan file + sha256 ke DB/S3, hapus tmp
... kapan saja setelahnya ...
  guardcompress verify --in <simpanan> --expect-sha256 <sidik>
    -> hash beda = file DIUBAH setelah lolos -> blocked
    -> scan ulang rules terbaru -> pola baru ketahuan -> blocked
```
Aturan: file yang lolos pun TERUS diawasi — setiap perubahan format/isi
terdeteksi saat verify berkala (cron/queue).

Deteksi berlapis di guard.Scan (berhenti di temuan pertama):
`nama eksekusi -> token stream (1MB chunk + overlap 4KB) ->
tag-pendek+payload (<?=/<% + keyword, min-run 12) -> chunk teks PNG
(zTXt/iTXt dibuka, cap 8MB) -> pixel flood (max_pixels) ->
container tertanam -> ClamAV (bila ada) -> VirusTotal (bila ada kunci)
-> allowlist MIME`.

## 2b. Struktur output (gampang dicek manual)
```
<out-dir>/
  video_liburan_anak.mp4   # nama ikut file asli, disanitasi; ext selalu dari MIME asli
  report.json              # laporan pretty-print (ada juga saat blocked, tanpa file media)
```
Opsi penamaan via `--config {"output": ...}`:
`original` (default) | `uuid` (acak 16 hex) | `"teks kustom"` (disanitasi).
Contoh: `evil.mp4.php` -> `evil_mp4_php.mp4`, `../../etc/passwd` -> `passwd.bin`.

## 3. Dependensi
| Komponen | Strategi v1 | Roadmap v2 |
|---|---|---|
| FFmpeg | static build per OS/arch, lazy-download ke `~/.cache/guardcompress`, SHA verify, `GUARDCOMPRESS_FFMPEG` | campur `go:embed` untuk 1 platform populer |
| Malware DB | heuristic bawaan + `rules/*.yar` sebagai spesifikasi cermin 1:1 | `yara-x` Go binding + `go:embed rules/` + auto-update via rilis binary |
| ClamAV | opsional hook `clamdscan` bila ada | - |
| VirusTotal | opsional cek hash SHA256 bila `virustotal_api_key` diisi | - |
| SVG | sanitizer Go stdlib sendiri (rujukan desain: DOMPurify/Cure53) | - |

## 4. Matrix rilis
`linux-amd64, linux-arm64, windows-amd64, darwin-arm64` via `.github/workflows/release.yml`.

## 5. Keamanan
- Jangan percaya extension; pakai magic numbers.
- Tanpa shell di semua wrapper (`proc_open` array / arg array).
- Timeout berlapis: ffmpeg core 100s < wrapper 120s (biar core yang menuai ffmpeg).
  File besar naikkan via `timeoutSec` + pindah ke queue (lihat `examples/`).
- Hapus tmp `in/out` setelah selesai; jangan log isi file.
- Report JSON berisi path server (`in_path`, `ffmpeg`) — jangan kirim
  mentah ke browser, kirim ringkasannya saja (`stored_at`, ukuran).
- Symlink input ditolak; ukuran + mtime dicek ulang setelah scan (TOCTOU).
- Output yang sama dengan input ditolak (biar CLI nggak overwrite diri sendiri).
- Config angka divalidasi (crf 0-51, bitrate `^\d+k$`, max_dim, quality).
- Tmp PHP pakai `random_bytes`, folder 0700.
- Yang belum ke-cover (catat): lirik MP3 berisi kata "eval(" ikut diblokir
  (fail-closed, nggak apa); CHECKSUMS rilis baru dilindungi TLS (mau cosign
  nanti); orphan ffmpeg kalau WRAPPER yang di-kill paksa di Windows
  (kalau core yang timeout aman; Linux aman via Pdeathsig).

## 6. Anti-down (server tidak boleh tumbang)

Ancaman: N user upload video bareng + ffmpeg rakus CPU/RAM/disk.
Rem berlapis (lihat `doctor`: `cpu_count`, `recommended_jobs`):

| Lapisan | Implementasi |
|---|---|
| CPU per job | `ffmpeg_threads` default **2** (1 upload tak menelan semua core) |
| Slot admission | semafor file lintas-proses (`internal/slots`, tanpa daemon): default = jumlah CPU, penuh → `busy` (HTTP 429). Lock yatim dibersihkan otomatis (15 mnt). `max_slots: 0` = tanpa batas |
| Waktu per job | timeout core 100s < wrapper 120s (hang = error, bukan gantung) |
| Ukuran per file | `max_mb` default 500 (lebih = blocked sebelum dibaca) |
| Pixel per gambar | `max_pixels` default 100MP (flood = blocked, tanpa alokasi pixel) |
| Paralel per request | `jobs` default 1; naikkan maks `recommended_jobs` (= CPU/2) |
| Paralel antar worker | contoh queue dibatasi (`concurrency: 2`, `--concurrency=2`, Horizon maxProcesses) |
| Video berat | wajib lewat queue (`examples/`), jangan di request HTTP langsung |
| Web server | `client_max_body_size` + rate-limit upload + FPM `max_children` wajar |
| Deploy check | jalankan `doctor` saat deploy; alert bila `ffmpeg_found=false` |
