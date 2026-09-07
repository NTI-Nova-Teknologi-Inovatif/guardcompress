<?php
declare(strict_types=1);

namespace GuardCompress;

class GuardCompress
{
    // Satu sistem fleksibel: process() umum + preset per jenis.
    // Contoh: GuardCompress::image($path)  // khusus jpg/png/gif/webp
    //         GuardCompress::video($path, ['video_crf' => 30])
    //         GuardCompress::audio($path)
    public static function image(string $inPath, array $opts = []): GuardResult
    {
        // $opts menang bila user menimpa allow_ext sendiri.
        return self::process($inPath, $opts + ['allow_ext' => ['jpg', 'jpeg', 'png', 'webp', 'gif']]);
    }

    public static function video(string $inPath, array $opts = []): GuardResult
    {
        return self::process($inPath, $opts + ['allow_ext' => ['mp4', 'mov', 'webm', 'mkv', 'avi']]);
    }

    public static function audio(string $inPath, array $opts = []): GuardResult
    {
        return self::process($inPath, $opts + ['allow_ext' => ['mp3', 'wav', 'ogg', 'oga', 'm4a', 'flac']]);
    }

    // Batch multi-input beda jenis sekaligus.
    // $items: ['avatar' => '/tmp/a.png', 'video' => '/tmp/b.mp4'] atau
    //         [['path' => ..., 'opts' => [...]], ...].
    // Tak pernah lempar untuk file DITOLAK (terkumpul per item); error teknis
    // (binary hilang) tetap dilempar langsung (fail-fast).
    // Return: ['avatar' => ['ok' => true, 'result' => GuardResult],
    //          'video'  => ['ok' => false, 'blocked' => true, 'reason' => ...]]
    public static function batch(array $items, array $opts = []): array
    {
        $out = [];
        foreach ($items as $key => $item) {
            $path = is_array($item) ? ($item['path'] ?? '') : $item;
            $iopts = $opts;
            if (is_array($item) && isset($item['opts']) && is_array($item['opts'])) {
                $iopts = $item['opts'] + $opts;
            }
            try {
                $out[$key] = ['ok' => true, 'result' => self::process((string)$path, $iopts)];
            } catch (InfectedFileException $e) {
                $out[$key] = ['ok' => false, 'blocked' => true, 'reason' => $e->getMessage(), 'report' => $e->report];
            }
        }
        return $out;
    }

    // batchParallel: batch multi-input yang jalan BERSAMAAN (jobs proses,
    // default 4, maks 16). Urutan hasil = urutan input. File ditolak
    // terkumpul; error teknis melempar langsung. Butuh proc_open().
    public static function batchParallel(array $items, array $opts = []): array
    {
        if (!function_exists('proc_open')) {
            throw new \RuntimeException('proc_open() dibutuhkan GuardCompress');
        }
        $bin = self::resolveBinary();
        $jobs = max(1, min((int)($opts['jobs'] ?? 4), 16));
        unset($opts['jobs']);
        $keys = array_keys($items);
        $out = [];
        $running = []; // idx => ['proc'=>, 'pipes'=>, 'outDir'=>, 'buf'=>]
        $next = 0;
        $n = count($keys);
        $startOne = function ($idx, $item) use ($bin, $opts, &$running) {
            $path = is_array($item) ? ($item['path'] ?? '') : $item;
            $iopts = $opts;
            if (is_array($item) && isset($item['opts']) && is_array($item['opts'])) {
                $iopts = $item['opts'] + $opts;
            }
            try {
                $suffix = bin2hex(random_bytes(8));
            } catch (\Throwable) {
                $suffix = uniqid('', true);
            }
            $outDir = sys_get_temp_dir() . '/gc-' . $suffix;
            mkdir($outDir, 0700, true);
            $proc = proc_open(
                [$bin, 'check', '--in', (string)$path, '--out-dir', $outDir,
                 '--config', json_encode($iopts ?: new \stdClass()), '--json'],
                [1 => ['pipe', 'w'], 2 => ['pipe', 'w']],
                $pipes
            );
            if (!is_resource($proc)) {
                self::rmDir($outDir);
                throw new GuardException('gagal menjalankan guardcompress binary');
            }
            stream_set_blocking($pipes[1], false);
            $running[$idx] = ['proc' => $proc, 'pipes' => $pipes, 'outDir' => $outDir, 'buf' => ''];
        };
        $finishOne = function ($idx) use (&$running, &$out, $keys) {
            $h = $running[$idx];
            unset($running[$idx]);
            $code = proc_close($h['proc']);
            $json = trim($h['buf']);
            $nl = strrpos($json, "\n");
            $report = json_decode($nl === false ? $json : substr($json, $nl + 1), true) ?? ['reason' => $json];
            $key = $keys[$idx];
            if ($code === 2) {
                self::rmDir($h['outDir']);
                $out[$key] = ['ok' => false, 'blocked' => true,
                    'reason' => $report['reason'] ?? 'blocked', 'report' => $report];
            } elseif (!empty($report['details']['busy'])) {
                self::rmDir($h['outDir']);
                throw new BusyException($report['reason'] ?? 'server busy', $report);
            } elseif ($code !== 0 || empty($report['out_path']) || !is_file($report['out_path'])) {
                self::rmDir($h['outDir']);
                throw new GuardException($report['reason'] ?? 'guardcompress failed', $report);
            } else {
                $out[$key] = ['ok' => true, 'result' => new GuardResult($report['out_path'], $report)];
            }
        };
        while ($next < $n || $running) {
            while ($next < $n && count($running) < $jobs) {
                $startOne($next, $items[$keys[$next]]);
                $next++;
            }
            $read = [];
            foreach ($running as $h) {
                $read[] = $h['pipes'][1];
            }
            $w = $e = null;
            if (@stream_select($read, $w, $e, 5)) {
                foreach ($read as $s) {
                    foreach ($running as $idx => $h) {
                        if ($h['pipes'][1] === $s) {
                            $chunk = fread($s, 8192);
                            if ($chunk !== false && $chunk !== '') {
                                $running[$idx]['buf'] .= $chunk;
                            }
                        }
                    }
                }
            }
            foreach (array_keys($running) as $idx) {
                $st = proc_get_status($running[$idx]['proc']);
                if (!$st['running']) {
                    // Kuras sisa output lalu selesaikan.
                    $s = $running[$idx]['pipes'][1];
                    while (($chunk = fread($s, 8192)) !== false && $chunk !== '') {
                        $running[$idx]['buf'] .= $chunk;
                    }
                    fclose($s);
                    fclose($running[$idx]['pipes'][2]);
                    $finishOne($idx);
                }
            }
        }
        // Kembalikan urutan input.
        $ordered = [];
        foreach ($keys as $key) {
            $ordered[$key] = $out[$key];
        }
        return $ordered;
    }

    public static function process(string $inPath, array $opts = []): GuardResult
    {
        $bin = self::resolveBinary();
        // Suffix acak kriptografis. uniqid gampang ditebak, jangan dipakai
        // buat nama folder tmp.
        try {
            $suffix = bin2hex(random_bytes(8));
        } catch (\Throwable) {
            $suffix = uniqid('', true);
        }
        $outDir = sys_get_temp_dir() . '/gc-' . $suffix;
        // 0700, bukan 0777: user lain di shared hosting jangan bisa intip.
        if (!mkdir($outDir, 0700, true) && !is_dir($outDir)) {
            throw new \RuntimeException("cannot create tmp dir: $outDir");
        }
        // proc_open array = tanpa shell. escapeshellarg+exec pecah di
        // Windows kalau JSON-nya ada kutip (cmd.exe mengupasnya).
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
        // Sinyal busy (backpressure): server penuh, minta retry (HTTP 429).
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
        // HOME sering kosong di PHP-FPM, cek getenv + USERPROFILE juga.
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
