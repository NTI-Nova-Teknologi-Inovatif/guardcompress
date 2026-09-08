'use strict';
const http = require('http');
const fs = require('fs');
const os = require('os');
const path = require('path');
const gc = require('./src/gc');

const PORT = 8080;
const ROOT = __dirname;
const config = JSON.parse(fs.readFileSync(path.join(ROOT, 'config.json'), 'utf8'));
const UPLOADS = path.join(ROOT, 'uploads');
fs.mkdirSync(UPLOADS, { recursive: true });

const TYPES = { '.html': 'text/html', '.js': 'text/javascript', '.css': 'text/css', '.png': 'image/png', '.jpg': 'image/jpeg', '.webp': 'image/webp', '.gif': 'image/gif', '.mp4': 'video/mp4', '.mp3': 'audio/mpeg' };

function safeName(name) {
  const base = path.basename(String(name || 'file')).replace(/[^A-Za-z0-9._-]/g, '_').slice(0, 80) || 'file';
  const dot = base.lastIndexOf('.');
  return (dot < 0 ? base : base.slice(0, dot)) + '-' + gc.randHex(4) + (dot < 0 ? '' : base.slice(dot));
}

const server = http.createServer((req, res) => {
  if (req.method === 'POST' && req.url === '/api/upload') {
    const fname = safeName(req.headers['x-filename']);
    const tmp = path.join(os.tmpdir(), 'webupload-' + gc.randHex(8));
    const ws = fs.createWriteStream(tmp);
    req.pipe(ws);
    ws.on('finish', () => {
      const opts = { allow_ext: config.allow_ext, max_mb: config.max_mb, ffmpeg_threads: config.ffmpeg_threads };
      if (config.max_slots) opts.max_slots = config.max_slots;
      try {
        const r = gc.check(tmp, opts);
        const dest = path.join(UPLOADS, safeName(fname));
        fs.copyFileSync(r.out_path, dest);
        r._cleanup();
        const saved = Math.max(0, (r.orig_bytes || 0) - (r.new_bytes || 0));
        res.writeHead(200, { 'Content-Type': 'application/json' });
        res.end(JSON.stringify({ status: 'clean', file: path.basename(dest), url: 'uploads/' + path.basename(dest), mime: r.detected_mime || '', from: r.orig_bytes || 0, to: r.new_bytes || 0, saved }));
      } catch (e) {
        if (e.code === 'BLOCKED') { res.writeHead(422, { 'Content-Type': 'application/json' }); res.end(JSON.stringify({ status: 'blocked', reason: e.message })); }
        else if (e.code === 'BUSY') { res.writeHead(429, { 'Content-Type': 'application/json' }); res.end(JSON.stringify({ status: 'busy', reason: e.message })); }
        else { res.writeHead(500, { 'Content-Type': 'application/json' }); res.end(JSON.stringify({ status: 'error', reason: e.message })); }
      } finally {
        fs.rm(tmp, { force: true }, () => {});
      }
    });
    ws.on('error', () => { res.writeHead(500, { 'Content-Type': 'application/json' }); res.end(JSON.stringify({ status: 'error', reason: 'gagal nampung upload' })); });
    return;
  }
  let f = req.url === '/' ? '/public/index.html' : decodeURIComponent(req.url.split('?')[0]);
  if (f.includes('..')) { res.writeHead(400); res.end('bad path'); return; }
  const full = path.join(ROOT, f);
  fs.readFile(full, (err, data) => {
    if (err) { res.writeHead(404); res.end('not found'); return; }
    res.writeHead(200, { 'Content-Type': TYPES[path.extname(full).toLowerCase()] || 'application/octet-stream' });
    res.end(data);
  });
});

server.listen(PORT, () => console.log(`contoh uji jalan di http://localhost:${PORT}`));
