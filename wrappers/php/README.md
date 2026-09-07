# GuardCompress PHP wrapper

```bash
composer require guardcompress/php
php bin/install-binary.php v0.1.0   # download binary yang cocok (sekali aja)
```

```php
use GuardCompress\GuardCompress;
try {
    $r = GuardCompress::process($request->file('video')->getRealPath(), ['max_mb'=>500,'video_crf'=>28]);
    Storage::putFile('media', new File($r->path));
} catch (\GuardCompress\InfectedFileException $e) {
    return response()->json(['blocked'=>$e->getMessage()], 422);
}
```

Env: `GUARDCOMPRESS_BIN` (override path binary), `GUARDCOMPRESS_FFMPEG` (path ffmpeg static).
