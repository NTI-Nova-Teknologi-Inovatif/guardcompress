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

  if (r.status === 2) { const e = new Error('blocked: ' + (report.reason || '')); e.report = report; e.code = 'BLOCKED'; throw e; }
  if (r.status !== 0) { const e = new Error('guardcompress failed: ' + (report.reason || r.stderr)); e.report = report; throw e; }
  return { path: report.out_path, report };
}

module.exports = { process: processFile, processFile, resolveBinary };
