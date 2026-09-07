# Web contoh uji GuardCompress (port 8080)

JS doang (Node stdlib, tanpa `npm install`, tanpa PHP). Cuma contoh buat
coba sistem, bukan buat produksi. Tools di `src/` milik folder ini,
nggak require kode utama — cuma butuh path binary core.

```powershell
# dari folder web/
$env:GUARDCOMPRESS_BIN = "C:\tools\GuardCompress\core\bin\guardcompress-windows-amd64.exe"
node server.js
# buka http://localhost:8080, drop file, hasil langsung muncul
```

- Bersih: tersimpan di `web/uploads/`, tampil + info hemat byte.
- Virus: DITOLAK + alasan. Server penuh: disuruh coba lagi (429).
- Aturan upload di `config.json`.
