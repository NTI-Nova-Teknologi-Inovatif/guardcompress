'use strict';
// Download core binary + ffmpeg dari GitHub Releases (verifikasi SHA256).
// Env: GUARDCOMPRESS_VERSION, GUARDCOMPRESS_RELEASE_BASE
const https = require('https');
const fs = require('fs');
const os = require('os');
const path = require('path');
const crypto = require('crypto');

const version = process.env.GUARDCOMPRESS_VERSION || 'v0.1.0';
const base = process.env.GUARDCOMPRESS_RELEASE_BASE || 'https://github.com/NTI-Nova-Teknologi-Inovatif/guardcompress/releases/download';
const plat = process.platform === 'win32' ? 'windows' : process.platform;
const arch = process.arch === 'arm64' ? 'arm64' : 'amd64';
const ext = plat === 'windows' ? '.exe' : '';
const files = [`guardcompress-${plat}-${arch}${ext}`, `ffmpeg-${plat}-${arch}${ext}`];
const destDir = path.join(os.homedir(), '.cache', 'guardcompress');

function get(url) {
  return new Promise((resolve, reject) => {
    https.get(url, (res) => {
      if (res.statusCode >= 300 && res.statusCode < 400 && res.headers.location) {
        return resolve(get(res.headers.location));
      }
      if (res.statusCode !== 200) return reject(new Error(`HTTP ${res.statusCode}: ${url}`));
      const chunks = [];
      res.on('data', (c) => chunks.push(c));
      res.on('end', () => resolve(Buffer.concat(chunks)));
    }).on('error', reject);
  });
}

function sha256(buf) { return crypto.createHash('sha256').update(buf).digest('hex'); }

(async () => {
  fs.mkdirSync(destDir, { recursive: true });
  console.log(`[guardcompress] fetching CHECKSUMS.txt ${version} ...`);
  let sums;
  try {
    sums = (await get(`${base}/${version}/CHECKSUMS.txt`)).toString('utf8');
  } catch (e) {
    console.error(`[guardcompress] CHECKSUMS tak bisa diunduh: ${e.message}`);
    console.error(`[guardcompress] Set GUARDCOMPRESS_BIN manual. Core tetap bisa jalan mode guard-only.`);
    return; // jangan gagalkan npm install
  }
  const want = {};
  for (const line of sums.split('\n')) {
    const m = line.trim().match(/^([0-9a-f]{64})\s+(\S+)$/);
    if (m) want[m[2]] = m[1];
  }
  for (const f of files) {
    const dest = path.join(destDir, f);
    const isFFmpeg = f.startsWith('ffmpeg-');
    try {
      if (!want[f]) {
        if (isFFmpeg) { console.log(`[guardcompress] ${f} belum dirilis, lewati (mode guard-only).`); continue; }
        throw new Error(`checksum untuk ${f} tidak ada di CHECKSUMS.txt`);
      }
      if (fs.existsSync(dest) && sha256(fs.readFileSync(dest)) === want[f]) {
        console.log(`[guardcompress] ${f} sudah ada & cocok, lewati.`);
        continue;
      }
      console.log(`[guardcompress] downloading ${f} ...`);
      const buf = await get(`${base}/${version}/${f}`);
      if (sha256(buf) !== want[f]) throw new Error(`checksum ${f} TIDAK COCOK, dibuang`);
      fs.writeFileSync(dest, buf, { mode: 0o755 });
      console.log(`[guardcompress] installed: ${dest}`);
    } catch (e) {
      console.error(`[guardcompress] ${f} gagal: ${e.message}`);
      if (!isFFmpeg) process.exitCode = 1;
    }
  }
})();
