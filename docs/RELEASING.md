# Panduan Maintainer — Rilis GuardCompress

## 1. Siapkan binary FFmpeg (sekali per update FFmpeg)

> Wajib varian **LGPL** (bukan `full`/`gpl`) + catat atribusi.
> Lihat `docs/FFMPEG.md` untuk daftar sumber upstream.

| OS | File upstream | Rename jadi |
|---|---|---|
| Linux amd64 | johnvansickle.com `ffmpeg-release-amd64-static.tar.xz` → `ffmpeg` | `ffmpeg-linux-amd64` |
| Linux arm64 | `ffmpeg-release-arm64-static.tar.xz` → `ffmpeg` | `ffmpeg-linux-arm64` |
| Windows amd64 | GyanD `*-essentials_build.zip` → `bin/ffmpeg.exe` | `ffmpeg-windows-amd64.exe` |
| macOS arm64 | evermeet.cx `ffmpeg` | `ffmpeg-darwin-arm64` |

Tambahkan `COPYING.LGPLv2.1` (unduh dari gnu.org) ke arsip rilis.

## 2. Potong rilis

```powershell
# pastikan master hijau (CI lolos), update CHANGELOG.md, lalu:
git tag vX.Y.Z master
git push origin master:main
git push origin vX.Y.Z
```

Workflow `release` otomatis: build 4 binary core → buat GitHub Release
berisi `guardcompress-*` + `CHECKSUMS.txt`.

## 3. Lengkapi rilis (manual, via web GitHub)

1. Buka halaman Release `vX.Y.Z` → Edit (ikon pensil).
2. Upload: 4 file `ffmpeg-*` + `COPYING.LGPLv2.1`.
3. **Hitung ulang CHECKSUMS**: unduh `CHECKSUMS.txt`, tambahkan baris
   sha256 untuk file ffmpeg (`sha256sum ffmpeg-*`), upload ulang
   (replace file).
4. Di deskripsi rilis, tempel atribusi: sumber tiap build ffmpeg +
   link source-nya (lihat `docs/THIRD-PARTY-NOTICES.md`).
5. Update. Installer wrapper langsung bisa memakai ffmpeg penuh.

## 4. Publish registry (per rilis, dari akun masing-masing)

```bash
# npm (akun npmjs.com)
cd wrappers/node && npm login && npm publish && cd ../..

# Packagist: packagist.org/packages/submit (sekali saja per repo);
# update berikutnya otomatis mengikuti tag GitHub.

# PyPI (akun pypi.org + API token)
cd wrappers/python && pip install build twine && python -m build && twine upload dist/* && cd ../..

# Go: tanpa publish — tag GitHub cukup untuk `go get`.
```

## 5. Sinkron repo bahasa

Otomatis via workflow `subtree-split` (secret `SPLIT_PAT`).
Cadangan manual ada di `docs/RELEASE.md`.
