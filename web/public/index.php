<?php
// Web demo GuardCompress: drop file -> dicek -> bersih balik ke web, virus ditolak.
// Tools di src/ adalah SALINAN dari sistem utama (berdiri sendiri di folder ini).
declare(strict_types=1);

require __DIR__ . '/../src/GuardResult.php';
require __DIR__ . '/../src/GuardCompress.php';

use GuardCompress\GuardCompress;
use GuardCompress\InfectedFileException;
use GuardCompress\BusyException;

$config = require __DIR__ . '/../config.php';

if ($_SERVER['REQUEST_METHOD'] === 'POST') {
    header('Content-Type: application/json');
    $up = $_FILES['media'] ?? null;
    if (!$up || $up['error'] !== UPLOAD_ERR_OK) {
        http_response_code(400);
        echo json_encode(['status' => 'error', 'reason' => 'upload gagal, kode: ' . ($up['error'] ?? '?')]);
        exit;
    }
    $opts = [
        'allow_ext' => $config['allow_ext'],
        'max_mb' => $config['max_mb'],
        'ffmpeg_threads' => $config['ffmpeg_threads'],
    ];
    if (!empty($config['max_slots'])) {
        $opts['max_slots'] = $config['max_slots'];
    }
    try {
        $r = GuardCompress::process($up['tmp_name'], $opts);
    } catch (InfectedFileException $e) {
        http_response_code(422);
        echo json_encode(['status' => 'blocked', 'reason' => $e->getMessage()]);
        exit;
    } catch (BusyException $e) {
        http_response_code(429);
        echo json_encode(['status' => 'busy', 'reason' => $e->getMessage()]);
        exit;
    } catch (Throwable $e) {
        http_response_code(500);
        echo json_encode(['status' => 'error', 'reason' => $e->getMessage()]);
        exit;
    }
    // Lolos: tembak balik ke web (simpan + tampilkan).
    $destDir = $config['upload_dir'];
    if (!is_dir($destDir)) {
        mkdir($destDir, 0755, true);
    }
    $base = basename($r->path);
    $dot = strrpos($base, '.');
    $name = ($dot === false ? $base : substr($base, 0, $dot))
        . '-' . bin2hex(random_bytes(4))
        . ($dot === false ? '' : substr($base, $dot));
    copy($r->path, $destDir . '/' . $name);
    GuardCompress::cleanup(dirname($r->path));
    $saved = (int)($r->report['orig_bytes'] ?? 0) - (int)($r->report['new_bytes'] ?? 0);
    echo json_encode([
        'status' => 'clean',
        'file' => $name,
        'url' => 'uploads/' . $name,
        'mime' => $r->report['detected_mime'] ?? '',
        'from' => $r->report['orig_bytes'] ?? 0,
        'to' => $r->report['new_bytes'] ?? 0,
        'saved' => $saved > 0 ? $saved : 0,
    ]);
    exit;
}
?>
<!doctype html>
<html lang="id">
<head><meta charset="utf-8"><title>GuardCompress Web</title>
<style>
  body{font-family:sans-serif;max-width:720px;margin:2em auto;padding:0 1em}
  #drop{border:3px dashed #888;border-radius:12px;padding:3em 1em;text-align:center;color:#555}
  #drop.over{border-color:#1a7;background:#f0fff8}
  .card{border:1px solid #ccc;border-radius:8px;padding:.7em 1em;margin:.6em 0}
  .clean{border-left:8px solid green}.blocked{border-left:8px solid red}
  .busy,.error{border-left:8px solid orange}
  img{max-width:200px;display:block;margin-top:.5em}
</style></head>
<body>
<h1>Drop file di sini — langsung dicek</h1>
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
    card.textContent = 'tunggu... ' + f.name;
    out.prepend(card);
    const fd = new FormData(); fd.append('media', f);
    try {
      const res = await fetch('', {method: 'POST', body: fd});
      const r = await res.json();
      if (r.status === 'clean') {
        card.className = 'card clean';
        card.innerHTML = '<b>' + esc(f.name) + '</b> BERSIH — ' + esc(r.mime || '') +
          ' (' + r.from + ' -> ' + r.to + ' bytes, hemat ' + r.saved + ')' +
          (/\.(png|jpe?g|webp|gif)$/i.test(r.url || '') ? '<img src="' + r.url + '">' : '');
      } else if (r.status === 'blocked') {
        card.className = 'card blocked';
        card.innerHTML = '<b>' + esc(f.name) + '</b> DITOLAK — ' + esc(r.reason || '');
      } else if (r.status === 'busy') {
        card.className = 'card busy';
        card.innerHTML = '<b>' + esc(f.name) + '</b> SERVER SIBUK — coba lagi nanti';
      } else {
        card.className = 'card error';
        card.textContent = f.name + ' error: ' + (r.reason || '');
      }
    } catch (err) { card.className = 'card error'; card.textContent = f.name + ': ' + err; }
  }
}
function esc(s){return String(s).replace(/[&<>"']/g, c => ({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[c]));}
</script>
</body>
</html>
