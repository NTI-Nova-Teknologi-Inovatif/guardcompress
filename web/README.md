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

## File uji (`web/test-files/`, lokal saja, tidak ikut commit)

- `asli.png` — foto asli -> harus BERSIH.
- File beracun dibuat lalu **dimakan Windows Defender duluan** (terbukti
  dari log Threat-nya: pola kita memang dikenali sebagai webshell).
  Supaya bisa uji penolakan tools kita di laptop ini, kecualikan dulu
  foldernya (butuh admin): Windows Security -> Virus & threat protection
  -> Manage settings -> Exclusions -> Add -> Folder ->
  `C:\tools\GuardCompress\web\test-files`. Habis itu bikin file beracunnya,
  drop di web, harusnya DITOLAK. Di server (tanpa AV) tidak perlu ini.

