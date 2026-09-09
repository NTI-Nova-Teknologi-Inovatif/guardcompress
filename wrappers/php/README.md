# GuardCompress for PHP

Keamanan + kompresi upload untuk Laravel/WordPress. Thin wrapper di atas
binary inti Go — tanpa dependensi composer.

## Apa itu ini?

- **Wrapper** = class PHP (`GuardCompress::process()`) yang memanggil
  binary Go lewat `proc_open`, lalu menerjemahkan hasilnya jadi return
  atau exception (`InfectedFileException` = 422, `BusyException` = 429).
- **Binary inti** = program Go yang berisi SEMUA logika (scan + kompres).
- **FFmpeg** = mesin kompres, diunduh otomatis oleh installer.

## Instalasi

> **Hanya via GitHub untuk saat ini** — belum ada di Packagist.

```bash
composer require guardcompress/php:dev-main \
  --repository='{"type":"vcs","url":"https://github.com/NTI-Nova-Teknologi-Inovatif/guardcompress-php"}'
php bin/install-binary.php v0.1.0   # unduh binary yang cocok (sekali saja; ffmpeg menyusul)
```

## Pakai

```php
use GuardCompress\GuardCompress;
try {
    $r = GuardCompress::process($request->file('video')->getRealPath(), ['max_mb'=>500,'video_crf'=>28]);
    Storage::putFile('media', new File($r->path));
} catch (\GuardCompress\InfectedFileException $e) {
    return response()->json(['blocked'=>$e->getMessage()], 422);
}
```

Shortcut: `GuardCompress::image($p)`, `::video($p)`, `::audio($p)`.
Batch: `GuardCompress::batch(['avatar' => $p1, 'klip' => ['path' => $p2]])`.

Env: `GUARDCOMPRESS_BIN` (override path binary), `GUARDCOMPRESS_FFMPEG`
(path ffmpeg static).

Detail kontrak, config, dan keamanan: repo utama
[guardcompress](https://github.com/NTI-Nova-Teknologi-Inovatif/guardcompress).
Lisensi MIT.
