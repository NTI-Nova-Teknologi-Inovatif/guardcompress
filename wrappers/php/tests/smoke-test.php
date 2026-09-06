<?php
// Smoke test wrapper PHP (tidak perlu composer/phpunit).
declare(strict_types=1);
require __DIR__ . '/../src/GuardResult.php';
require __DIR__ . '/../src/GuardCompress.php';

use GuardCompress\GuardCompress;
use GuardCompress\InfectedFileException;

$fails = 0;
$r = GuardCompress::process('C:/Users/allmi/AppData/Local/Temp/opencode/clean.png', []);
echo "php-clean: {$r->report['status']}\n";
if ($r->report['status'] !== 'clean') { $fails++; }

// Regression: opts berisi kutip (array allow) harus lolos utuh ke core,
// terutama di Windows (cmd.exe merusak escapeshellarg+exec).
$r2 = GuardCompress::process('C:/Tools/GuardCompress/test/fixtures/clean.png', [
    'max_mb' => 50,
    'allow' => ['image/jpeg', 'image/png', 'image/webp'],
]);
echo "php-opts-quotes: {$r2->report['status']}\n";
if ($r2->report['status'] !== 'clean') { $fails++; }

try {
    GuardCompress::process('C:/Users/allmi/AppData/Local/Temp/opencode/eicar.txt', []);
    echo "php-blocked: FAIL (tidak diblokir)\n";
    $fails++;
} catch (InfectedFileException $e) {
    echo "php-blocked: BLOCKED\n";
}
exit($fails ? 1 : 0);
