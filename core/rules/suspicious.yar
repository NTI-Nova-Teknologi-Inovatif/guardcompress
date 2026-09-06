rule Suspicious_Webshell {
  meta:
    description = "Cerminan token heuristic core (guard.go). v2: file ini di-embed via go:embed + yara-x."
    author = "GuardCompress"
  strings:
    $php_tag = "<?php"                    // wajib lowercase di engine PHP
    $php_short = "<?="
    $asp_tag = "<%"
    $script = "<script" nocase            // HTML case-insensitive
    $eval = "eval(" nocase                // konstruksi PHP case-insensitive
    $assert = "assert(" nocase
    $b64 = "base64_decode" nocase         // fungsi PHP case-insensitive
    $rot13 = "str_rot13" nocase
    $inflate = "gzinflate" nocase
    $createfn = "create_function" nocase
    $shellexec = "shell_exec" nocase
    $passthru = "passthru" nocase
    $popen = "popen(" nocase
    $procpopen = "proc_open(" nocase
    $c99 = "c99shell"
    $r57 = "r57shell"
    $cmd = "cmd.exe" nocase
    $sh = "/bin/sh"
    $eicar = "X5O!P%@AP[4\\PZX54(P^)7CC)7}$EICAR"
  condition:
    any of them
}
