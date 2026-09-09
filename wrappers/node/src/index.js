'use strict';
const { resolveBinary, processFile, processFileAsync, cleanup } = require('./client');
const { batch, batchAsync } = require('./batch');
const { image, video, audio } = require('./presets');

module.exports = {
  process: processFile,
  processFile,
  processFileAsync,
  resolveBinary,
  cleanup,
  image,
  video,
  audio,
  batch,
  batchAsync,
};
