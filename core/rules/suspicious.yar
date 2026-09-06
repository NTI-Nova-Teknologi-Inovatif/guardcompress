rule Suspicious_Webshell {
  meta:
    description = "Placeholder YARA rule - v2 akan di-embed via go:embed + yara-x"
    author = "GuardCompress"
  strings:
    $php_tag = "<?php"
    $asp_tag = "<%"
    $eval = "eval(" nocase
    $b64 = "base64_decode" nocase
    $c99 = "c99shell" nocase
    $eicar = "X5O!P%@AP[4\\PZX54(P^)7CC)7}$EICAR"
  condition:
    any of them
}
