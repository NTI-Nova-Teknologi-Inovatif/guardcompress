'use strict';
const { spawnSync, spawn } = require('child_process');
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
  if (r.status !== 0) {
    cleanup();
    if (report.details && report.details.busy) { const e = new Error(report.reason || 'server busy'); e.report = report; e.code = 'BUSY'; throw e; }
    const e = new Error('guardcompress failed: ' + (report.reason || r.stderr)); e.report = report; throw e;
  }
  if (r.error) { cleanup(); const e = new Error('guardcompress timeout/crash: ' + r.error.message); e.report = report; throw e; }
  if (!report.out_path) { cleanup(); const e = new Error('guardcompress: out_path hilang dari report'); e.report = report; throw e; }
  return { path: report.out_path, report };
}

function toList(items) {
  if (Array.isArray(items)) return items.map((v) => (typeof v === 'string' ? { path: v } : v));
  return Object.entries(items).map(([key, v]) => ({ key, ...(typeof v === 'string' ? { path: v } : v) }));
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
        let report = {};
        try { report = JSON.parse((stdout || '').trim().split('\n').pop() || '{}'); }
        catch { report = { reason: stdout + stderr }; }
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

async function batchAsync(items, opts = {}) {
  const list = toList(items);
  const asDict = !Array.isArray(items);
  const jobs = Math.max(1, Math.min(opts.jobs || Math.min(os.cpus().length, 4), 16));
  const out = new Array(list.length);
  let next = 0;
  async function worker() {
    while (next < list.length) {
      const i = next++;
      const it = list[i];
      try {
        out[i] = { ok: true, ...(await processFileAsync(it.path, { ...opts, ...(it.opts || {}) })) };
      } catch (e) {
        if (e.code === 'BLOCKED') out[i] = { ok: false, blocked: true, reason: e.message, report: e.report };
        else throw e;
      }
    }
  }
  await Promise.all(Array.from({ length: Math.min(jobs, list.length) }, worker));
  if (!asDict) return out;
  const keys = list.map((it, i) => (it.key !== undefined ? it.key : i));
  return Object.fromEntries(keys.map((k, i) => [k, out[i]]));
}

module.exports = { process: processFile, processFile, resolveBinary,
  cleanup: (dir) => { try { fs.rmSync(dir, { recursive: true, force: true }); } catch {} },
  image: (p, o = {}) => processFile(p, { allow_ext: ['jpg', 'jpeg', 'png', 'webp', 'gif'], ...o }),
  video: (p, o = {}) => processFile(p, { allow_ext: ['mp4', 'mov', 'webm', 'mkv', 'avi'], ...o }),
  audio: (p, o = {}) => processFile(p, { allow_ext: ['mp3', 'wav', 'ogg', 'oga', 'm4a', 'flac'], ...o }),
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
  batchAsync, processFileAsync,
};
