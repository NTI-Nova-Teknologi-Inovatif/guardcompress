'use strict';
// postinstall: download binary sesuai platform dari GitHub Releases
const https = require('https');
const fs = require('fs');
const os = require('os');
const path = require('path');

const version = process.env.GUARDCOMPRESS_VERSION || 'v0.1.0';
const base = process.env.GUARDCOMPRESS_RELEASE_BASE || 'https://github.com/guardcompress/guardcompress/releases/download';
const plat = process.platform === 'win32' ? 'windows' : process.platform;
const arch = process.arch === 'arm64' ? 'arm64' : 'amd64';
const ext = plat === 'windows' ? '.exe' : '';
const name = `guardcompress-${plat}-${arch}${ext}`;
const url = `${base}/${version}/${name}`;
const destDir = path.join(os.homedir(), '.cache', 'guardcompress');
const dest = path.join(destDir, name);

fs.mkdirSync(destDir, { recursive: true });
console.log(`Downloading ${url} ...`);
https.get(url, (res) => {
  if (res.statusCode !== 200) { console.error(`Download gagal: ${res.statusCode} ${url}`); process.exit(1); }
  const f = fs.createWriteStream(dest, { mode: 0o755 });
  res.pipe(f);
  f.on('finish', () => { f.close(); console.log(`Installed: ${dest}`); });
}).on('error', (e) => { console.error(e.message); process.exit(1); });
