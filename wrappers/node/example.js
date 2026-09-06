// Express/Multer example (docs, bukan bagian package)
const multer = require('multer');
const gc = require('./src/index');
const upload = multer({ dest: 'tmp/' });

app.post('/upload', upload.single('video'), (req, res) => {
  try {
    const r = gc.process(req.file.path, { max_mb: 500, video_crf: 28 });
    // TODO: picks3.put(r.path) lalu hapus tmp
    res.json({ stored: r.path, from: r.report.orig_bytes, to: r.report.new_bytes });
  } catch (e) {
    if (e.code === 'BLOCKED') return res.status(422).json({ blocked: e.message });
    return res.status(500).json({ error: e.message });
  }
});
