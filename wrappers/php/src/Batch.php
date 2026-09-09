<?php
declare(strict_types=1);

namespace GuardCompress;

class Batch
{
    public static function run(array $items, array $opts = []): array
    {
        $out = [];
        foreach ($items as $key => $item) {
            $path = is_array($item) ? ($item['path'] ?? '') : $item;
            $iopts = $opts;
            if (is_array($item) && isset($item['opts']) && is_array($item['opts'])) {
                $iopts = $item['opts'] + $opts;
            }
            try {
                $out[$key] = ['ok' => true, 'result' => Client::process((string)$path, $iopts)];
            } catch (InfectedFileException $e) {
                $out[$key] = ['ok' => false, 'blocked' => true, 'reason' => $e->getMessage(), 'report' => $e->report];
            }
        }
        return $out;
    }

    public static function runParallel(array $items, array $opts = []): array
    {
        if (!function_exists('proc_open')) {
            throw new \RuntimeException('proc_open() dibutuhkan GuardCompress');
        }
        $bin = Client::resolveBinary();
        $jobs = max(1, min((int)($opts['jobs'] ?? 4), 16));
        unset($opts['jobs']);
        $keys = array_keys($items);
        $out = [];
        $running = [];
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
                Client::rmDir($outDir);
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
                Client::rmDir($h['outDir']);
                $out[$key] = ['ok' => false, 'blocked' => true,
                    'reason' => $report['reason'] ?? 'blocked', 'report' => $report];
            } elseif (!empty($report['details']['busy'])) {
                Client::rmDir($h['outDir']);
                throw new BusyException($report['reason'] ?? 'server busy', $report);
            } elseif ($code !== 0 || empty($report['out_path']) || !is_file($report['out_path'])) {
                Client::rmDir($h['outDir']);
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
        $ordered = [];
        foreach ($keys as $key) {
            $ordered[$key] = $out[$key];
        }
        return $ordered;
    }
}
