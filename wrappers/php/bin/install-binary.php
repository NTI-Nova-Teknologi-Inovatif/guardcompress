<?php
// Installer: download core binary + ffmpeg static dari GitHub Releases
// ke ~/.cache/guardcompress, verifikasi SHA256 via CHECKSUMS.txt.
// Usage: php bin/install-binary.php [version]
// Idempotent: file yang hash-nya sudah cocok dilewati.
// Tanpa dependensi tambahan, hanya pakai copy()/file_get_contents() + hash_file().
declare(strict_types=1);

$version = $argv[1] ?? getenv('GUARDCOMPRESS_VERSION') ?: 'v0.1.0';
$os = strtolower(PHP_OS_FAMILY);
$arch = php_uname('m');
$arch = (stripos($arch, 'arm') !== false || stripos($arch, 'aarch64') !== false) ? 'arm64' : 'amd64';
$ext = $os === 'windows' ? '.exe' : '';
$files = ["guardcompress-{$os}-{$arch}{$ext}", "ffmpeg-{$os}-{$arch}{$ext}"];

$base = getenv('GUARDCOMPRESS_RELEASE_BASE') ?: "https://github.com/NTI-Nova-Teknologi-Inovatif/guardcompress/releases/download";

$home = $_SERVER['HOME'] ?? getenv('USERPROFILE') ?? sys_get_temp_dir();
$destDir = "$home/.cache/guardcompress";
if (!is_dir($destDir)) mkdir($destDir, 0755, true);

function fetch(string $url): ?string {
    $ctx = stream_context_create(['http' => ['timeout' => 60, 'follow_location' => 1]]);
    $data = @file_get_contents($url, false, $ctx);
    return $data === false ? null : $data;
}

$sumsRaw = fetch("$base/$version/CHECKSUMS.txt");
if ($sumsRaw === null) {
    fwrite(STDERR, "CHECKSUMS tak bisa diunduh: $base/$version/CHECKSUMS.txt\nSet GUARDCOMPRESS_BIN manual. Core tetap bisa jalan mode guard-only.\n");
    exit(0); // jangan gagalkan composer install
}
$want = [];
foreach (explode("\n", $sumsRaw) as $line) {
    if (preg_match('/^([0-9a-f]{64})\s+(\S+)$/', trim($line), $m)) $want[$m[2]] = $m[1];
}

$fail = 0;
foreach ($files as $f) {
    $isFFmpeg = str_starts_with($f, 'ffmpeg-');
    $dest = "$destDir/$f";
    if (!isset($want[$f])) {
        echo $isFFmpeg ? "$f belum dirilis, lewati (mode guard-only).\n" : "checksum $f tidak ada di CHECKSUMS.txt\n";
        if (!$isFFmpeg) $fail = 1;
        continue;
    }
    if (is_file($dest) && hash_file('sha256', $dest) === $want[$f]) {
        echo "$f sudah ada & cocok, lewati.\n";
        continue;
    }
    echo "Downloading $f ...\n";
    $buf = fetch("$base/$version/$f");
    if ($buf === null || hash('sha256', $buf) !== $want[$f]) {
        fwrite(STDERR, "$f gagal (download/checksum tidak cocok), dibuang.\n");
        if (!$isFFmpeg) $fail = 1;
        continue;
    }
    file_put_contents($dest, $buf);
    if ($os !== 'windows') chmod($dest, 0755);
    echo "Installed: $dest\n";
}
exit($fail);
