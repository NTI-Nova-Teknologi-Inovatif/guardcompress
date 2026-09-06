// Contoh BullMQ worker (Node): GuardCompress di background.
// npm i bullmq && node examples/bullmq-worker.js
const { Worker } = require('bullmq');
const fs = require('fs');
const gc = require('../wrappers/node/src/index');

new Worker('uploads', async (job) => {
  const { tmpPath, userId } = job.data;
  try {
    const r = gc.process(tmpPath, { max_mb: 500, video_crf: 28, timeoutSec: 600 });
    // TODO: upload r.path ke S3, simpan report ke DB
    return { stored: r.path, saved: r.report.orig_bytes - r.report.new_bytes };
  } catch (e) {
    if (e.code === 'BLOCKED') return { blocked: String(e.message) }; // jangan retry
    throw e; // error teknis -> retry otomatis BullMQ
  } finally {
    fs.rm(tmpPath, { force: true }, () => {});
  }
});
