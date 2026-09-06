<?php
declare(strict_types=1);

namespace GuardCompress;

class GuardCompress
{
    public static function process(string $inPath, array $opts = []): GuardResult
    {
        $bin = self::resolveBinary();
        // AUDIT: suffix acak kriptografis (bukan uniqid yang bisa ditebak)
        // agar user lain tak bisa menebak & mengintip folder tmp.
        try {
            $suffix = bin2hex(random_bytes(8));
        } catch (\Throwable) {
            $suffix = uniqid('', true);
        }
        $outDir = sys_get_temp_dir() . '/gc-' . $suffix;
        // AUDIT: 0700 (bukan 0777) agar user lain di shared hosting tak bisa intip file.
        if (!mkdir($outDir, 0700, true) && !is_dir($outDir)) {
            throw new \RuntimeException("cannot create tmp dir: $outDir");
        }
        // AUDIT: proc_open array (tanpa shell) — escapeshellarg+exec rusak
        // di Windows bila JSON berisi kutip (cmd.exe mengupasnya).
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

        // Ambil baris JSON terakhir (abaikan log lain)
        $json = trim((string)$stdout);
        $last = substr($json, strrpos($json, "\n") === false ? 0 : strrpos($json, "\n") + 1);
        $report = json_decode($last, true) ?? ['reason' => $json];

        if ($code === 2) {
            self::rmDir($outDir); // file kotor: buang output
            throw new InfectedFileException($report['reason'] ?? 'blocked', $report);
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

    /** Hapus folder tmp output setelah file dipindah ke storage permanen. */
    public static function cleanup(string $dir): void
    {
        self::rmDir($dir);
    }

    private static function rmDir(string $dir): void
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
        // 1. env override, 2. cache dir, 3. vendor/bin fallback
        if ($env = getenv('GUARDCOMPRESS_BIN')) return $env;
        $os = strtolower(PHP_OS_FAMILY); // linux, windows, darwin
        $arch = php_uname('m');
        $arch = str_contains($arch, 'arm') || str_contains($arch, 'aarch64') ? 'arm64' : 'amd64';
        $ext = $os === 'windows' ? '.exe' : '';
        $name = "guardcompress-{$os}-{$arch}{$ext}";
        // AUDIT: HOME bisa kosong di PHP-FPM; coba getenv + USERPROFILE juga.
        $home = $_SERVER['HOME'] ?? getenv('HOME') ?? getenv('USERPROFILE') ?: sys_get_temp_dir();
        foreach([
            "$home/.cache/guardcompress/$name",
            __DIR__ . "/../../core/bin/$name",
            __DIR__ . "/../../bin/$name",
        ] as $p) { if (is_file($p)) return $p; }
        // belum di-download -> arahkan ke installer
        throw new \RuntimeException(
            "guardcompress binary not found ($name). Run: php bin/install-binary.php"
        );
    }
}
