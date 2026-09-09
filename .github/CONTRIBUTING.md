# Kontribusi

1. Fork repo full, bikin branch (`fitur/apa` / `fix/apa`).
2. Untuk core Go: pastikan `gofmt -l` bersih + `go vet ./...` lolos.
3. Untuk wrapper: sesuaikan SEMUA bahasa bila ubah kontrak (jangan PHP doang).
4. Jangan ubah kontrak CLI/report tanpa diskusi dulu (lihat `../docs/CONTRACT.md`).
5. Jangan commit binary, file upload, atau hasil scan (`core/bin/`, `web/uploads/`).
6. Buka PR dengan deskripsi: masalahnya apa, diubah apa, hasil ujinya apa.
7. **Celah keamanan JANGAN via PR/issue publik** — lihat `SECURITY.md`.

Lisensi kontribusi = MIT (sama dengan repo). Dengan berkontribusi kamu
setuju karyamu dirilis di bawah MIT.
