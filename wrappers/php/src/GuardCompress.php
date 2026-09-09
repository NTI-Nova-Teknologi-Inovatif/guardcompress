<?php
declare(strict_types=1);

namespace GuardCompress;

class GuardCompress
{
    public static function process(string $inPath, array $opts = []): GuardResult
    {
        return Client::process($inPath, $opts);
    }

    public static function image(string $inPath, array $opts = []): GuardResult
    {
        return Client::process($inPath, $opts + ['allow_ext' => ['jpg', 'jpeg', 'png', 'webp', 'gif']]);
    }

    public static function video(string $inPath, array $opts = []): GuardResult
    {
        return Client::process($inPath, $opts + ['allow_ext' => ['mp4', 'mov', 'webm', 'mkv', 'avi']]);
    }

    public static function audio(string $inPath, array $opts = []): GuardResult
    {
        return Client::process($inPath, $opts + ['allow_ext' => ['mp3', 'wav', 'ogg', 'oga', 'm4a', 'flac']]);
    }

    public static function batch(array $items, array $opts = []): array
    {
        return Batch::run($items, $opts);
    }

    public static function batchParallel(array $items, array $opts = []): array
    {
        return Batch::runParallel($items, $opts);
    }

    public static function cleanup(string $dir): void
    {
        Client::cleanup($dir);
    }
}
