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

try {
    GuardCompress::process('C:/Users/allmi/AppData/Local/Temp/opencode/eicar.txt', []);
    echo "php-blocked: FAIL (tidak diblokir)\n";
    $fails++;
} catch (InfectedFileException $e) {
    echo "php-blocked: BLOCKED\n";
}
exit($fails ? 1 : 0);
