# Test GuardCompress

## Fixture (`fixtures/`, dibuat via `python gen-fixtures.py`)

| File | Isi | Hasil harapan |
|---|---|---|
| `clean.png` | PNG valid 8x8 | `clean` |
| `evil-php.png` | PNG valid + `<?php system(...)` | `blocked` (token `<?php`) |
| `evil-eicar.png` | PNG valid + string EICAR | `blocked` |

Cek cepat (dari root repo):
```powershell
$bin = "./core/bin/guardcompress-windows-amd64.exe"
& $bin check --in test/fixtures/clean.png --out-dir test/fixtures/out-clean --json
& $bin check --in test/fixtures/evil-php.png --out-dir test/fixtures/out-evil --json
```

## Web uji coba (`web/`)

```powershell
$env:GUARDCOMPRESS_BIN = "C:\tools\GuardCompress\core\bin\guardcompress-windows-amd64.exe"
cd test/web; php -S localhost:8080
# buka http://localhost:8080, upload clean.png vs evil-*.png
```

Hanya untuk test lokal. Hasil upload tersimpan di `web/uploads/` (di-ignore git).
