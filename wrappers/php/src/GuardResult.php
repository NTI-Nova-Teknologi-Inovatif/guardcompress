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

class GuardException extends \RuntimeException
{
    public function __construct(string $msg, public readonly array $report = [], int $code = 0, ?\Throwable $prev = null)
    { parent::__construct($msg, $code, $prev); }
}

class InfectedFileException extends GuardException {}
