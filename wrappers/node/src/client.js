'use strict';
const { spawnSync, spawn } = require('child_process');
const os = require('os');
const path = require('path');
const fs = require('fs');

function resolveBinary() {
  if (process.env.GUARDCOMPRESS_BIN) return process.env.GUARDCOMPRESS_BIN;
  const plat = process.platform === 'win32' ? 'windows' : process.platform;
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

function parseReport(stdout, stderr) {
  try { return JSON.parse((stdout || '').trim().split('\n').pop() || '{}'); }
  catch { return { reason: stdout + stderr }; }
}

function processFile(inPath, opts = {}) {
  const bin = resolveBinary();
  const outDir = fs.mkdtempSync(path.join(os.tmpdir(), 'gc-'));
  const r = spawnSync(bin, [
    'check', '--in', inPath, '--out-dir', outDir,
    '--config', JSON.stringify(opts), '--json',
  ], { encoding: 'utf8', timeout: (opts.timeoutSec || 120) * 1000 });

  const report = parseReport(r.stdout, r.stderr);
  const cleanup = () => { try { fs.rmSync(outDir, { recursive: true, force: true }); } catch {} };
  if (r.status === 2) { cleanup(); const e = new Error('blocked: ' + (report.reason || '')); e.report = report; e.code = 'BLOCKED'; throw e; }
  if (r.status !== 0) {
    cleanup();
    if (report.details && report.details.busy) { const e = new Error(report.reason || 'server busy'); e.report = report; e.code = 'BUSY'; throw e; }
    const e = new Error('guardcompress failed: ' + (report.reason || r.stderr)); e.report = report; throw e;
  }
  if (r.error) { cleanup(); const e = new Error('guardcompress timeout/crash: ' + r.error.message); e.report = report; throw e; }
  if (!report.out_path) { cleanup(); const e = new Error('guardcompress: out_path hilang dari report'); e.report = report; throw e; }
  return { path: report.out_path, report };
}

function processFileAsync(inPath, opts = {}) {
  return new Promise((resolve, reject) => {
    let bin;
    try { bin = resolveBinary(); } catch (e) { return reject(e); }
    fs.mkdtemp(path.join(os.tmpdir(), 'gc-'), (err, outDir) => {
      if (err) return reject(err);
      const cleanup = () => { try { fs.rmSync(outDir, { recursive: true, force: true }); } catch {} };
      const child = spawn(bin, ['check', '--in', inPath, '--out-dir', outDir,
        '--config', JSON.stringify(opts), '--json'], { timeout: (opts.timeoutSec || 120) * 1000 });
      let stdout = '', stderr = '';
      child.stdout.on('data', (d) => { stdout += d; });
      child.stderr.on('data', (d) => { stderr += d; });
      child.on('error', (e) => { cleanup(); e.report = {}; reject(e); });
      child.on('close', (code) => {
        const report = parseReport(stdout, stderr);
        if (code === 2) { cleanup(); const e = new Error('blocked: ' + (report.reason || '')); e.report = report; e.code = 'BLOCKED'; return reject(e); }
        if (code !== 0) {
          cleanup();
          if (report.details && report.details.busy) { const e = new Error(report.reason || 'server busy'); e.report = report; e.code = 'BUSY'; return reject(e); }
          const e = new Error('guardcompress failed: ' + (report.reason || stderr)); e.report = report; return reject(e);
        }
        if (!report.out_path) { cleanup(); const e = new Error('guardcompress: out_path hilang dari report'); e.report = report; return reject(e); }
        resolve({ path: report.out_path, report });
      });
    });
  });
}

function cleanup(dir) {
  try { fs.rmSync(dir, { recursive: true, force: true }); } catch {}
}

module.exports = { resolveBinary, processFile, processFileAsync, cleanup };
