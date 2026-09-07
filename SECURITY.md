# Kebijakan Keamanan — GuardCompress

## Yang dilindungi

GuardCompress adalah **lapisan pertahanan upload**, bukan pengganti
antivirus enterprise, firewall, audit pentest, atau hardening server.
Pakai sebagai satu lapis dari pertahanan berlapis.

## Lapor celah (responsible disclosure)

Jangan buka issue publik untuk celah keamanan. Kirim tertutup ke maintainer:

- Subjek: `[SECURITY] ringkasan singkat`
- Sertakan: versi, langkah reproduksi minimal, dampak yang dinilai.
- Kami targetkan respons awal **3x24 jam** dan perbaikan **14 hari**
  untuk celah kritis, lalu publikasi + kredit reporter (bila mau).

Peneliti yang lapor dengan itikad baik dan tidak merusak/mencuri data
tidak akan dituntut (safe harbor).

## Versi yang didukung

Hanya rilis stabil terakhir (`v0.x` terbaru). Versi lama tidak di-patch —
upgrade.

## Yang di luar cakupan

- Bug yang butuh akses root/server korban duluan.
- Kelemahan ffmpeg/ClamAV upstream (laporkan ke mereka, kami update build).
- Social engineering, phishing, kredensial bocor.
- Klaim "lolos 100% malware" — tidak ada scanner yang bisa janji itu,
  termasuk kami. Lihat batas yang diakui di `docs/ARCHITECTURE.md` §5–§6.

## Privasi

Tools berjalan **100% lokal** di server pemasang: tidak ada telemetri,
tidak ada upload sampel ke pihak ketiga, tidak ada panggilan jaringan
saat berjalan. Report berisi path server — jangan dipublikasikan mentah.
