'use strict';
// Mini client milik folder web/ ini: ngomong langsung ke binary core.
// Nggak require kode utama, cuma butuh env GUARDCOMPRESS_BIN.
const { spawnSync } = require('child_process');
const os = require('os');
const path = require('path');
const fs = require('fs');
const crypto = require('crypto');

function binary() {
  if (process.env.GUARDCOMPRESS_BIN && fs.existsSync(process.env.GUARDCOMPRESS_BIN)) {
    return process.env.GUARDCOMPRESS_BIN;
  }
  const plat = process.platform === 'win32' ? 'windows' : process.platform;
  const arch = process.arch === 'arm64' ? 'arm64' : 'amd64';
  const ext = plat === 'windows' ? '.exe' : '';
  const cand = path.join(os.homedir(), '.cache', 'guardcompress', `guardcompress-${plat}-${arch}${ext}`);
  if (fs.existsSync(cand)) return cand;
  const e = new Error('Binary GuardCompress tidak ketemu. Set env GUARDCOMPRESS_BIN.');
  e.code = 'NOBIN';
  throw e;
}

function check(inPath, opts = {}) {
  const bin = binary();
  const outDir = fs.mkdtempSync(path.join(os.tmpdir(), 'webgc-'));
  const cleanup = () => { try { fs.rmSync(outDir, { recursive: true, force: true }); } catch {} };
  const r = spawnSync(bin, ['check', '--in', inPath, '--out-dir', outDir,
    '--config', JSON.stringify(opts), '--json'],
    { encoding: 'utf8', timeout: (opts.timeoutSec || 120) * 1000 });
  let report = {};
  try { report = JSON.parse((r.stdout || '').trim().split('\n').pop() || '{}'); }
  catch { report = { reason: r.stdout + r.stderr }; }
  if (r.status === 2) { cleanup(); const e = new Error(report.reason || 'blocked'); e.report = report; e.code = 'BLOCKED'; throw e; }
  if (r.status !== 0) {
    cleanup();
    if (report.details && report.details.busy) { const e = new Error(report.reason || 'server busy'); e.report = report; e.code = 'BUSY'; throw e; }
    const e = new Error(report.reason || r.stderr || 'guardcompress failed'); e.report = report; throw e;
  }
  if (r.error) { cleanup(); const e = new Error('timeout/crash: ' + r.error.message); e.report = report; throw e; }
  if (!report.out_path || !fs.existsSync(report.out_path)) { cleanup(); const e = new Error('out_path hilang'); e.report = report; throw e; }
  report._out_dir = outDir;
  report._cleanup = cleanup;
  return report;
}

function randHex(n) { return crypto.randomBytes(n).toString('hex'); }

module.exports = { binary, check, randHex };
