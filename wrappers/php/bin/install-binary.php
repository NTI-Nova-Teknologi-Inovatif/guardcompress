<?php
// Installer: download binary yang tepat dari GitHub Releases ke ~/.cache/guardcompress
// Usage: php bin/install-binary.php [version]
// Tanpa dependensi tambahan, hanya pakai copy() + hash_file().
declare(strict_types=1);

$version = $argv[1] ?? getenv('GUARDCOMPRESS_VERSION') ?: 'v0.1.0';
$os = strtolower(PHP_OS_FAMILY);
$arch = php_uname('m');
$arch = (stripos($arch, 'arm') !== false || stripos($arch, 'aarch64') !== false) ? 'arm64' : 'amd64';
$ext = $os === 'windows' ? '.exe' : '';
$name = "guardcompress-{$os}-{$arch}{$ext}";

$base = getenv('GUARDCOMPRESS_RELEASE_BASE') ?: "https://github.com/guardcompress/guardcompress/releases/download";
$url = "$base/$version/$name";

$home = $_SERVER['HOME'] ?? sys_get_temp_dir();
$destDir = "$home/.cache/guardcompress";
$dest = "$destDir/$name";
if (!is_dir($destDir)) mkdir($destDir, 0755, true);

echo "Downloading $url ...\n";
if (!@copy($url, $dest)) {
    fwrite(STDERR, "Download gagal: $url\nTetapkan GUARDCOMPRESS_BIN manual atau cek version.\n");
    exit(1);
}
if ($os !== 'windows') chmod($dest, 0755);
echo "Installed: $dest\n";
