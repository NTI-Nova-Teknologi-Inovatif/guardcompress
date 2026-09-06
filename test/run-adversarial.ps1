# Uji adversarial: jalankan semua file attack vs harapan, laporkan PASS/FAIL.
# Usage dari root repo:  powershell -File test/run-adversarial.ps1
# Exit 0 = semua sesuai harapan. Butuh binary di core/bin/.
$ErrorActionPreference = 'Stop'
$root = Split-Path -Parent (Split-Path -Parent $PSCommandPath)
$arch = if ([System.Environment]::Is64BitOperatingSystem) { 'amd64' } else { 'amd64' }
$bin = Join-Path $root 'core\bin\guardcompress-windows-amd64.exe'
if (-not (Test-Path $bin)) { Write-Error "binary tidak ada: $bin. Build dulu: cd core; go build -o bin/guardcompress-windows-amd64.exe ."; exit 1 }

$expected = Get-Content (Join-Path $root 'test\adversarial\expected.json') -Raw | ConvertFrom-Json
$pass = 0; $fail = 0
foreach ($prop in $expected.PSObject.Properties) {
    $name = $prop.Name; $want = $prop.Value
    $in = Join-Path $root ("test\adversarial\" + $name)
    $out = Join-Path ([System.IO.Path]::GetTempPath()) ("gcadv-" + [System.IO.Path]::GetRandomFileName())
    $json = & $bin check --in $in --out-dir $out --json 2>$null | Select-Object -Last 1 | ConvertFrom-Json
    $got = $json.status
    Remove-Item -Recurse -Force $out -ErrorAction SilentlyContinue
    if ($got -eq $want) { $pass++; Write-Host "PASS  $name -> $got" }
    else { $fail++; Write-Host "FAIL  $name -> dapat $got, harap $want ($($json.reason))" -ForegroundColor Red }
}
Write-Host "`n$pass PASS, $fail FAIL dari $($pass + $fail) kasus"
exit $fail
