'use strict';
const { processFile } = require('./client');

const image = (p, o = {}) => processFile(p, { allow_ext: ['jpg', 'jpeg', 'png', 'webp', 'gif'], ...o });
const video = (p, o = {}) => processFile(p, { allow_ext: ['mp4', 'mov', 'webm', 'mkv', 'avi'], ...o });
const audio = (p, o = {}) => processFile(p, { allow_ext: ['mp3', 'wav', 'ogg', 'oga', 'm4a', 'flac'], ...o });

module.exports = { image, video, audio };
