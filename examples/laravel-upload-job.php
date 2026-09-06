<?php
// Contoh Laravel Queue Job: upload besar jangan block request.
// php artisan make:job CompressUpload -> tempel isi handle() ini.
namespace App\Jobs;

use GuardCompress\GuardCompress;
use Illuminate\Bus\Queueable;
use Illuminate\Contracts\Queue\ShouldQueue;
use Illuminate\Support\Facades\Storage;

class CompressUpload implements ShouldQueue
{
    use Queueable;

    public function __construct(public string $tmpPath, public int $userId) {}

    public function handle(): void
    {
        try {
            $r = GuardCompress::process($this->tmpPath, ['max_mb' => 500, 'video_crf' => 28, 'timeoutSec' => 600]);
            $stored = Storage::disk('s3')->putFile('media', new \File($r->path));
            // TODO: simpan $stored + $r->report ke DB, notify user
        } catch (\GuardCompress\InfectedFileException $e) {
            \Log::warning('Upload blocked', ['user' => $this->userId, 'report' => $e->report]);
            // TODO: notify user "file ditolak"
        } finally {
            @unlink($this->tmpPath);
            if (isset($r)) \GuardCompress\GuardCompress::cleanup(dirname($r->path));
        }
    }
}
