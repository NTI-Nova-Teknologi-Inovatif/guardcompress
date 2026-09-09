'use strict';
const os = require('os');
const { processFile, processFileAsync } = require('./client');

function toList(items) {
  if (Array.isArray(items)) return items.map((v) => (typeof v === 'string' ? { path: v } : v));
  return Object.entries(items).map(([key, v]) => ({ key, ...(typeof v === 'string' ? { path: v } : v) }));
}

function collect(list, asDict, out) {
  if (!asDict) return out;
  const keys = list.map((it, i) => (it.key !== undefined ? it.key : i));
  return Object.fromEntries(keys.map((k, i) => [k, out[i]]));
}

function batch(items, opts = {}) {
  const list = toList(items);
  const asDict = !Array.isArray(items);
  const out = asDict ? {} : [];
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
  return collect(list, asDict, out);
}

module.exports = { batch, batchAsync };
