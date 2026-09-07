'use strict';
// Thin wrapper: spawnSync guardcompress binary, parse report.json
const { spawnSync } = require('child_process');
const os = require('os');
const path = require('path');
const fs = require('fs');

function resolveBinary() {
  if (process.env.GUARDCOMPRESS_BIN) return process.env.GUARDCOMPRESS_BIN;
  const plat = process.platform === 'win32' ? 'windows' : process.platform; // linux, darwin
  const arch = process.arch === 'arm64' ? 'arm64' : 'amd64';
  const ext = plat === 'windows' ? '.exe' : '';
  const name = `guardcompress-${plat}-${arch}${ext}`;
  const cands = [
    path.join(os.homedir(), '.cache', 'guardcompress', name),
    path.join(__dirname, '..', '..', 'core', 'bin', name),
  ];
  for (const p of cands) if (fs.existsSync(p)) return p;
  throw new Error(`guardcompress binary not found (${name}). Run node scripts/postinstall.js`);
}

function processFile(inPath, opts = {}) {
  const bin = resolveBinary();
  const outDir = fs.mkdtempSync(path.join(os.tmpdir(), 'gc-'));
  const r = spawnSync(bin, [
    'check', '--in', inPath, '--out-dir', outDir,
    '--config', JSON.stringify(opts), '--json',
  ], { encoding: 'utf8', timeout: (opts.timeoutSec || 120) * 1000 });

  let report = {};
  try { report = JSON.parse((r.stdout || '').trim().split('\n').pop() || '{}'); }
  catch { report = { reason: r.stdout + r.stderr }; }

  const cleanup = () => { try { fs.rmSync(outDir, { recursive: true, force: true }); } catch {} };
  if (r.status === 2) { cleanup(); const e = new Error('blocked: ' + (report.reason || '')); e.report = report; e.code = 'BLOCKED'; throw e; }
  if (r.status !== 0) { cleanup(); const e = new Error('guardcompress failed: ' + (report.reason || r.stderr)); e.report = report; throw e; }
  if (r.error) { cleanup(); const e = new Error('guardcompress timeout/crash: ' + r.error.message); e.report = report; throw e; }
  // Gagal cepat di batas: jangan kembalikan path undefined yang meledak
  // belakangan di kode app dengan pesan membingungkan.
  if (!report.out_path) { cleanup(); const e = new Error('guardcompress: out_path hilang dari report'); e.report = report; throw e; }
  return { path: report.out_path, report };
}

module.exports = { process: processFile, processFile, resolveBinary,
  // Preset per jenis (1 sistem di belakangnya, opts user menang bila menimpa).
  image: (p, o = {}) => processFile(p, { allow_ext: ['jpg', 'jpeg', 'png', 'webp', 'gif'], ...o }),
  video: (p, o = {}) => processFile(p, { allow_ext: ['mp4', 'mov', 'webm', 'mkv', 'avi'], ...o }),
  audio: (p, o = {}) => processFile(p, { allow_ext: ['mp3', 'wav', 'ogg', 'oga', 'm4a', 'flac'], ...o }),
  // Batch multi-input beda jenis: {avatar:'/tmp/a.png', video:'/tmp/b.mp4'}
  // atau [{path, opts}]. File ditolak terkumpul (ok:false), error teknis lempar.
  batch: (items, opts = {}) => {
    const list = Array.isArray(items) ? items : Object.entries(items).map(([k, v]) => ({ key: k, ...(typeof v === 'string' ? { path: v } : v) }));
    const out = Array.isArray(items) ? [] : {};
    for (const it of list) {
      const k = it.key !== undefined ? it.key : out.length;
      try {
        out[k] = { ok: true, ...processFile(it.path, { ...opts, ...(it.opts || {}) }) };
      } catch (e) {
        if (e.code === 'BLOCKED') out[k] = { ok: false, blocked: true, reason: e.message, report: e.report };
        else throw e;
      }
    }
    return out;
  },
};
