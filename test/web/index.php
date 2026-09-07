<?php
// Web uji coba GuardCompress — drag & drop, hasil langsung (AJAX).
// HANYA untuk test lokal, jangan expose ke publik.
// Jalankan:  php -S localhost:8080  (dari folder test/web/)
// Perlu: env GUARDCOMPRESS_BIN menunjuk ke binary core.
declare(strict_types=1);

require __DIR__ . '/../../wrappers/php/src/GuardResult.php';
require __DIR__ . '/../../wrappers/php/src/GuardCompress.php';

use GuardCompress\GuardCompress;
use GuardCompress\InfectedFileException;

if ($_SERVER['REQUEST_METHOD'] === 'POST') {
    header('Content-Type: application/json');
    $up = $_FILES['media'] ?? null;
    if (!$up || $up['error'] !== UPLOAD_ERR_OK) {
        echo json_encode(['status' => 'error', 'reason' => 'upload gagal, kode: ' . ($up['error'] ?? '?')]);
        exit;
    }
    try {
        $r = GuardCompress::process($up['tmp_name'], [
            'max_mb' => 50,
            'allow_ext' => ['jpg', 'jpeg', 'png', 'webp', 'gif', 'mp4', 'mp3'],
        ]);
        $destDir = __DIR__ . '/uploads';
        if (!is_dir($destDir)) mkdir($destDir, 0755, true);
        copy($r->path, $destDir . '/' . basename($r->path));
        GuardCompress::cleanup(dirname($r->path));
        echo json_encode([
            'status' => 'clean',
            'file' => basename($r->path),
            'url' => 'uploads/' . basename($r->path),
            'mime' => $r->report['detected_mime'] ?? '',
            'from' => $r->report['orig_bytes'] ?? 0,
            'to' => $r->report['new_bytes'] ?? 0,
        ]);
    } catch (InfectedFileException $e) {
        echo json_encode(['status' => 'blocked', 'reason' => $e->getMessage()]);
    } catch (Throwable $e) {
        echo json_encode(['status' => 'error', 'reason' => $e->getMessage()]);
    }
    exit;
}
?>
<!doctype html>
<html lang="id">
<head><meta charset="utf-8"><title>Uji GuardCompress — Drop &amp; Scan</title>
<style>
  body{font-family:sans-serif;max-width:720px;margin:2em auto;padding:0 1em}
  #drop{border:3px dashed #888;border-radius:12px;padding:3em 1em;text-align:center;color:#555}
  #drop.over{border-color:#1a7;background:#f0fff8}
  .card{border:1px solid #ccc;border-radius:8px;padding:.7em 1em;margin:.6em 0}
  .clean{border-left:8px solid green}.blocked{border-left:8px solid red}.error{border-left:8px solid orange}
  img{max-width:200px;display:block;margin-top:.5em}
</style></head>
<body>
<h1>Drop file di sini — langsung di-scan</h1>
<div id="drop">Tarik &amp; letakkan file di sini<br>atau <input type="file" id="pick" multiple></div>
<div id="out"></div>
<script>
const drop = document.getElementById('drop'), out = document.getElementById('out');
drop.addEventListener('dragover', e => { e.preventDefault(); drop.classList.add('over'); });
drop.addEventListener('dragleave', () => drop.classList.remove('over'));
drop.addEventListener('drop', e => { e.preventDefault(); drop.classList.remove('over'); send(e.dataTransfer.files); });
document.getElementById('pick').addEventListener('change', e => send(e.target.files));
async function send(files) {
  for (const f of files) {
    const card = document.createElement('div');
    card.className = 'card';
    card.textContent = '⏳ ' + f.name + ' ...';
    out.prepend(card);
    const fd = new FormData(); fd.append('media', f);
    try {
      const r = await (await fetch('', {method: 'POST', body: fd})).json();
      if (r.status === 'clean') {
        const save = r.from - r.to;
        card.className = 'card clean';
        card.innerHTML = '✅ <b>' + esc(f.name) + '</b> BERSIH — ' + r.mime +
          ' (' + r.from + ' → ' + r.to + ' bytes, hemat ' + save + ')' +
          (r.url.match(/\.(png|jpe?g|webp|gif)$/i) ? '<img src="' + r.url + '">' : '');
      } else if (r.status === 'blocked') {
        card.className = 'card blocked';
        card.innerHTML = '⛔ <b>' + esc(f.name) + '</b> DITOLAK — ' + esc(r.reason);
      } else {
        card.className = 'card error';
        card.textContent = '⚠️ ' + f.name + ' error: ' + r.reason;
      }
    } catch (err) { card.className = 'card error'; card.textContent = '⚠️ ' + f.name + ': ' + err; }
  }
}
function esc(s){return s.replace(/[&<>"']/g, c => ({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[c]));}
</script>
<hr><p>File uji: <code>../fixtures/clean.png</code> (lolos) vs
<code>../fixtures/evil-php.png</code> / <code>../fixtures/evil-eicar.png</code> (ditolak).</p>
</body>
</html>
