<?php
declare(strict_types=1);

namespace GuardCompress;

class Client
{
    public static function process(string $inPath, array $opts = []): GuardResult
    {
        $bin = self::resolveBinary();
        try {
            $suffix = bin2hex(random_bytes(8));
        } catch (\Throwable) {
            $suffix = uniqid('', true);
        }
        $outDir = sys_get_temp_dir() . '/gc-' . $suffix;
        if (!mkdir($outDir, 0700, true) && !is_dir($outDir)) {
            throw new \RuntimeException("cannot create tmp dir: $outDir");
        }
        if (!function_exists('proc_open')) {
            throw new \RuntimeException('proc_open() dibutuhkan GuardCompress');
        }
        $config = json_encode($opts ?: new \stdClass());
        $proc = proc_open(
            [$bin, 'check', '--in', $inPath, '--out-dir', $outDir, '--config', $config, '--json'],
            [1 => ['pipe', 'w'], 2 => ['pipe', 'w']],
            $pipes
        );
        if (!is_resource($proc)) {
            self::rmDir($outDir);
            throw new GuardException('gagal menjalankan guardcompress binary');
        }
        $stdout = stream_get_contents($pipes[1]);
        fclose($pipes[1]);
        fclose($pipes[2]);
        $code = proc_close($proc);

        $json = trim((string)$stdout);
        $last = substr($json, strrpos($json, "\n") === false ? 0 : strrpos($json, "\n") + 1);
        $report = json_decode($last, true) ?? ['reason' => $json];

        if ($code === 2) {
            self::rmDir($outDir);
            throw new InfectedFileException($report['reason'] ?? 'blocked', $report);
        }
        if (!empty($report['details']['busy'])) {
            self::rmDir($outDir);
            throw new BusyException($report['reason'] ?? 'server busy', $report);
        }
        if ($code !== 0) {
            self::rmDir($outDir);
            throw new GuardException($report['reason'] ?? 'guardcompress failed', $report);
        }
        if (empty($report['out_path']) || !is_file($report['out_path'])) {
            self::rmDir($outDir);
            throw new GuardException('guardcompress: out_path hilang', $report);
        }
        return new GuardResult($report['out_path'], $report);
    }

    public static function cleanup(string $dir): void
    {
        self::rmDir($dir);
    }

    public static function rmDir(string $dir): void
    {
        if (!is_dir($dir)) return;
        $it = new \RecursiveIteratorIterator(
            new \RecursiveDirectoryIterator($dir, \FilesystemIterator::SKIP_DOTS),
            \RecursiveIteratorIterator::CHILD_FIRST
        );
        foreach ($it as $f) { $f->isDir() ? rmdir($f->getPathname()) : unlink($f->getPathname()); }
        rmdir($dir);
    }

    public static function resolveBinary(): string
    {
        if ($env = getenv('GUARDCOMPRESS_BIN')) return $env;
        $os = strtolower(PHP_OS_FAMILY);
        $arch = php_uname('m');
        $arch = str_contains($arch, 'arm') || str_contains($arch, 'aarch64') ? 'arm64' : 'amd64';
        $ext = $os === 'windows' ? '.exe' : '';
        $name = "guardcompress-{$os}-{$arch}{$ext}";
        $home = $_SERVER['HOME'] ?? getenv('HOME') ?? getenv('USERPROFILE') ?: sys_get_temp_dir();
        foreach([
            "$home/.cache/guardcompress/$name",
            __DIR__ . "/../../core/bin/$name",
            __DIR__ . "/../../bin/$name",
        ] as $p) { if (is_file($p)) return $p; }
        throw new \RuntimeException(
            "guardcompress binary not found ($name). Run: php bin/install-binary.php"
        );
    }
}
