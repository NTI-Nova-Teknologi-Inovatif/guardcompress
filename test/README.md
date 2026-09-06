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

## Uji adversarial (`adversarial/`, 14 teknik serangan inert)

Bukan malware aktif: pola jinak berlabel yang meniru teknik penyerang.
Generator (`gen-adversarial.py`) merakit pola dari potongan string agar
file generatornya sendiri tidak dikarantina antivirus.

```powershell
python test/gen-adversarial.py      # generate ulang 14 file + expected.json
powershell -File test/run-adversarial.ps1   # verdict vs harapan (exit 0 = semua PASS)
```

Kasus: webshell ekor-PNG, varian HURUF BESAR, obfuscation gzinflate,
ASP tags, XSS script, UTF-16LE, payload di batas chunk 1MB & tengah 5MB,
padding biner, double-ext `.phtml`, string uji AV, 2 kontrol bersih,
`cmd.exe`, script mentah. Hasil terakhir: **14 PASS, 0 FAIL**.
