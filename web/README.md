# Web demo GuardCompress (port 8080)

Folder mandiri: UI + tools salinan di `src/` (tidak require kode utama,
cuma butuh path binary core). Aturan upload di `config.php`.

```powershell
# dari root repo
$env:GUARDCOMPRESS_BIN = "C:\tools\GuardCompress\core\bin\guardcompress-windows-amd64.exe"
cd web/public
php -S localhost:8080
# buka http://localhost:8080, drop file, hasil langsung muncul
```

- Bersih: tersimpan di `web/uploads/`, tampil + info hemat byte.
- Virus: DITOLAK + alasan. Server penuh: disuruh coba lagi (429).
