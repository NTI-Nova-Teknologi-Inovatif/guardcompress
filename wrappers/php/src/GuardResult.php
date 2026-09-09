<?php
declare(strict_types=1);

namespace GuardCompress;

class GuardResult
{
    public function __construct(
        public readonly string $path,
        public readonly array $report = []
    ) {}
    public function getOrigBytes(): int { return (int)($this->report['orig_bytes'] ?? 0); }
    public function getNewBytes(): int { return (int)($this->report['new_bytes'] ?? 0); }
}
