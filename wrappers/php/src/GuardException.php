<?php
declare(strict_types=1);

namespace GuardCompress;

class GuardException extends \RuntimeException
{
    public function __construct(string $msg, public readonly array $report = [], int $code = 0, ?\Throwable $prev = null)
    { parent::__construct($msg, $code, $prev); }
}
