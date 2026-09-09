# Glosarium — apa itu apa di GuardCompress

Bahasa manusia untuk istilah yang muncul di repo ini.

| Istilah | Artinya |
|---|---|
| **Monorepo** | Satu repo berisi semua kode (inti + 4 wrapper + docs). Ngoding di sini. |
| **Repo bahasa** | Repo kecil hasil salinan otomatis 1 folder wrapper (misal `guardcompress-js` = isi `wrappers/node/`). Untuk user yang cuma butuh 1 bahasa + untuk publish ke registry. |
| **Wrapper** | Penerjemah: kode PHP/JS/Python/Go yang memanggil binary inti lalu menerjemahkan hasilnya jadi return/exception enak. Tidak ada logika scan di sini. |
| **Binary inti / core** | Program Go (`guardcompress-linux-amd64` dsb) berisi SEMUA logika: scan + kompres. Satu file, dijalankan via `exec`. |
| **FFmpeg** | Mesin kompres video/gambar/audio (proyek open-source). Diunduh otomatis sebagai file static, bukan diinstall manual. |
| **Registry** | Toko paket per bahasa: npm (JS), Packagist (PHP), PyPI (Python). Go tidak pakai registry — langsung `go get` dari GitHub. Status: GitHub ✅, registry ⏳ (segera). |
| **Postinstall / installer** | Script yang jalan otomatis (atau sekali manual) setelah install wrapper: mengunduh binary + ffmpeg yang cocok OS/arch + verifikasi SHA. Gagal unduh = mode guard-only, tidak menggagalkan install. |
| **report.json** | Laporan tiap pemindaian: status (clean/blocked), MIME terdeteksi, ukuran, sha256, alasan. Berisi path server — jangan kirim mentah ke browser. |
| **Exit code** | Sinyal binary: `0` bersih, `2` diblokir, `1` error/sibuk. Wrapper menerjemahkannya jadi exception per bahasa. |
| **Fail-closed** | Prinsip: ragu sedikit = tolak. False positive diterima, file jahat lolos tidak. |
| **Subtree split** | Mekanisme GitHub Actions: tiap push tag di repo full, folder wrapper didorong otomatis ke repo bahasanya. Tanpa copy manual. |
| **Quarantine** | Folder opsional tempat file jahat disimpan untuk forensik. Default: langsung dibuang. |
| **Verify** | Perintah memastikan file simpanan tidak berubah setelah lolos (`verify --expect-sha256`). |
| **Doctor** | Perintah cek kapasitas mesin (CPU, ffmpeg ada/tidak, jobs rekomendasi). |
| **YARA rules** | Pola deteksi malware (`core/rules/`) — dipakai penuh di v2. |
| **LGPL** | Lisensi FFmpeg: boleh redistribute binary asal atribusi + sertakan teks lisensi + link source. Lihat `docs/THIRD-PARTY-NOTICES.md`. |
| **Chunk PNG** | Potongan data dalam PNG (`IHDR`, `IDAT`, `tEXt`, `zTXt`...). `zTXt`/`iTXt` boleh terkompres — isinya dibuka + dipindai. |
| **Segmen COM** | Kolom komentar dalam JPEG — tempat favorit menyembunyikan webshell. Dipindai. |
| **Polyglot** | Satu file yang valid sebagai dua jenis (misal JPEG sekaligus PHP). Ditolak via token/nama. |
