<?php
declare(strict_types=1);

namespace GuardCompress;

class GuardCompress
{
    public static function process(string $inPath, array $opts = []): GuardResult
    {
        $bin = self::resolveBinary();
        $outDir = sys_get_temp_dir() . '/gc-' . uniqid();
        if (!mkdir($outDir, 0777, true) && !is_dir($outDir)) {
            throw new \RuntimeException("cannot create tmp dir: $outDir");
        }
        $config = escapeshellarg(json_encode($opts ?: new \stdClass()));
        $cmd = escapeshellarg($bin)
            . ' check --in ' . escapeshellarg($inPath)
            . ' --out-dir ' . escapeshellarg($outDir)
            . ' --config ' . $config
            . ' --json 2>&1';

        exec($cmd, $lines, $code);
        $json = implode("\n", $lines);
        $report = json_decode($json, true) ?? ['reason' => $json];

        if ($code === 2) {
            throw new InfectedFileException($report['reason'] ?? 'blocked', $report);
        }
        if ($code !== 0) {
            throw new GuardException($report['reason'] ?? 'guardcompress failed', $report);
        }
        return new GuardResult($report['out_path'], $report);
    }

    public static function resolveBinary(): string
    {
        // 1. env override, 2. cache dir, 3. vendor/bin fallback
        if ($env = getenv('GUARDCOMPRESS_BIN')) return $env;
        $os = strtolower(PHP_OS_FAMILY); // linux, windows, darwin
        $arch = php_uname('m');
        $arch = str_contains($arch, 'arm') || str_contains($arch, 'aarch64') ? 'arm64' : 'amd64';
        $ext = $os === 'windows' ? '.exe' : '';
        $name = "guardcompress-{$os}-{$arch}{$ext}";
        foreach([
            ($_SERVER['HOME'] ?? sys_get_temp_dir()) . "/.cache/guardcompress/$name",
            __DIR__ . "/../../core/bin/$name",
            __DIR__ . "/../../bin/$name",
        ] as $p) { if (is_file($p)) return $p; }
        // belum di-download -> arahkan ke installer
        throw new \RuntimeException(
            "guardcompress binary not found ($name). Run: php bin/install-binary.php"
        );
    }
}
