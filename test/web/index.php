<?php
// Web uji coba GuardCompress — HANYA untuk test lokal, jangan expose ke publik.
// Jalankan:  php -S localhost:8080  (dari folder test/web/)
// Perlu: env GUARDCOMPRESS_BIN menunjuk ke binary core.
// Coba upload: ../fixtures/clean.png (lolos) vs evil-php.png / evil-eicar.png (ditolak).
declare(strict_types=1);

require __DIR__ . '/../../wrappers/php/src/GuardResult.php';
require __DIR__ . '/../../wrappers/php/src/GuardCompress.php';

use GuardCompress\GuardCompress;
use GuardCompress\InfectedFileException;

$result = null;
$error = null;

if ($_SERVER['REQUEST_METHOD'] === 'POST' && isset($_FILES['media'])) {
    $up = $_FILES['media'];
    if ($up['error'] !== UPLOAD_ERR_OK) {
        $error = 'Upload gagal, kode: ' . $up['error'];
    } else {
        try {
            $r = GuardCompress::process($up['tmp_name'], [
                'max_mb' => 50,
                'allow' => ['image/jpeg', 'image/png', 'image/webp'],
            ]);
            $destDir = __DIR__ . '/uploads';
            if (!is_dir($destDir)) mkdir($destDir, 0755, true);
            $dest = $destDir . '/' . basename($r->path);
            copy($r->path, $dest);
            GuardCompress::cleanup(dirname($r->path));
            $result = [
                'stored' => 'uploads/' . basename($r->path),
                'from' => $r->report['orig_bytes'] ?? 0,
                'to' => $r->report['new_bytes'] ?? 0,
                'mime' => $r->report['detected_mime'] ?? '',
            ];
        } catch (InfectedFileException $e) {
            $error = 'DITOLAK (file berbahaya): ' . htmlspecialchars($e->getMessage());
        } catch (Throwable $e) {
            $error = 'Error: ' . htmlspecialchars($e->getMessage());
        }
    }
}
?>
<!doctype html>
<html lang="id">
<head><meta charset="utf-8"><title>Uji GuardCompress</title></head>
<body>
<h1>Uji Upload GuardCompress</h1>
<form method="post" enctype="multipart/form-data">
  <input type="file" name="media" accept="image/*" required>
  <button type="submit">Upload &amp; Periksa</button>
</form>
<?php if ($result): ?>
  <h2 style="color:green">BERSIH &amp; TERSIMPAN</h2>
  <p>MIME: <?= htmlspecialchars($result['mime']) ?> |
     <?= (int)$result['from'] ?> -&gt; <?= (int)$result['to'] ?> bytes</p>
  <p><img src="<?= htmlspecialchars($result['stored']) ?>" style="max-width:300px"></p>
<?php elseif ($error): ?>
  <h2 style="color:red"><?= $error ?></h2>
<?php endif; ?>
<hr>
<p>Coba: <code>../fixtures/clean.png</code> (lolos) vs
<code>../fixtures/evil-php.png</code> / <code>../fixtures/evil-eicar.png</code> (ditolak).</p>
</body>
</html>
